package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	ChromePath           string   `json:"chrome_path"`
	ExtensionPath        string   `json:"extension_path"`
	ProfilePath          string   `json:"profile_path"`
	RelaunchDelaySeconds int      `json:"relaunch_delay_seconds"`
	ScanIntervalSeconds  int      `json:"scan_interval_seconds"`
	KillUnmanagedChrome  bool     `json:"kill_unmanaged_chrome"`
	StrictExtensionOnly  bool     `json:"strict_extension_only"`
	ForceRestartOnStart  bool     `json:"force_restart_on_start"`
	ChromeLogPath        string   `json:"chrome_log_path"`
	StartupURLs          []string `json:"startup_urls"`
	ChromeArgs           []string `json:"chrome_args"`
}

// TimeControlConfig is intentionally separate from the legacy Chrome runner
// configuration. The production service never launches Chrome itself.
type TimeControlConfig struct {
	ServerURL             string `json:"server_url"`
	MachineID             string `json:"machine_id"`
	DisplayName           string `json:"display_name"`
	LittleMonkCode        string `json:"little_monk_code"`
	LittleMonkDisplayName string `json:"little_monk_display_name"`
	EnrollmentKey         string `json:"enrollment_key"`
	DataDir               string `json:"data_dir"`
	PolicyPollSeconds     int    `json:"policy_poll_seconds"`
	CommandPollSeconds    int    `json:"command_poll_seconds"`
	HeartbeatSeconds      int    `json:"heartbeat_seconds"`
	AgentPath             string `json:"agent_path"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	cfg.applyDefaults()
	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.RelaunchDelaySeconds <= 0 {
		c.RelaunchDelaySeconds = 2
	}

	if c.ScanIntervalSeconds <= 0 {
		c.ScanIntervalSeconds = 3
	}
}

func (c Config) Validate() error {
	if c.ChromePath == "" {
		return errors.New("chrome_path is required")
	}

	if c.ExtensionPath == "" {
		return errors.New("extension_path is required")
	}

	if c.ProfilePath == "" {
		return errors.New("profile_path is required")
	}

	manifestPath := filepath.Join(c.ExtensionPath, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		return fmt.Errorf("extension_path must point to an unpacked extension folder containing manifest.json: %w", err)
	}

	if c.RelaunchDelaySeconds < 1 {
		return fmt.Errorf("relaunch_delay_seconds must be >= 1")
	}

	if c.ScanIntervalSeconds < 1 {
		return fmt.Errorf("scan_interval_seconds must be >= 1")
	}

	return nil
}

func (c Config) RelaunchDelay() time.Duration {
	return time.Duration(c.RelaunchDelaySeconds) * time.Second
}

func (c Config) ScanInterval() time.Duration {
	return time.Duration(c.ScanIntervalSeconds) * time.Second
}

func LoadTimeControl(path string) (TimeControlConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return TimeControlConfig{}, err
	}
	var cfg TimeControlConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return TimeControlConfig{}, err
	}
	if cfg.DataDir == "" {
		cfg.DataDir = `C:\ProgramData\ForLittle\TimeControl`
	}
	if cfg.PolicyPollSeconds <= 0 {
		cfg.PolicyPollSeconds = 60
	}
	if cfg.CommandPollSeconds <= 0 {
		cfg.CommandPollSeconds = 10
	}
	if cfg.HeartbeatSeconds <= 0 {
		cfg.HeartbeatSeconds = 30
	}
	if cfg.AgentPath == "" {
		cfg.AgentPath = `C:\Program Files\ForLittle\TimeControl\ForLittle.TimeControl.Agent.exe`
	}
	if cfg.LittleMonkCode == "" {
		cfg.LittleMonkCode = cfg.MachineID
	}
	if cfg.LittleMonkDisplayName == "" {
		cfg.LittleMonkDisplayName = cfg.DisplayName
		if cfg.LittleMonkDisplayName == "" {
			cfg.LittleMonkDisplayName = cfg.MachineID
		}
	}
	if cfg.ServerURL == "" || cfg.MachineID == "" {
		return TimeControlConfig{}, errors.New("server_url and machine_id are required")
	}
	return cfg, nil
}
