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
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
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

	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) CreateSession(name string) (string, error) {
	id := uuid.New().String()
	_, err := d.db.Exec(`INSERT INTO sessions (uuid, name, type, status) VALUES (?, ?, 'local', 'created')`, id, name)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	return id, nil
}

func (d *DB) ListSessions() ([]Session, error) {
	rows, err := d.db.Query(`SELECT uuid, name, type, status, created_at FROM sessions ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.UUID, &s.Name, &s.Type, &s.Status, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (d *DB) GetSessionByName(name string) (*Session, error) {
	var s Session
	err := d.db.QueryRow(`SELECT uuid, name, type, status, created_at FROM sessions WHERE name = ?`, name).
		Scan(&s.UUID, &s.Name, &s.Type, &s.Status, &s.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to query session: %w", err)
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
