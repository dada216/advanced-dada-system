package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/advanced-dada-system/ads/internal/meta"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <profile-name>",
	Short: "Edit profile metadata tags interactively",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		profileName := args[0]

		db, err := meta.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening meta DB: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		p, err := db.GetProfileMetadata(profileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching profile: %v\n", err)
			os.Exit(1)
		}

		reader := bufio.NewReader(os.Stdin)

		askField := func(prompt string, current sql.NullString) sql.NullString {
			defaultVal := ""
			if current.Valid {
				defaultVal = current.String
			}
			if defaultVal != "" {
				fmt.Printf("%s (%s): ", prompt, defaultVal)
			} else {
				fmt.Printf("%s: ", prompt)
			}

			input, _ := reader.ReadString('\n')
			input = strings.TrimSuffix(input, "\n")
			input = strings.TrimSuffix(input, "\r")

			if input == "" {
				return current // keep existing
			}
			if strings.TrimSpace(input) == "" {
				return sql.NullString{Valid: false} // "inputs a blank space" -> null
			}
			return sql.NullString{String: input, Valid: true}
		}

		fmt.Printf("Editing metadata for profile: %s\n", profileName)

		newCustomer := askField("Customer", p.Customer)
		newServer := askField("Server", p.Server)
		newProject := askField("Project", p.Project)

		err = db.UpdateProfileMetadata(profileName, newCustomer, newServer, newProject)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating profile metadata: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Profile metadata updated successfully.")
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
