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
	remote_port INTEGER
);
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

	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) CreateLocalSession(name string) (string, error) {
	id := uuid.New().String()
	_, err := d.db.Exec(`INSERT INTO sessions (uuid, name, type, status) VALUES (?, ?, 'local', 'created')`, id, name)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	return id, nil
}

func (d *DB) CreateRemoteSession(name, remoteUser, remoteHost string, remotePort int) (string, error) {
	id := uuid.New().String()
	_, err := d.db.Exec(`INSERT INTO sessions (uuid, name, type, status, remote_user, remote_host, remote_port) VALUES (?, ?, 'remote', 'created', ?, ?, ?)`, id, name, remoteUser, remoteHost, remotePort)
	if err != nil {
		return "", fmt.Errorf("failed to create remote session: %w", err)
	}
	return id, nil
}

func (d *DB) ListSessions() ([]Session, error) {
	rows, err := d.db.Query(`SELECT uuid, name, type, status, created_at, remote_user, remote_host, remote_port FROM sessions ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		var ru, rh sql.NullString
		var rp sql.NullInt32
		if err := rows.Scan(&s.UUID, &s.Name, &s.Type, &s.Status, &s.CreatedAt, &ru, &rh, &rp); err != nil {
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
	err := d.db.QueryRow(`SELECT uuid, name, type, status, created_at, remote_user, remote_host, remote_port FROM sessions WHERE name = ?`, name).
		Scan(&s.UUID, &s.Name, &s.Type, &s.Status, &s.CreatedAt, &ru, &rh, &rp)
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
