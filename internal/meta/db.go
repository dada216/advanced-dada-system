package meta

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/advanced-dada-system/ads/internal/config"
	"github.com/google/uuid"
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

# Provide interactive federated search
bind-key s command-prompt -p "ADS Search Query:" "split-window -v -l 20 '{{.AdsBinaryPath}} search \"%%\" | less -R'"

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

	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) CreateLocalSession(name, profile string) (string, error) {
	if profile == "" {
		profile = "default"
	}
	id := uuid.New().String()
	_, err := d.db.Exec(`INSERT INTO sessions (uuid, name, type, status, profile_name) VALUES (?, ?, 'local', 'created', ?)`, id, name, profile)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	return id, nil
}

func (d *DB) CreateRemoteSession(name, remoteUser, remoteHost string, remotePort int, profile string) (string, error) {
	if profile == "" {
		profile = "default"
	}
	id := uuid.New().String()
	_, err := d.db.Exec(`INSERT INTO sessions (uuid, name, type, status, remote_user, remote_host, remote_port, profile_name) VALUES (?, ?, 'remote', 'created', ?, ?, ?, ?)`, id, name, remoteUser, remoteHost, remotePort, profile)
	if err != nil {
		return "", fmt.Errorf("failed to create remote session: %w", err)
	}
	return id, nil
}

func (d *DB) ListSessions() ([]Session, error) {
	rows, err := d.db.Query(`SELECT uuid, name, type, status, created_at, remote_user, remote_host, remote_port, profile_name FROM sessions ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		var ru, rh sql.NullString
		var rp sql.NullInt32
		if err := rows.Scan(&s.UUID, &s.Name, &s.Type, &s.Status, &s.CreatedAt, &ru, &rh, &rp, &s.Profile); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		if ru.Valid {
			s.RemoteUser = ru.String
		}
		if rh.Valid {
			s.RemoteHost = rh.String
		}
		if rp.Valid {
			s.RemotePort = int(rp.Int32)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (d *DB) GetSessionByName(name string) (*Session, error) {
	var s Session
	var ru, rh sql.NullString
	var rp sql.NullInt32
	err := d.db.QueryRow(`SELECT uuid, name, type, status, created_at, remote_user, remote_host, remote_port, profile_name FROM sessions WHERE name = ?`, name).
		Scan(&s.UUID, &s.Name, &s.Type, &s.Status, &s.CreatedAt, &ru, &rh, &rp, &s.Profile)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to query session: %w", err)
	}
	if ru.Valid {
		s.RemoteUser = ru.String
	}
	if rh.Valid {
		s.RemoteHost = rh.String
	}
	if rp.Valid {
		s.RemotePort = int(rp.Int32)
	}
	return &s, nil
}

func (d *DB) GetTmuxProfile(name string) (string, error) {
	var config string
	err := d.db.QueryRow(`SELECT config_text FROM tmux_profiles WHERE name = ?`, name).Scan(&config)
	return config, err
}

func (d *DB) UpdateSessionStatus(uuid, status string) error {
	_, err := d.db.Exec(`UPDATE sessions SET status = ? WHERE uuid = ?`, status, uuid)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	return nil
}

func (d *DB) DeleteSession(uuid string) error {
	_, err := d.db.Exec(`DELETE FROM sessions WHERE uuid = ?`, uuid)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}
