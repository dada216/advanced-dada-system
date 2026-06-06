package sessiondb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/advanced-dada-system/ads/internal/ansi"
	"github.com/advanced-dada-system/ads/internal/config"
	_ "github.com/mattn/go-sqlite3"
)

const schema = `
CREATE TABLE IF NOT EXISTS io_stream (
	id   INTEGER PRIMARY KEY,
	ts   DATETIME DEFAULT CURRENT_TIMESTAMP,
	data BLOB NOT NULL
);
CREATE VIRTUAL TABLE IF NOT EXISTS fts_index USING fts5(text);
CREATE TABLE IF NOT EXISTS command_history (
    id INTEGER PRIMARY KEY,
    command_text TEXT,
    start_ts DATETIME,
    end_ts DATETIME,
    exit_code INTEGER
);
CREATE TABLE IF NOT EXISTS metadata (
    customer TEXT,
    server TEXT,
    project TEXT,
    date DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

type DB struct {
	db *sql.DB
}

func Open(uuid string) (*DB, error) {
	appDir, err := config.InitAppDataDir()
	if err != nil {
		return nil, err
	}

	sessionsDir := filepath.Join(appDir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}

	dbPath := filepath.Join(sessionsDir, fmt.Sprintf("%s.db", uuid))

	// Open in WAL mode explicitly via DSN with busy_timeout to prevent database locked errors
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open session database: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &DB{db: db}, nil
}

func (d *DB) InsertCommand(commandText string, startTs, endTs time.Time, exitCode int) error {
	_, err := d.db.Exec(`INSERT INTO command_history (command_text, start_ts, end_ts, exit_code) VALUES (?, ?, ?, ?)`, commandText, startTs, endTs, exitCode)
	return err
}

func (d *DB) InjectMetadata(customer, server, project sql.NullString) error {
	var count int
	_ = d.db.QueryRow("SELECT COUNT(*) FROM metadata").Scan(&count)
	if count > 0 {
		return nil
	}
	_, err := d.db.Exec(`INSERT INTO metadata (customer, server, project) VALUES (?, ?, ?)`, customer, server, project)
	return err
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) WriteChunk(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}

	res, err := tx.Exec(`INSERT INTO io_stream (data) VALUES (?)`, data)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to insert io_stream: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	cleanText := ansi.Strip(data)
	if cleanText != "" {
		_, err = tx.Exec(`INSERT INTO fts_index (rowid, text) VALUES (?, ?)`, id, cleanText)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to insert fts_index: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}
