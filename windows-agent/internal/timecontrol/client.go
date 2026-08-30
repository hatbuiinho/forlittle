package timecontrol

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"forlittle/windows-agent/internal/config"

	"github.com/gorilla/websocket"
)

type Client struct {
	baseURL string
	http    *http.Client
	token   string
	cfg     config.TimeControlConfig
}

type policyResponse struct {
	Policy     Policy           `json:"policy"`
	Schedule   []ScheduleWindow `json:"schedule"`
	ServerTime time.Time        `json:"server_time"`
}

func NewClient(cfg config.TimeControlConfig, token string) *Client {
	return &Client{baseURL: strings.TrimRight(cfg.ServerURL, "/"), http: &http.Client{Timeout: 20 * time.Second}, token: token, cfg: cfg}
}

func (c *Client) Enroll(ctx context.Context) (string, error) {
	var response struct {
		DeviceToken string `json:"device_token"`
	}
	err := c.request(ctx, http.MethodPost, "/api/v1/devices/enroll", map[string]string{
		"machine_id":               c.cfg.MachineID,
		"display_name":             c.cfg.DisplayName,
		"little_monk_code":         c.cfg.LittleMonkCode,
		"little_monk_display_name": c.cfg.LittleMonkDisplayName,
		"enrollment_key":           c.cfg.EnrollmentKey,
	}, false, &response)
	if err != nil {
		return "", err
	}
	if response.DeviceToken == "" {
		return "", fmt.Errorf("enrollment returned no device token")
	}
	c.token = response.DeviceToken
	return c.token, nil
}

func (c *Client) FetchPolicy(ctx context.Context) (Policy, time.Time, error) {
	var response policyResponse
	if err := c.request(ctx, http.MethodGet, "/api/v1/devices/time-policy", nil, true, &response); err != nil {
		return Policy{}, time.Time{}, err
	}
	response.Policy.Schedule = response.Schedule
	return response.Policy, response.ServerTime, nil
}

func (c *Client) FetchCommands(ctx context.Context) ([]Command, time.Time, error) {
	var response struct {
		Commands   []Command `json:"commands"`
		ServerTime time.Time `json:"server_time"`
	}
	if err := c.request(ctx, http.MethodGet, "/api/v1/devices/commands", nil, true, &response); err != nil {
		return nil, time.Time{}, err
	}
	return response.Commands, response.ServerTime, nil
}

func (c *Client) Heartbeat(ctx context.Context, state EffectiveState, agentHealthy bool, appliedPolicyVersion int) (time.Time, error) {
	var response struct {
		ServerTime time.Time `json:"server_time"`
	}
	payload := map[string]any{"effective_state": state.State, "state_reason": state.Reason, "next_allowed_at": state.NextAllowedAt, "extended_until": state.ExtendedUntil, "agent_healthy": agentHealthy, "applied_policy_version": appliedPolicyVersion}
	if err := c.request(ctx, http.MethodPost, "/api/v1/devices/heartbeat", payload, true, &response); err != nil {
		return time.Time{}, err
	}
	return response.ServerTime, nil
}

func (c *Client) Ack(ctx context.Context, commandID, status, message string) error {
	return c.request(ctx, http.MethodPost, "/api/v1/devices/commands/"+url.PathEscape(commandID)+"/ack", map[string]string{"status": status, "error": message}, true, nil)
}

func (c *Client) SendUsage(ctx context.Context, buckets []UsageBucket) error {
	if len(buckets) == 0 {
		return nil
	}
	return c.request(ctx, http.MethodPost, "/api/v1/devices/usage", map[string]any{"buckets": buckets}, true, nil)
}

func (c *Client) Notifications(ctx context.Context) <-chan struct{} {
	notifications := make(chan struct{}, 1)
	go func() {
		defer close(notifications)
		backoff := time.Second
		for ctx.Err() == nil {
			wsURL, err := websocketURL(c.baseURL + "/api/v1/devices/ws")
			if err == nil {
				headers := http.Header{"Authorization": []string{"Bearer " + c.token}}
				connection, _, dialErr := websocket.DefaultDialer.DialContext(ctx, wsURL, headers)
				if dialErr == nil {
					backoff = time.Second
					for {
						if _, _, readErr := connection.ReadMessage(); readErr != nil {
							break
						}
						select {
						case notifications <- struct{}{}:
						default:
						}
					}
					_ = connection.Close()
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
		}
	}()
	return notifications
}

func (c *Client) request(ctx context.Context, method, path string, payload any, authenticated bool, output any) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if authenticated {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("server returned %s", response.Status)
	}
	if output == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(output)
}

func websocketURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "https" {
		parsed.Scheme = "wss"
	} else if parsed.Scheme == "http" {
		parsed.Scheme = "ws"
	} else {
		return "", fmt.Errorf("unsupported server url scheme")
	}
	return parsed.String(), nil
}
