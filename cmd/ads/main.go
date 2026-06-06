package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/advanced-dada-system/ads/internal/config"
	"github.com/advanced-dada-system/ads/internal/meta"
	"github.com/advanced-dada-system/ads/internal/orchestrator"
	"github.com/advanced-dada-system/ads/internal/plugin"
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

var isRemote bool
var profileName string

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
		var uuid string

		if isRemote {
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("SSH User (default: %s): ", os.Getenv("USER"))
			user, _ := reader.ReadString('\n')
			user = strings.TrimSpace(user)
			if user == "" {
				user = os.Getenv("USER")
			}

			fmt.Printf("SSH Host: ")
			host, _ := reader.ReadString('\n')
			host = strings.TrimSpace(host)
			if host == "" {
				fmt.Fprintln(os.Stderr, "SSH Host is required for remote sessions")
				os.Exit(1)
			}

			fmt.Printf("SSH Port (default: 22): ")
			portStr, _ := reader.ReadString('\n')
			portStr = strings.TrimSpace(portStr)
			port := 22
			if portStr != "" {
				p, err := strconv.Atoi(portStr)
				if err == nil && p > 0 {
					port = p
				} else {
					fmt.Fprintln(os.Stderr, "Invalid port number")
					os.Exit(1)
				}
			}

			uuid, err = db.CreateRemoteSession(name, user, host, port, profileName)
		} else {
			uuid, err = db.CreateLocalSession(name, profileName)
		}

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
		fmt.Fprintln(w, "NAME\tUUID\tTYPE\tSTATUS\tPROFILE\tCREATED AT")
		for _, s := range sessions {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", s.Name, s.UUID, s.Type, s.Status, s.Profile, s.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		w.Flush()
	},
}

var runCmd = &cobra.Command{
	Use:   "run [name]",
	Short: "Run a terminal session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		err := orchestrator.Run(name, runShell)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				fmt.Printf("Warning: Session '%s' does not exist.\n", name)
				fmt.Printf("Would you like to automatically create a new local session named '%s'? (y/N): ", name)

				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.ToLower(strings.TrimSpace(response))

				if response == "y" || response == "yes" {
					db, errDB := meta.Open()
					if errDB != nil {
						fmt.Fprintf(os.Stderr, "Error opening meta database: %v\n", errDB)
						os.Exit(1)
					}

					uuid, errDB := db.CreateLocalSession(name, "default")
					db.Close()

					if errDB != nil {
						fmt.Fprintf(os.Stderr, "Error creating session: %v\n", errDB)
						os.Exit(1)
					}

					fmt.Printf("Created local session %s (UUID: %s)\n", name, uuid)

					if errRun := orchestrator.Run(name, runShell); errRun != nil {
						fmt.Fprintf(os.Stderr, "Error running session: %v\n", errRun)
						os.Exit(1)
					}
					return
				} else {
					fmt.Println("Exiting gracefully. No session created.")
					os.Exit(0)
				}
			}

			fmt.Fprintf(os.Stderr, "Error running session: %v\n", err)
			os.Exit(1)
		}
	},
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search across all recorded sessions",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		term := args[0]
		results, err := plugin.CallSearchPlugin(term)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Search failed: %v\n", err)
			os.Exit(1)
		}

		if len(results) == 0 {
			fmt.Println("No matches found.")
			return
		}

		for _, r := range results {
			fmt.Printf("[%s] (%s): %s\n", r.SessionName, r.Date, r.Snippet)
		}
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		db, err := meta.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening meta database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		session, err := db.GetSessionByName(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting session: %v\n", err)
			os.Exit(1)
		}

		appDir, _ := config.InitAppDataDir()
		dbPath := filepath.Join(appDir, "sessions", session.UUID+".db")
		_ = os.Remove(dbPath)

		if err := db.DeleteSession(session.UUID); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting session from meta db: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Session '%s' deleted successfully.\n", name)
	},
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication testing utilities",
}

var authTestCmd = &cobra.Command{
	Use:   "test [name]",
	Short: "Test SSH connection for a remote session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		db, err := meta.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening meta database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		session, err := db.GetSessionByName(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting session: %v\n", err)
			os.Exit(1)
		}

		if session.Type != "remote" {
			fmt.Fprintf(os.Stderr, "Session '%s' is not a remote session\n", name)
			os.Exit(1)
		}

		fmt.Printf("Testing SSH connection to %s@%s:%d...\n", session.RemoteUser, session.RemoteHost, session.RemotePort)

		sshArgs := []string{
			"-p", strconv.Itoa(session.RemotePort),
			"-o", "BatchMode=yes",
			"-o", "ConnectTimeout=5",
			fmt.Sprintf("%s@%s", session.RemoteUser, session.RemoteHost),
			"echo 'SSH authentication successful'",
		}

		testCmd := exec.Command("ssh", sshArgs...)
		testCmd.Stdout = os.Stdout
		testCmd.Stderr = os.Stderr

		if err := testCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "\nSSH authentication failed!\n")
			fmt.Fprintf(os.Stderr, "Please ensure your ssh-agent is running and you have added the correct key.\n")
			os.Exit(1)
		}
	},
}

var runShell string

func init() {
	newCmd.Flags().BoolVarP(&isRemote, "remote", "r", false, "Create a remote SSH session")
	newCmd.Flags().StringVarP(&profileName, "profile", "p", "default", "Tmux profile to use")
	runCmd.Flags().StringVarP(&runShell, "shell", "s", "", "Explicit shell to launch")

	authCmd.AddCommand(authTestCmd)

	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(authCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
