package orchestrator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/advanced-dada-system/ads/internal/meta"
)

func Run(name string) error {
	db, err := meta.Open()
	if err != nil {
		return fmt.Errorf("failed to open meta db: %w", err)
	}
	defer db.Close()

	session, err := db.GetSessionByName(name)
	if err != nil {
		return err
	}

	if session.Status == "running" {
		// Try to reattach
		fmt.Printf("Session %s is already running. Attempting to reattach...\n", name)
		attachCmd := exec.Command("tmux", "attach", "-t", session.UUID)
		attachCmd.Stdin = os.Stdin
		attachCmd.Stdout = os.Stdout
		attachCmd.Stderr = os.Stderr
		return attachCmd.Run()
	}

	// Update status
	if err := db.UpdateSessionStatus(session.UUID, "running"); err != nil {
		return err
	}

	// Make sure we update to sealed when we exit
	defer func() {
		_ = db.UpdateSessionStatus(session.UUID, "sealed")
	}()

	// 1. Create detached tmux session
	newCmd := exec.Command("tmux", "new-session", "-d", "-s", session.UUID, "bash")
	if err := newCmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Get absolute path of current executable to find ads-recorder
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	binDir := filepath.Dir(execPath)
	recorderBin := filepath.Join(binDir, "ads-recorder")

	// 2. Start pipe-pane
	// Redirect stderr to a log file so we can catch any startup/db errors.
	logFile := filepath.Join(os.TempDir(), fmt.Sprintf("ads-recorder-%s.log", session.UUID))
	pipeCommand := fmt.Sprintf("%s --session %s 2>> %s", recorderBin, session.UUID, logFile)

	// We use standard pipe-pane without '-o' which can toggle instead of forcefully open.
	pipeCmd := exec.Command("tmux", "pipe-pane", "-t", session.UUID, pipeCommand)
	if err := pipeCmd.Run(); err != nil {
		return fmt.Errorf("failed to start pipe-pane: %w", err)
	}

	// 3. Attach
	attachCmd := exec.Command("tmux", "attach", "-t", session.UUID)
	attachCmd.Stdin = os.Stdin
	attachCmd.Stdout = os.Stdout
	attachCmd.Stderr = os.Stderr

	if err := attachCmd.Run(); err != nil {
		return fmt.Errorf("failed to attach to tmux session: %w", err)
	}

	return nil
}
