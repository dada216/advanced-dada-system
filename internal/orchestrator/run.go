package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

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

	// Update status to running
	if err := db.UpdateSessionStatus(session.UUID, "running"); err != nil {
		return err
	}

	// Get absolute path of current executable to find ads-shell
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	binDir := filepath.Dir(execPath)
	shellBin := filepath.Join(binDir, "ads-shell")

	// We use syscall.Exec to replace the current process with ads-shell
	args := []string{"ads-shell", "--session", session.UUID}
	env := os.Environ()

	if err := syscall.Exec(shellBin, args, env); err != nil {
		return fmt.Errorf("failed to exec ads-shell: %w", err)
	}

	return nil
}
