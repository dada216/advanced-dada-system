package meta

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/advanced-dada-system/ads/internal/config"
	_ "github.com/mattn/go-sqlite3"
)

const schema = `
CREATE TABLE IF NOT EXISTS sessions (
	uuid   TEXT PRIMARY KEY,
	name   TEXT UNIQUE NOT NULL,
	type   TEXT NOT NULL DEFAULT 'local',
	status TEXT NOT NULL DEFAULT 'created',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	remote_user TEXT,
	remote_host TEXT,
	remote_port INTEGER,
	profile_name TEXT NOT NULL DEFAULT 'default'
);

CREATE TABLE IF NOT EXISTS tmux_profiles (
	id INTEGER PRIMARY KEY,
	name TEXT UNIQUE NOT NULL,
	config_text TEXT NOT NULL
);
`

const defaultProfile = `
# ADS Managed Tmux Configuration
# Enable massive scrollback and mouse support
set-option -g history-limit 100000
set -g mouse on

# Auto-enter copy-mode (Ctrl-B [) seamlessly when scrolling the mouse wheel
bind-key -n WheelUpPane if-shell -F -t = "#{mouse_any_flag}" "send-keys -M" "if -Ft= '#{pane_in_mode}' 'send-keys -M' 'copy-mode -e; send-keys -M'"
bind-key -n WheelDownPane if-shell -F -t = "#{mouse_any_flag}" "send-keys -M" "if -Ft= '#{pane_in_mode}' 'send-keys -M' ''"

# Provide native interactive federated search UI via floating popup
bind-key -n C-s display-popup -E -w 90% -h 90% "{{.AdsBinaryPath}} search-interactive"

# Global paste shortcut without prefix
bind-key -n C-] paste-buffer

# Optional: Source user's local config if it exists
if-shell "test -f ~/.tmux.conf" "source-file ~/.tmux.conf"
`

type DB struct {
	db *sql.DB
}

func Open() (*DB, error) {
	appDir, err := config.InitAppDataDir()
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(appDir, "meta.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open meta database: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// v0.2 Migrations (ignore errors if columns already exist)
	_, _ = db.Exec(`ALTER TABLE sessions ADD COLUMN remote_user TEXT;`)
	_, _ = db.Exec(`ALTER TABLE sessions ADD COLUMN remote_host TEXT;`)
	_, _ = db.Exec(`ALTER TABLE sessions ADD COLUMN remote_port INTEGER;`)

	// v0.4 Migrations
	_, _ = db.Exec(`ALTER TABLE sessions ADD COLUMN profile_name TEXT NOT NULL DEFAULT 'default';`)

	// Seed default profile
	_, _ = db.Exec(`INSERT OR IGNORE INTO tmux_profiles (name, config_text) VALUES ('default', ?)`, defaultProfile)

	// Force update the default profile to ensure existing users get the new scrollback bindings
	_, _ = db.Exec(`UPDATE tmux_profiles SET config_text = ? WHERE name = 'default'`, defaultProfile)

	// v2.0 Migrations
	// Ignore errors if columns already exist
	_, _ = db.Exec(`ALTER TABLE tmux_profiles ADD COLUMN customer TEXT;`)
	_, _ = db.Exec(`ALTER TABLE tmux_profiles ADD COLUMN server TEXT;`)
	_, _ = db.Exec(`ALTER TABLE tmux_profiles ADD COLUMN project TEXT;`)

	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}
