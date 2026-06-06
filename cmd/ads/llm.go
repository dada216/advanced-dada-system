package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/advanced-dada-system/ads/internal/config"
	"github.com/advanced-dada-system/ads/internal/llmformat"
	"github.com/advanced-dada-system/ads/internal/meta"
	"github.com/advanced-dada-system/ads/internal/plugin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var llmCmd = &cobra.Command{
	Use:   "llm [subcommand]",
	Short: "AI analytics powered by OpenRouter",
}

// Global flag for LLM format
var llmFormatFlag string

var llmSummarizeCmd = &cobra.Command{
	Use:   "summarize [session-name]",
	Short: "Summarize a session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sessionName := args[0]

		context, err := getSessionContext(sessionName, llmFormatFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting session context: %v\n", err)
			os.Exit(1)
		}

		prompt := fmt.Sprintf("Summarize the main events and any errors in the terminal session named %s. Here is the highly optimized terminal output context:\n\n%s", sessionName, context)

		fmt.Printf("Analyzing session '%s' with OpenRouter (Format: %s)...\n", sessionName, llmFormatFlag)
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

		context, err := getSessionContext(sessionName, llmFormatFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting session context: %v\n", err)
			os.Exit(1)
		}

		prompt := fmt.Sprintf("Regarding the terminal session named %s. Here is the highly optimized terminal output context:\n\n%s\n\nQuestion: %s", sessionName, context, question)

		fmt.Printf("Analyzing session '%s' with OpenRouter (Format: %s)...\n", sessionName, llmFormatFlag)
		res, err := plugin.CallPlugin("ads-plugin-llm", map[string]string{"prompt": prompt})
		if err != nil {
			fmt.Fprintf(os.Stderr, "LLM Plugin Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n--- Answer ---")
		fmt.Println(res)
	},
}

var llmExportCmd = &cobra.Command{
	Use:   "export [session-name]",
	Short: "Export a session's history optimized for LLM ingestion",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sessionName := args[0]
		context, err := getSessionContext(sessionName, llmFormatFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting session context: %v\n", err)
			os.Exit(1)
		}
		// Print strictly the optimized output to stdout for piping
		fmt.Print(context)
	},
}

func getSessionContext(sessionName string, format string) (string, error) {
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

	sdb, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return "", err
	}
	defer sdb.Close()

	formatter := llmformat.GetFormatter(format)
	cfg := llmformat.Config{
		Limit:  50,
		Format: format,
	}

	return formatter.Format(sdb, cfg)
}

func init() {
	llmCmd.PersistentFlags().StringVarP(&llmFormatFlag, "format", "f", "semantic", "Format to use for LLM context (semantic, raw, json)")
	
	llmCmd.AddCommand(llmSummarizeCmd)
	llmCmd.AddCommand(llmAskCmd)
	llmCmd.AddCommand(llmExportCmd)
	rootCmd.AddCommand(llmCmd)
}
