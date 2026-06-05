package config

import (
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
