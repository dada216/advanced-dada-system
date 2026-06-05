package main

import (
	"fmt"
	"os"

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
		fmt.Printf("Recorder started for session: %s\n", sessionUUID)
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
