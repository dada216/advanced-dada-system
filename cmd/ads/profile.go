package main

import (
	"fmt"
	"os"
	"os/exec"
	"text/tabwriter"

	"github.com/advanced-dada-system/ads/internal/meta"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage tmux profiles and metadata",
}

var profileNewCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create a new profile (clones the default config)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		db, err := meta.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening meta DB: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		err = db.CreateProfile(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating profile: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Profile '%s' created successfully.\n", name)
		fmt.Printf("You can now edit its metadata with: ads edit %s\n", name)
	},
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := meta.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening meta DB: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		profiles, err := db.ListProfiles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing profiles: %v\n", err)
			os.Exit(1)
		}

		if len(profiles) == 0 {
			fmt.Println("No profiles found.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tCUSTOMER\tSERVER\tPROJECT")
		for _, p := range profiles {
			c := ""
			s := ""
			proj := ""
			if p.Customer.Valid {
				c = p.Customer.String
			}
			if p.Server.Valid {
				s = p.Server.String
			}
			if p.Project.Valid {
				proj = p.Project.String
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Name, c, s, proj)
		}
		w.Flush()
	},
}

var profileEditConfigCmd = &cobra.Command{
	Use:   "edit-config <name>",
	Short: "Edit the raw tmux configuration for a profile",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		db, err := meta.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening meta DB: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		configText, err := db.GetTmuxProfile(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching profile config: %v\n", err)
			os.Exit(1)
		}

		// Create a temporary file
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("ads-profile-%s-*.conf", name))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating temp file: %v\n", err)
			os.Exit(1)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString(configText); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to temp file: %v\n", err)
			os.Exit(1)
		}
		tmpFile.Close()

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nano"
		}

		editCmd := exec.Command(editor, tmpFile.Name())
		editCmd.Stdin = os.Stdin
		editCmd.Stdout = os.Stdout
		editCmd.Stderr = os.Stderr

		if err := editCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running editor: %v\n", err)
			os.Exit(1)
		}

		newConfigBytes, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading updated config: %v\n", err)
			os.Exit(1)
		}

		newConfig := string(newConfigBytes)
		if newConfig != configText {
			if err := db.UpdateProfileConfig(name, newConfig); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving new config: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Successfully updated configuration for profile '%s'.\n", name)
		} else {
			fmt.Println("No changes made to configuration.")
		}
	},
}

func init() {
	profileCmd.AddCommand(profileNewCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileEditConfigCmd)
	rootCmd.AddCommand(profileCmd)
}
