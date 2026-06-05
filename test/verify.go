package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/advanced-dada-system/ads/internal/config"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: verify <uuid>")
		os.Exit(1)
	}
	uuid := os.Args[1]

	appDir, _ := config.InitAppDataDir()
	dbPath := filepath.Join(appDir, "sessions", fmt.Sprintf("%s.db", uuid))
	
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Printf("Failed to open DB: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	var text string
	err = db.QueryRow("SELECT text FROM fts_index LIMIT 1").Scan(&text)
	if err != nil {
		fmt.Printf("Failed to read FTS index: %v\n", err)
		os.Exit(1)
	}

	if !strings.Contains(text, "hello from tmux") {
		fmt.Printf("Expected text not found in DB. Got: %s\n", text)
		os.Exit(1)
	}

	fmt.Println("SUCCESS: Found output in SQLite DB!")
}
