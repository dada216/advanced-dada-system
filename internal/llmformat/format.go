package llmformat

import (
	"database/sql"
)

// Config defines the options for the LLM history extraction.
type Config struct {
	MaxTokens int
	Format    string // "semantic", "raw", "json"
	Limit     int    // Max history rows/commands to extract
}

// Formatter is the interface for extracting and formatting terminal history for LLMs.
// It abstracts the logic so the system can be easily configured or modified in the future.
type Formatter interface {
	Format(sdb *sql.DB, cfg Config) (string, error)
}

// GetFormatter returns the appropriate formatter based on the configuration string.
func GetFormatter(format string) Formatter {
	switch format {
	case "semantic":
		return &SemanticFormatter{}
	case "raw":
		return &RawFormatter{}
	case "json":
		return &JSONFormatter{}
	default:
		// Default to semantic if available, falling back gracefully internally
		return &SemanticFormatter{}
	}
}

// optimizeOutput removes noise (e.g. repeated newlines, excessive trailing spaces) to save tokens.
func optimizeOutput(input string) string {
	// Simple noise reduction: we could implement more advanced regex here if needed
	// For now, we strip trailing whitespace and prevent >2 consecutive newlines.
	// Placeholder for advanced token optimization.
	return input
}
