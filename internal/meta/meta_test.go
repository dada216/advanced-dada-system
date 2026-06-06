package meta

import (
	"database/sql"
	"os"
	"testing"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	tempDir, err := os.MkdirTemp("", "ads-meta-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tempDir)

	db, err := Open()
	if err != nil {
		t.Fatalf("failed to open meta db: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Setenv("XDG_DATA_HOME", oldXDG)
		os.RemoveAll(tempDir)
	}

	return db, cleanup
}

func TestMetaProfiles(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// default profile should exist
	profiles, err := db.ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles failed: %v", err)
	}
	if len(profiles) != 1 || profiles[0].Name != "default" {
		t.Errorf("expected 1 'default' profile, got: %v", profiles)
	}

	// Create profile
	err = db.CreateProfile("dev")
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}

	// Update metadata
	err = db.UpdateProfileMetadata("dev", sql.NullString{String: "cust", Valid: true}, sql.NullString{String: "srv", Valid: true}, sql.NullString{String: "proj", Valid: true})
	if err != nil {
		t.Fatalf("UpdateProfileMetadata failed: %v", err)
	}

	// Get metadata
	p, err := db.GetProfileMetadata("dev")
	if err != nil {
		t.Fatalf("GetProfileMetadata failed: %v", err)
	}
	if !p.Customer.Valid || p.Customer.String != "cust" || !p.Server.Valid || p.Server.String != "srv" || !p.Project.Valid || p.Project.String != "proj" {
		t.Errorf("Metadata mismatch, got: %v", p)
	}

	// Get config
	cfg, err := db.GetTmuxProfile("dev")
	if err != nil {
		t.Fatalf("GetTmuxProfile failed: %v", err)
	}
	// default config contains "# ADS Managed Tmux Configuration"
	if cfg == "" {
		t.Errorf("expected config content")
	}

	// Update config
	err = db.UpdateProfileConfig("dev", "new config")
	if err != nil {
		t.Fatalf("UpdateProfileConfig failed: %v", err)
	}

	cfg, _ = db.GetTmuxProfile("dev")
	if cfg != "new config" {
		t.Errorf("expected 'new config', got: %s", cfg)
	}
}

func TestMetaSessions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create local session
	uuid, err := db.CreateLocalSession("local-1", "dev")
	if err != nil {
		t.Fatalf("CreateLocalSession failed: %v", err)
	}
	if uuid == "" {
		t.Errorf("expected uuid, got empty string")
	}

	// Create remote session
	remoteUuid, err := db.CreateRemoteSession("remote-1", "user", "host", 22, "dev")
	if err != nil {
		t.Fatalf("CreateRemoteSession failed: %v", err)
	}

	// List sessions
	sessions, err := db.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}

	// Get by name
	sess, err := db.GetSessionByName("local-1")
	if err != nil {
		t.Fatalf("GetSessionByName failed: %v", err)
	}
	if sess.UUID != uuid || sess.Name != "local-1" || sess.Type != "local" {
		t.Errorf("unexpected session data: %+v", sess)
	}

	// Get by UUID
	sess2, err := db.GetSessionByUUID(remoteUuid)
	if err != nil {
		t.Fatalf("GetSessionByUUID failed: %v", err)
	}
	if sess2.Type != "remote" || sess2.RemoteHost != "host" {
		t.Errorf("unexpected remote session data: %+v", sess2)
	}

	// Update status
	err = db.UpdateSessionStatus(uuid, "active")
	if err != nil {
		t.Fatalf("UpdateSessionStatus failed: %v", err)
	}
	sess, _ = db.GetSessionByName("local-1")
	if sess.Status != "active" {
		t.Errorf("expected 'active' status, got '%s'", sess.Status)
	}

	// Rename session
	err = db.RenameSession(uuid, "local-renamed")
	if err != nil {
		t.Fatalf("RenameSession failed: %v", err)
	}
	_, err = db.GetSessionByName("local-1")
	if err == nil {
		t.Errorf("expected error getting old session name")
	}

	// Update profile
	err = db.UpdateSessionProfile(uuid, "default")
	if err != nil {
		t.Fatalf("UpdateSessionProfile failed: %v", err)
	}

	// Delete session
	err = db.DeleteSession(uuid)
	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}
	sessions, _ = db.ListSessions()
	if len(sessions) != 1 {
		t.Errorf("expected 1 session after deletion, got %d", len(sessions))
	}
}

func TestMetaPlugins(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := db.SetPluginConfig("search", "api_key", "secret")
	if err != nil {
		t.Fatalf("SetPluginConfig failed: %v", err)
	}

	val, err := db.GetPluginConfig("search", "api_key")
	if err != nil {
		t.Fatalf("GetPluginConfig failed: %v", err)
	}
	if val != "secret" {
		t.Errorf("expected 'secret', got '%s'", val)
	}

	// Update existing key
	err = db.SetPluginConfig("search", "api_key", "new-secret")
	if err != nil {
		t.Fatalf("SetPluginConfig update failed: %v", err)
	}
	val, _ = db.GetPluginConfig("search", "api_key")
	if val != "new-secret" {
		t.Errorf("expected 'new-secret', got '%s'", val)
	}
}
