package llmformat

import (
	"database/sql"
	"strings"
)

// RawFormatter implements the legacy formatting style but applies
// LLM-specific token optimizations like noise reduction.
type RawFormatter struct{}

func (r *RawFormatter) Format(sdb *sql.DB, cfg Config) (string, error) {
	limit := cfg.Limit
	if limit <= 0 {
		limit = 200 // Default legacy limit
	}

	rows, err := sdb.Query(`SELECT text FROM fts_index ORDER BY rowid DESC LIMIT ?`, limit)
	if err != nil {
		return "", nil // Return empty if table is missing or errors
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err == nil {
			text = strings.TrimRight(text, " \t")
			if text != "" {
				lines = append(lines, text)
			}
		}
	}

	// Reverse to chronological order
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}

	return optimizeOutput(strings.Join(lines, "\n")), nil
}
