package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTimeControlDefaultsLittleMonkToMachine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"server_url":"https://example.test","machine_id":"PC-001","display_name":"Laptop 01"}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadTimeControl(path)
	if err != nil {
		t.Fatalf("LoadTimeControl returned an error: %v", err)
	}
	if cfg.LittleMonkCode != "PC-001" {
		t.Fatalf("LittleMonkCode = %q, want %q", cfg.LittleMonkCode, "PC-001")
	}
	if cfg.LittleMonkDisplayName != "Laptop 01" {
		t.Fatalf("LittleMonkDisplayName = %q, want %q", cfg.LittleMonkDisplayName, "Laptop 01")
	}
}
