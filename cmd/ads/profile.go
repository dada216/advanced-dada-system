package main

import (
	"fmt"
	"os"
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

func init() {
	profileCmd.AddCommand(profileNewCmd)
	profileCmd.AddCommand(profileListCmd)
	rootCmd.AddCommand(profileCmd)
}
