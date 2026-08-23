package timecontrol

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type credentials struct {
	DeviceToken string `json:"device_token"`
}

func loadCredentials(dataDir string) (credentials, error) {
	data, err := os.ReadFile(filepath.Join(dataDir, "credentials.json"))
	if os.IsNotExist(err) {
		return credentials{}, nil
	}
	if err != nil {
		return credentials{}, fmt.Errorf("read credentials: %w", err)
	}
	var value credentials
	if err := json.Unmarshal(data, &value); err != nil {
		return credentials{}, fmt.Errorf("decode credentials: %w", err)
	}
	return value, nil
}

func saveCredentials(dataDir string, value credentials) error {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return fmt.Errorf("create credential directory: %w", err)
	}
	path := filepath.Join(dataDir, "credentials.json")
	temporary := path + ".tmp"
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode credentials: %w", err)
	}
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	return os.Rename(temporary, path)
}
