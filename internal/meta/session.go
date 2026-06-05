package meta

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	UUID       string
	Name       string
	Type       string
	Status     string
	CreatedAt  time.Time
	RemoteUser string
	RemoteHost string
	RemotePort int
	Profile    string
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
