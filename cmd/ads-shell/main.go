package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/advanced-dada-system/ads/internal/ansi"
	"github.com/advanced-dada-system/ads/internal/meta"
	"github.com/advanced-dada-system/ads/internal/sessiondb"
	"github.com/creack/pty"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	sessionUUID string
)

var rootCmd = &cobra.Command{
	Use:   "ads-shell",
	Short: "PTY proxy and recorder for Advanced Dada System",
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

		mdb, err := meta.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening meta database: %v\n", err)
			os.Exit(1)
		}
		defer mdb.Close()

		sessionInfo, err := mdb.GetSessionByUUID(sessionUUID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting session info: %v\n", err)
			os.Exit(1)
		}

		shellCmd := "bash"
		var shellArgs []string
		if sessionInfo.Type == "remote" {
			shellCmd = "ssh"
			shellArgs = []string{"-t", "-p", fmt.Sprintf("%d", sessionInfo.RemotePort), fmt.Sprintf("%s@%s", sessionInfo.RemoteUser, sessionInfo.RemoteHost)}
		}

		c := exec.Command(shellCmd, shellArgs...)
		c.Env = append(os.Environ(), fmt.Sprintf("ADS_SESSION=%s", sessionUUID))

		// Start the command in a pty.
		ptmx, err := pty.Start(c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error starting pty: %v\n", err)
			os.Exit(1)
		}
		// Make sure to close the pty at the end.
		defer func() { _ = ptmx.Close() }() // Best effort.

		// Handle pty size.
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGWINCH)
		go func() {
			for range ch {
				_ = pty.InheritSize(os.Stdin, ptmx)
			}
		}()
		ch <- syscall.SIGWINCH                        // Initial resize.
		defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done.

		// Set stdin in raw mode if it's a terminal.
		if term.IsTerminal(int(os.Stdin.Fd())) {
			oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error setting raw mode: %v\n", err)
				os.Exit(1)
			}
			defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.
		}

		// Copy from stdin to pty (input)
		go func() {
			_, _ = io.Copy(ptmx, os.Stdin)
		}()

		// Copy from pty to stdout and record
		scanner := ansi.NewOSCScanner(db)
		buf := make([]byte, 4096)
		for {
			n, readErr := ptmx.Read(buf)
			if n > 0 {
				chunk := buf[:n]

				// Write to database
				_ = db.WriteChunk(chunk)
				_, _ = scanner.Write(chunk)

				// Write to stdout
				_, _ = os.Stdout.Write(chunk)
			}
			if readErr != nil {
				break
			}
		}

		// Wait for the command to finish
		err = c.Wait()

		// Update status to sealed
		_ = mdb.UpdateSessionStatus(sessionUUID, "sealed")

		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		os.Exit(0)
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
