package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// InitAppDataDir resolves the XDG data directory and ensures the ads data directory exists.
func InitAppDataDir() (string, error) {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".local", "share")
	}

	adsDir := filepath.Join(dataDir, "ads")
	if err := os.MkdirAll(adsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create ads data directory: %w", err)
	}

	return adsDir, nil
}

type Config struct {
	DefaultShell string `json:"default_shell"`
}

func GetConfigPath() (string, error) {
	dir, err := InitAppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func LoadConfig() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil // return empty config
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
