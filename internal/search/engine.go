package search

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/advanced-dada-system/ads/internal/config"
	"github.com/advanced-dada-system/ads/internal/meta"
	_ "github.com/mattn/go-sqlite3"
)

type Result struct {
	SessionName string
	SessionUUID string
	RowID       int64
	Snippet     string
}

func Query(term string) ([]Result, error) {
	appDir, err := config.InitAppDataDir()
	if err != nil {
		return nil, err
	}

	metaDB, err := meta.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open meta db: %w", err)
	}
	defer metaDB.Close()

	sessions, err := metaDB.ListSessions()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	var results []Result

	// Use ANSI color codes for highlight markers: Red text for matches
	query := `SELECT rowid, highlight(fts_index, 0, "` + "\033[31m" + `", "` + "\033[0m" + `") 
              FROM fts_index WHERE fts_index MATCH ? ORDER BY rank LIMIT 5`

	for _, s := range sessions {
		// Use _busy_timeout to prevent SQLITE_BUSY silent skips when ads-recorder is actively writing
		dbPath := filepath.Join(appDir, "sessions", fmt.Sprintf("%s.db", s.UUID)) + "?_busy_timeout=5000&_journal_mode=WAL"

		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to open db for session %s: %v\n", s.Name, err)
			continue
		}

		cleanTerm := strings.ReplaceAll(term, "\"", "")
		ftsTerm := "\"" + cleanTerm + "\"*"

		rows, err := db.Query(query, ftsTerm)
		if err != nil {
			// Don't warn if the table just doesn't exist yet, but warn for other errors (like locks)
			if err.Error() != "no such table: fts_index" {
				fmt.Fprintf(os.Stderr, "Warning: query failed for session %s: %v\n", s.Name, err)
			}
			db.Close()
			continue
		}

		for rows.Next() {
			var r Result
			r.SessionName = s.Name
			r.SessionUUID = s.UUID
			if err := rows.Scan(&r.RowID, &r.Snippet); err == nil {
				results = append(results, r)
			}
		}
		rows.Close()
		db.Close()
	}

	return results, nil
}
