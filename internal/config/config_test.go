package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigOperations(t *testing.T) {
	// Setup temporary XDG_DATA_HOME
	tempDir, err := os.MkdirTemp("", "ads-config-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", oldXDG)
	os.Setenv("XDG_DATA_HOME", tempDir)

	// Test InitAppDataDir
	appDir, err := InitAppDataDir()
	if err != nil {
		t.Fatalf("InitAppDataDir failed: %v", err)
	}
	expectedAppDir := filepath.Join(tempDir, "ads")
	if appDir != expectedAppDir {
		t.Errorf("expected %s, got %s", expectedAppDir, appDir)
	}

	// Verify directory creation
	info, err := os.Stat(appDir)
	if err != nil || !info.IsDir() {
		t.Errorf("InitAppDataDir did not create directory properly")
	}

	// Test GetConfigPath
	configPath, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath failed: %v", err)
	}
	if configPath != filepath.Join(expectedAppDir, "config.json") {
		t.Errorf("unexpected config path: %s", configPath)
	}

	// Test LoadConfig (no file exists yet)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed when file missing: %v", err)
	}
	if cfg.DefaultShell != "" {
		t.Errorf("expected empty DefaultShell, got %s", cfg.DefaultShell)
	}

	// Test SaveConfig
	cfg.DefaultShell = "zsh"
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file creation
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("SaveConfig did not create config.json")
	}

	// Test LoadConfig (file exists)
	loadedCfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed when file exists: %v", err)
	}
	if loadedCfg.DefaultShell != "zsh" {
		t.Errorf("expected 'zsh', got '%s'", loadedCfg.DefaultShell)
	}
}
