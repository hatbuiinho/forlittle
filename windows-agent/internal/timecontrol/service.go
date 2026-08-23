package timecontrol

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"forlittle/windows-agent/internal/config"
)

type Publisher interface{ Publish(StateMessage) }

// enrollmentVersion forces one safe re-enrollment when the device identity
// format changes. Version zero represents credentials created by older agents.
const enrollmentVersion = 2

type Service struct {
	cfg                   config.TimeControlConfig
	store                 *Store
	client                *Client
	publisher             Publisher
	logger                *log.Logger
	state                 PersistedState
	lastAgentHeartbeat    time.Time
	lastAgentStartAttempt time.Time
	mu                    sync.Mutex
}

func NewService(cfg config.TimeControlConfig, publisher Publisher, logger *log.Logger) *Service {
	return &Service{cfg: cfg, store: NewStore(cfg.DataDir), publisher: publisher, logger: logger}
}

func (s *Service) Run(ctx context.Context) error {
	state, err := s.store.Load()
	if err != nil {
		return err
	}
	s.state = state
	credentials, err := loadCredentials(s.cfg.DataDir)
	if err != nil {
		return err
	}
	deviceToken := credentials.DeviceToken
	if credentials.EnrollmentVersion < enrollmentVersion {
		deviceToken = ""
	}
	s.client = NewClient(s.cfg, deviceToken)
	if !s.ensureEnrollment(ctx) {
		s.logger.Printf("initial enrollment unavailable; enforcing cached state")
	}

	s.recalculate(time.Now().UTC())
	go s.ensureAgent(ctx)
	s.syncPolicy(ctx)
	s.syncCommands(ctx)

	policyTicker := time.NewTicker(time.Duration(s.cfg.PolicyPollSeconds) * time.Second)
	commandTicker := time.NewTicker(time.Duration(s.cfg.CommandPollSeconds) * time.Second)
	heartbeatTicker := time.NewTicker(time.Duration(s.cfg.HeartbeatSeconds) * time.Second)
	evaluateTicker := time.NewTicker(15 * time.Second)
	agentTicker := time.NewTicker(15 * time.Second)
	usageTicker := time.NewTicker(time.Minute)
	defer policyTicker.Stop()
	defer commandTicker.Stop()
	defer heartbeatTicker.Stop()
	defer evaluateTicker.Stop()
	defer agentTicker.Stop()
	defer usageTicker.Stop()
	notifications := s.client.Notifications(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-evaluateTicker.C:
			s.recalculate(time.Now().UTC())
		case <-policyTicker.C:
			s.syncPolicy(ctx)
		case <-commandTicker.C:
			s.syncCommands(ctx)
		case <-notifications:
			s.syncCommands(ctx)
		case <-heartbeatTicker.C:
			s.sendHeartbeat(ctx)
		case <-agentTicker.C:
			s.ensureAgent(ctx)
		case <-usageTicker.C:
			s.flushUsage(ctx)
		}
	}
}

func (s *Service) ensureAgent(ctx context.Context) {
	s.mu.Lock()
	healthy := !s.lastAgentHeartbeat.IsZero() && time.Since(s.lastAgentHeartbeat) < 15*time.Second
	tooSoon := !s.lastAgentStartAttempt.IsZero() && time.Since(s.lastAgentStartAttempt) < 30*time.Second
	if !healthy && !tooSoon {
		s.lastAgentStartAttempt = time.Now().UTC()
	}
	s.mu.Unlock()
	if healthy || tooSoon {
		return
	}
	if err := restartAgent(ctx, s.cfg.AgentPath); err != nil {
		s.logger.Printf("agent restart request failed: %v", err)
		return
	}
	s.logger.Printf("started UI agent %q", s.cfg.AgentPath)
}

func credentialsFromToken(token string) credentials {
	return credentials{DeviceToken: token, EnrollmentVersion: enrollmentVersion}
}

func (s *Service) syncPolicy(ctx context.Context) {
	if !s.ensureEnrollment(ctx) {
		return
	}
	policy, serverTime, err := s.client.FetchPolicy(ctx)
	if err != nil {
		s.logger.Printf("policy sync failed: %v", err)
		return
	}
	s.mu.Lock()
	s.state.Policy = policy
	s.setServerTime(serverTime)
	_ = s.saveLocked()
	s.mu.Unlock()
	s.recalculate(s.trustedNow())
}

func (s *Service) syncCommands(ctx context.Context) {
	if !s.ensureEnrollment(ctx) {
		return
	}
	commands, serverTime, err := s.client.FetchCommands(ctx)
	if err != nil {
		s.logger.Printf("command sync failed: %v", err)
		return
	}
	s.mu.Lock()
	s.setServerTime(serverTime)
	s.mu.Unlock()
	for _, command := range commands {
		s.applyCommand(ctx, command)
	}
}

func (s *Service) applyCommand(ctx context.Context, command Command) {
	s.mu.Lock()
	if _, exists := s.state.AppliedCommandIDs[command.CommandID]; exists {
		s.mu.Unlock()
		_ = s.client.Ack(ctx, command.CommandID, "IGNORED_DUPLICATE", "already applied")
		return
	}
	s.mu.Unlock()
	_ = s.client.Ack(ctx, command.CommandID, "RECEIVED", "")

	var payload struct {
		DurationSeconds int64 `json:"duration_seconds"`
	}
	if err := json.Unmarshal([]byte(command.PayloadJSON), &payload); err != nil {
		_ = s.client.Ack(ctx, command.CommandID, "FAILED", "invalid payload")
		return
	}
	now := s.trustedNow()
	s.mu.Lock()
	success := true
	switch command.Type {
	case "BLOCK":
		s.state.Override.ForceBlocked = true
	case "UNBLOCK":
		if payload.DurationSeconds <= 0 {
			success = false
			break
		}
		s.state.Override.ForceBlocked = false
		until := now.Add(time.Duration(payload.DurationSeconds) * time.Second)
		s.state.Override.ManualUnlockUntil = &until
	case "EXTRA_TIME":
		if payload.DurationSeconds <= 0 {
			success = false
			break
		}
		until := now.Add(time.Duration(payload.DurationSeconds) * time.Second)
		s.state.Override.ExtendedUntil = &until
	case "REFRESH_POLICY", "POLICY_UPDATED":
		// The next sync occurs immediately after releasing the lock.
	case "FORCE_LOCK", "FORCE_LOGOUT":
		if s.publisher != nil {
			s.publisher.Publish(StateMessage{Type: command.Type})
		}
	default:
		success = false
	}
	if success {
		s.state.AppliedCommandIDs[command.CommandID] = now
		s.pruneCommandsLocked(now)
		_ = s.saveLocked()
	}
	s.mu.Unlock()
	if !success {
		_ = s.client.Ack(ctx, command.CommandID, "FAILED", "unsupported command")
		return
	}
	s.recalculate(now)
	if command.Type == "REFRESH_POLICY" || command.Type == "POLICY_UPDATED" {
		s.syncPolicy(ctx)
	}
	_ = s.client.Ack(ctx, command.CommandID, "APPLIED", "")
}

func (s *Service) recalculate(now time.Time) {
	s.mu.Lock()
	effective := Evaluate(now, s.state.Policy, s.state.Override)
	changed := !sameEffectiveState(effective, s.state.Effective)
	s.state.Effective = effective
	_ = s.saveLocked()
	s.mu.Unlock()
	if changed && s.publisher != nil {
		s.publisher.Publish(StateMessage{Type: "STATE_CHANGED", State: effective.State, Reason: effective.Reason, NextAllowedAt: effective.NextAllowedAt, ExtendedUntil: effective.ExtendedUntil})
	}
}

func sameEffectiveState(left, right EffectiveState) bool {
	return left.State == right.State &&
		left.Reason == right.Reason &&
		sameOptionalTime(left.NextAllowedAt, right.NextAllowedAt) &&
		sameOptionalTime(left.ExtendedUntil, right.ExtendedUntil)
}

func sameOptionalTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func (s *Service) sendHeartbeat(ctx context.Context) {
	if !s.ensureEnrollment(ctx) {
		return
	}
	s.mu.Lock()
	effective := s.state.Effective
	s.mu.Unlock()
	serverTime, err := s.client.Heartbeat(ctx, effective, s.AgentHealthy())
	if err != nil {
		s.logger.Printf("heartbeat failed: %v", err)
		return
	}
	s.mu.Lock()
	s.setServerTime(serverTime)
	_ = s.saveLocked()
	s.mu.Unlock()
}

func (s *Service) HandleAgentMessage(message AgentMessage) {
	if message.Type == "AGENT_HEARTBEAT" {
		s.mu.Lock()
		s.lastAgentHeartbeat = time.Now().UTC()
		s.mu.Unlock()
		return
	}
	if message.Type != "USAGE_SAMPLE" || message.WindowsUser == "" || message.Application == "" || message.ActiveSeconds < 0 || message.IdleSeconds < 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Effective.State == StateBlocked {
		return
	}
	now := s.trustedNowLocked()
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	key := date.Format("2006-01-02") + "|" + message.WindowsUser + "|" + message.Application
	bucket := s.state.UsageBuckets[key]
	bucket.WindowsUser, bucket.Application, bucket.UsageDate = message.WindowsUser, message.Application, date
	bucket.ActiveSeconds += message.ActiveSeconds
	bucket.IdleSeconds += message.IdleSeconds
	s.state.UsageBuckets[key] = bucket
	_ = s.saveLocked()
}

func (s *Service) AgentHealthy() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.lastAgentHeartbeat.IsZero() && time.Since(s.lastAgentHeartbeat) < 15*time.Second
}

func (s *Service) ensureEnrollment(ctx context.Context) bool {
	if s.client.token != "" {
		return true
	}
	token, err := s.client.Enroll(ctx)
	if err != nil {
		s.logger.Printf("service enrollment failed: %v", err)
		return false
	}
	if err := saveCredentials(s.cfg.DataDir, credentialsFromToken(token)); err != nil {
		s.logger.Printf("could not persist service credentials: %v", err)
		return false
	}
	return true
}

func (s *Service) CurrentMessage() StateMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	effective := s.state.Effective
	return StateMessage{Type: "STATE_CHANGED", State: effective.State, Reason: effective.Reason, NextAllowedAt: effective.NextAllowedAt, ExtendedUntil: effective.ExtendedUntil}
}

func (s *Service) trustedNow() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.trustedNowLocked()
}

func (s *Service) trustedNowLocked() time.Time {
	return time.Now().UTC().Add(time.Duration(s.state.ServerTimeOffsetMs) * time.Millisecond)
}

func (s *Service) flushUsage(ctx context.Context) {
	if !s.ensureEnrollment(ctx) {
		return
	}
	s.mu.Lock()
	buckets := make([]UsageBucket, 0, len(s.state.UsageBuckets))
	for _, bucket := range s.state.UsageBuckets {
		buckets = append(buckets, bucket)
	}
	s.mu.Unlock()
	if err := s.client.SendUsage(ctx, buckets); err != nil {
		s.logger.Printf("usage sync failed: %v", err)
	}
}

func (s *Service) setServerTime(serverTime time.Time) {
	if serverTime.IsZero() {
		return
	}
	s.state.ServerTimeOffsetMs = serverTime.Sub(time.Now().UTC()).Milliseconds()
}

func (s *Service) saveLocked() error { return s.store.Save(s.state) }

func (s *Service) pruneCommandsLocked(now time.Time) {
	for id, appliedAt := range s.state.AppliedCommandIDs {
		if now.Sub(appliedAt) > 30*24*time.Hour {
			delete(s.state.AppliedCommandIDs, id)
		}
	}
}
