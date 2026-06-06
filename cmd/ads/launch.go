package main

import (
	"fmt"
	"os"
	"time"

	"github.com/advanced-dada-system/ads/internal/meta"
	"github.com/advanced-dada-system/ads/internal/orchestrator"
	"github.com/spf13/cobra"
)

var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Automatically create and run a new local session",
	Run: func(cmd *cobra.Command, args []string) {
		name := fmt.Sprintf("ads-%d", time.Now().Unix())

		db, err := meta.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening meta database: %v\n", err)
			os.Exit(1)
		}

		_, err = db.CreateLocalSession(name, "default")
		db.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating session: %v\n", err)
			os.Exit(1)
		}

		// fmt.Printf("Created local session %s (UUID: %s)\n", name, uuid)

		if err := orchestrator.Run(name, launchShell); err != nil {
			fmt.Fprintf(os.Stderr, "Error running session: %v\n", err)
			os.Exit(1)
		}
	},
}

var launchShell string

func init() {
	launchCmd.Flags().StringVarP(&launchShell, "shell", "s", "", "Explicit shell to launch")
	rootCmd.AddCommand(launchCmd)
}
