package timecontrol

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(dataDir string) *Store {
	return &Store{path: filepath.Join(dataDir, "time-control-state.json")}
}

func (s *Store) Load() (PersistedState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return PersistedState{AppliedCommandIDs: make(map[string]time.Time), UsageBuckets: make(map[string]UsageBucket)}, nil
	}
	if err != nil {
		return PersistedState{}, fmt.Errorf("read state: %w", err)
	}
	var state PersistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return PersistedState{}, fmt.Errorf("decode state: %w", err)
	}
	if state.AppliedCommandIDs == nil {
		state.AppliedCommandIDs = make(map[string]time.Time)
	}
	if state.UsageBuckets == nil {
		state.UsageBuckets = make(map[string]UsageBucket)
	}
	return state, nil
}

func (s *Store) Save(state PersistedState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	state.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	temporary := s.path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	if err := os.Rename(temporary, s.path); err != nil {
		return fmt.Errorf("commit state: %w", err)
	}
	return nil
}
