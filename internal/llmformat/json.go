package llmformat

import (
	"database/sql"
	"encoding/json"
	"strings"
)

// JSONFormatter formats the history as a JSON string, which is highly
// effective for structured reasoning agents or LLMs operating with function calling.
type JSONFormatter struct{}

type SessionEvent struct {
	Command  string `json:"command,omitempty"`
	Output   string `json:"output,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
}

func (j *JSONFormatter) Format(sdb *sql.DB, cfg Config) (string, error) {
	limit := cfg.Limit
	if limit <= 0 {
		limit = 50
	}

	rows, err := sdb.Query(`SELECT command_text, output_text, exit_code FROM command_history ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var events []SessionEvent
	for rows.Next() {
		var cmdText, outText string
		var exitCode int
		if err := rows.Scan(&cmdText, &outText, &exitCode); err == nil {
			events = append(events, SessionEvent{
				Command:  strings.TrimSpace(cmdText),
				Output:   strings.TrimSpace(outText),
				ExitCode: exitCode,
			})
		}
	}

	// Reverse to chronological order (oldest first)
	for i, k := 0, len(events)-1; i < k; i, k = i+1, k-1 {
		events[i], events[k] = events[k], events[i]
	}

	b, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return "", err
	}

	return string(b), nil
}
