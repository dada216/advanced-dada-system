package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/advanced-dada-system/ads/internal/config"
	"github.com/advanced-dada-system/ads/internal/meta"
	"github.com/advanced-dada-system/ads/internal/plugin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var llmCmd = &cobra.Command{
	Use:   "llm [subcommand]",
	Short: "AI analytics powered by OpenRouter",
}

var llmSummarizeCmd = &cobra.Command{
	Use:   "summarize [session-name]",
	Short: "Summarize a session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sessionName := args[0]

		context, err := getSessionContext(sessionName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting session context: %v\n", err)
			os.Exit(1)
		}

		prompt := fmt.Sprintf("Summarize the main events and any errors in the terminal session named %s. Here is the recent terminal output context:\n\n%s", sessionName, context)

		fmt.Printf("Analyzing session '%s' with OpenRouter...\n", sessionName)
		res, err := plugin.CallPlugin("ads-plugin-llm", map[string]string{"prompt": prompt})
		if err != nil {
			fmt.Fprintf(os.Stderr, "LLM Plugin Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n--- Summary ---")
		fmt.Println(res)
	},
}

var llmAskCmd = &cobra.Command{
	Use:   "ask [session-name] [question]",
	Short: "Ask a specific question about a session",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		sessionName := args[0]
		question := args[1]

		context, err := getSessionContext(sessionName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting session context: %v\n", err)
			os.Exit(1)
		}

		prompt := fmt.Sprintf("Regarding the terminal session named %s. Here is the recent terminal output context:\n\n%s\n\nQuestion: %s", sessionName, context, question)

		fmt.Printf("Analyzing session '%s' with OpenRouter...\n", sessionName)
		res, err := plugin.CallPlugin("ads-plugin-llm", map[string]string{"prompt": prompt})
		if err != nil {
			fmt.Fprintf(os.Stderr, "LLM Plugin Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n--- Answer ---")
		fmt.Println(res)
	},
}

func getSessionContext(sessionName string) (string, error) {
	db, err := meta.Open()
	if err != nil {
		return "", err
	}
	defer db.Close()

	session, err := db.GetSessionByName(sessionName)
	if err != nil {
		return "", err
	}

	appDir, _ := config.InitAppDataDir()
	dbPath := filepath.Join(appDir, "sessions", session.UUID+".db")

	sdb, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return "", err
	}
	defer sdb.Close()

	rows, err := sdb.Query(`SELECT text FROM fts_index ORDER BY rowid DESC LIMIT 200`)
	if err != nil {
		// Table might not exist or be empty
		return "", nil
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err == nil {
			lines = append(lines, text)
		}
	}

	// Reverse to chronological order
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}

	return strings.Join(lines, "\n"), nil
}

func init() {
	llmCmd.AddCommand(llmSummarizeCmd)
	llmCmd.AddCommand(llmAskCmd)
	rootCmd.AddCommand(llmCmd)
}
