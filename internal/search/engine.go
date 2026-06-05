package search

import (
	"database/sql"
	"fmt"
	"path/filepath"

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
	query := `SELECT rowid, snippet(fts_index, 0, "` + "\033[31m" + `", "` + "\033[0m" + `", '...', 10) 
              FROM fts_index WHERE fts_index MATCH ? ORDER BY rank LIMIT 5`

	for _, s := range sessions {
		dbPath := filepath.Join(appDir, "sessions", fmt.Sprintf("%s.db", s.UUID))

		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			continue // Skip errors for individual DBs to allow partial federated search
		}

		rows, err := db.Query(query, term)
		if err != nil {
			db.Close()
			continue // Skip if fts_index table doesn't exist or query fails
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
