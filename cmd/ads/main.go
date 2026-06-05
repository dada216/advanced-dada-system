package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/advanced-dada-system/ads/internal/meta"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ads",
	Short: "Advanced Dada System - Terminal Analytics Platform",
	Long:  "ads is a CLI tool to manage and analyze recorded terminal sessions.",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var newCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Create a new session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		db, err := meta.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening meta database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		name := args[0]
		uuid, err := db.CreateSession(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating session: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Session created!\n")
		fmt.Printf("Name: %s\n", name)
		fmt.Printf("UUID: %s\n", uuid)
		fmt.Printf("\nNext step: Run 'ads run %s' to start the session.\n", name)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sessions",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := meta.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening meta database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		sessions, err := db.ListSessions()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing sessions: %v\n", err)
			os.Exit(1)
		}

		if len(sessions) == 0 {
			fmt.Println("No sessions found.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tUUID\tTYPE\tSTATUS\tCREATED AT")
		for _, s := range sessions {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", s.Name, s.UUID, s.Type, s.Status, s.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(listCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
