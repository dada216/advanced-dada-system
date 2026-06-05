package orchestrator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/advanced-dada-system/ads/internal/meta"
	"github.com/advanced-dada-system/ads/internal/sessiondb"
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

	profileMeta, err := db.GetProfileMetadata(session.Profile)
	if err == nil {
		sdb, serr := sessiondb.Open(session.UUID)
		if serr == nil {
			_ = sdb.InjectMetadata(profileMeta.Customer, profileMeta.Server, profileMeta.Project)
			sdb.Close()
		}
	}

	// Check if tmux session already exists natively in our isolated 'ads' server
	hasSessionCmd := exec.Command("tmux", "-L", "ads", "has-session", "-t", session.UUID)
	sessionExists := hasSessionCmd.Run() == nil

	// Get absolute path of current executable to find ads-recorder and set AdsBinaryPath
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	binDir := filepath.Dir(execPath)
	recorderBin := filepath.Join(binDir, "ads-recorder")

	// Render tmux profile ALWAYS
	configTpl, err := db.GetTmuxProfile(session.Profile)
	if err != nil {
		return fmt.Errorf("failed to get tmux profile '%s': %w", session.Profile, err)
	}

	tpl, err := template.New("tmux").Parse(configTpl)
	if err != nil {
		return fmt.Errorf("failed to parse tmux profile template: %w", err)
	}

	confPath := filepath.Join(os.TempDir(), fmt.Sprintf("ads-tmux-%s.conf", session.UUID))
	f, err := os.Create(confPath)
	if err != nil {
		return fmt.Errorf("failed to create tmux config file: %w", err)
	}

	err = tpl.Execute(f, struct {
		AdsBinaryPath string
		SessionUUID   string
	}{
		AdsBinaryPath: execPath,
		SessionUUID:   session.UUID,
	})
	f.Close()
	if err != nil {
		return fmt.Errorf("failed to render tmux profile: %w", err)
	}

	if sessionExists {
		fmt.Printf("Session %s is already alive in tmux. Reattaching...\n", name)

		// Force apply the configuration to the server in case it was updated
		sourceCmd := exec.Command("tmux", "-L", "ads", "source-file", confPath)
		_ = sourceCmd.Run()
	} else {
		// Use SSH for remote sessions, bash for local
		shellCmd := "bash"
		if session.Type == "remote" {
			shellCmd = fmt.Sprintf("ssh -t -p %d %s@%s", session.RemotePort, session.RemoteUser, session.RemoteHost)
		}

		newCmd := exec.Command("tmux", "-L", "ads", "-f", confPath, "new-session", "-d", "-s", session.UUID, shellCmd)
		if err := newCmd.Run(); err != nil {
			return fmt.Errorf("failed to create tmux session: %w", err)
		}

		// Force apply the configuration to the server in case it was already running and ignored the -f flag
		sourceCmd := exec.Command("tmux", "-L", "ads", "source-file", confPath)
		_ = sourceCmd.Run()

		// 2. Start pipe-pane
		// Redirect stderr to a log file so we can catch any startup/db errors.
		logFile := filepath.Join(os.TempDir(), fmt.Sprintf("ads-recorder-%s.log", session.UUID))
		pipeCommand := fmt.Sprintf("'%s' --session '%s' 2>> '%s'", recorderBin, session.UUID, logFile)

		// We use standard pipe-pane without '-o' which can toggle instead of forcefully open.
		pipeCmd := exec.Command("tmux", "-L", "ads", "pipe-pane", "-t", session.UUID, pipeCommand)
		if err := pipeCmd.Run(); err != nil {
			return fmt.Errorf("failed to start pipe-pane: %w", err)
		}
	}

	// Update status to running
	if err := db.UpdateSessionStatus(session.UUID, "running"); err != nil {
		return err
	}

	// Make sure we update to sealed when we exit
	defer func() {
		_ = db.UpdateSessionStatus(session.UUID, "sealed")
	}()

	// 3. Attach
	attachCmd := exec.Command("tmux", "-L", "ads", "attach", "-t", session.UUID)
	attachCmd.Stdin = os.Stdin
	attachCmd.Stdout = os.Stdout
	attachCmd.Stderr = os.Stderr

	if err := attachCmd.Run(); err != nil {
		return fmt.Errorf("failed to attach to tmux session: %w", err)
	}

	return nil
}
