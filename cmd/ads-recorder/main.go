package main

import (
	"fmt"
	"io"
	"os"

	"github.com/advanced-dada-system/ads/internal/ansi"
	"github.com/advanced-dada-system/ads/internal/sessiondb"
	"github.com/spf13/cobra"
)

var (
	sessionUUID string
)

var rootCmd = &cobra.Command{
	Use:   "ads-recorder",
	Short: "Recorder daemon for Advanced Dada System",
	Long:  "ads-recorder reads raw bytes from stdin and writes them to a session database.",
	Run: func(cmd *cobra.Command, args []string) {
		if sessionUUID == "" {
			fmt.Fprintln(os.Stderr, "Error: --session flag is required")
			_ = cmd.Help()
			os.Exit(1)
		}

		db, err := sessiondb.Open(sessionUUID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening session database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		buf := make([]byte, 4096)
		scanner := ansi.NewOSCScanner(db)

		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				chunk := buf[:n]
				if wErr := db.WriteChunk(chunk); wErr != nil {
					fmt.Fprintf(os.Stderr, "Error writing chunk to db: %v\n", wErr)
					os.Exit(1)
				}
				_, _ = scanner.Write(chunk)
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	rootCmd.Flags().StringVar(&sessionUUID, "session", "", "UUID of the session to record")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
