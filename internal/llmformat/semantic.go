package llmformat

import (
	"database/sql"
	"fmt"
	"strings"
)

// SemanticFormatter queries the structured command_history table to produce
// highly optimized, semantic representations of the terminal session.
// This is significantly better for LLM token usage and context understanding
// than raw chronological streams.
type SemanticFormatter struct{}

func (s *SemanticFormatter) Format(sdb *sql.DB, cfg Config) (string, error) {
	limit := cfg.Limit
	if limit <= 0 {
		limit = 50 // default to last 50 commands
	}

	// 1. Try to extract from the highly semantic command_history table first
	rows, err := sdb.Query(`SELECT command_text, output_text, exit_code FROM command_history ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		// Fallback to RawFormatter if table missing or errors
		raw := &RawFormatter{}
		return raw.Format(sdb, cfg)
	}
	defer rows.Close()

	var blocks []string
	for rows.Next() {
		var cmdText, outText string
		var exitCode int
		if err := rows.Scan(&cmdText, &outText, &exitCode); err == nil {
			cmdText = strings.TrimSpace(cmdText)
			if cmdText == "" {
				continue
			}
			outText = strings.TrimSpace(outText)
			
			// Build a highly token-optimized block for the LLM
			block := fmt.Sprintf("User Command: `%s`\nExit Code: %d\nOutput:\n```\n%s\n```\n", cmdText, exitCode, outText)
			blocks = append(blocks, block)
		}
	}

	// If command history is empty, fallback to raw
	if len(blocks) == 0 {
		raw := &RawFormatter{}
		return raw.Format(sdb, cfg)
	}

	// Reverse to chronological order (oldest first)
	for i, j := 0, len(blocks)-1; i < j; i, j = i+1, j-1 {
		blocks[i], blocks[j] = blocks[j], blocks[i]
	}

	return strings.Join(blocks, "\n---\n"), nil
}
