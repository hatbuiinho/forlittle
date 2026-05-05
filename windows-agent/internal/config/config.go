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
	ChromeArgs           []string `json:"chrome_args"`
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
