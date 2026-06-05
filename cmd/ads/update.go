package main

import (
	"fmt"
	"os"

	"github.com/advanced-dada-system/ads/internal/meta"
	"github.com/spf13/cobra"
)

var updateProfileName string

var updateCmd = &cobra.Command{
	Use:   "update <session-name>",
	Short: "Update an existing session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sessionName := args[0]
		db, err := meta.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening meta DB: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		session, err := db.GetSessionByName(sessionName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching session: %v\n", err)
			os.Exit(1)
		}

		if updateProfileName != "" {
			// Verify profile exists
			_, err := db.GetProfileMetadata(updateProfileName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Profile '%s' does not exist.\n", updateProfileName)
				os.Exit(1)
			}

			err = db.UpdateSessionProfile(session.UUID, updateProfileName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error updating session profile: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Successfully updated session '%s' to use profile '%s'.\n", sessionName, updateProfileName)
			fmt.Printf("Note: The new profile configuration and metadata will be applied the next time you 'ads run %s'.\n", sessionName)
		} else {
			fmt.Println("No updates provided. Use --profile to change the session's profile.")
		}
	},
}

func init() {
	updateCmd.Flags().StringVarP(&updateProfileName, "profile", "p", "", "Attach a new profile to the session")
	rootCmd.AddCommand(updateCmd)
}
