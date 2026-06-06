package sessiondb

import (
	"database/sql"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/advanced-dada-system/ads/internal/ansi"
	"github.com/creack/pty"
	"github.com/google/uuid"
)

func TestIntegrationCommandHistory(t *testing.T) {
	uuidStr := uuid.New().String()
	db, err := Open(uuidStr)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()
	defer os.Remove(uuidStr + ".db")
	defer os.Remove(uuidStr + ".db-shm")
	defer os.Remove(uuidStr + ".db-wal")

	// Set up PTY with bash
	cmd := exec.Command("bash", "--noprofile", "--norc")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatalf("pty.Start failed: %v", err)
	}
	defer ptmx.Close()

	// Read from PTY and write to DB
	scanner := ansi.NewOSCScanner(db)

	done := make(chan struct{})
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				_ = db.WriteChunk(buf[:n])
				_, _ = scanner.Write(buf[:n])
			}
			if err != nil {
				if err == io.EOF || strings.Contains(err.Error(), "input/output error") {
					close(done)
					return
				}
			}
		}
	}()

	// Inject hook
	hookScript := `
_ads_prompt_command() {
    local exit_code=$?
    printf "\033]133;D;%s\007" "$exit_code"
    printf "\033]133;A\007"
}
PROMPT_COMMAND="_ads_prompt_command; $PROMPT_COMMAND"
PS1="$PS1\[\033]133;B\007\]"
PS0="\033]133;C\007"
`
	_, _ = ptmx.Write([]byte(hookScript + "\n"))
	time.Sleep(200 * time.Millisecond)

	// Run a command
	_, _ = ptmx.Write([]byte("echo my_secret_test_command\n"))
	time.Sleep(200 * time.Millisecond)

	// Exit
	_, _ = ptmx.Write([]byte("exit\n"))

	// Wait for bash to exit
	_ = cmd.Wait()
	<-done

	// Verify command_history
	rows, err := db.db.Query("SELECT command_text FROM command_history")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	var commands []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		commands = append(commands, text)
		t.Logf("Found command in history: %q", text)
	}

	found := false
	for _, cmd := range commands {
		if strings.Contains(cmd, "my_secret_test_command") {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("Command history did not contain 'my_secret_test_command'. Got: %v", commands)
	}
}

func TestIntegrationCommandHistoryZsh(t *testing.T) {
	uuidStr := uuid.New().String()
	db, err := Open(uuidStr)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()
	defer os.Remove(uuidStr + ".db")
	defer os.Remove(uuidStr + ".db-shm")
	defer os.Remove(uuidStr + ".db-wal")

	// Set up PTY with bash (we simulate zsh hook output manually)
	cmd := exec.Command("bash", "--noprofile", "--norc")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatalf("pty.Start failed: %v", err)
	}
	defer ptmx.Close()

	scanner := ansi.NewOSCScanner(db)
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				_ = db.WriteChunk(buf[:n])
				_, _ = scanner.Write(buf[:n])
			}
			if err != nil {
				if err == io.EOF || strings.Contains(err.Error(), "input/output error") {
					close(done)
					return
				}
			}
		}
	}()

	// Simulate Zsh hook where ZLE redraws prompt right before execution
	zshSimulation := `
# A normal prompt starts
printf "\033]133;D;0\007"
printf "\033]133;A\007"
printf "\033]133;B\007"
# User types a command
printf "echo my_zsh_redrawn_command\n"
# ZLE redraws the prompt, wiping the PTY buffer!
printf "\033]133;B\007"
# The preexec hook runs, sending the command text explicitly
printf "\033]133;C;echo my_zsh_redrawn_command\007"
# Command executes
echo my_zsh_redrawn_command
# Command finishes, emit D marker
printf "\033]133;D;0\007"
exit
`
	_, _ = ptmx.Write([]byte(zshSimulation))

	_ = cmd.Wait()
	<-done

	rows, err := db.db.Query("SELECT command_text FROM command_history")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	var commands []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		commands = append(commands, text)
		t.Logf("Found command in history: %q", text)
	}

	found := false
	for _, cmd := range commands {
		if strings.Contains(cmd, "echo my_zsh_redrawn_command") {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("Command history did not contain 'echo my_zsh_redrawn_command'. Got: %v", commands)
	}
}

func TestSessionDBMisc(t *testing.T) {
	uuidStr := uuid.New().String()
	db, err := Open(uuidStr)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()
	defer os.Remove(uuidStr + ".db")
	defer os.Remove(uuidStr + ".db-shm")
	defer os.Remove(uuidStr + ".db-wal")

	// Test InjectMetadata
	err = db.InjectMetadata(sql.NullString{String: "cust", Valid: true}, sql.NullString{String: "srv", Valid: true}, sql.NullString{String: "proj", Valid: true})
	if err != nil {
		t.Fatalf("InjectMetadata failed: %v", err)
	}

	// Second injection should do nothing (idempotent)
	err = db.InjectMetadata(sql.NullString{String: "cust2", Valid: true}, sql.NullString{String: "srv2", Valid: true}, sql.NullString{String: "proj2", Valid: true})
	if err != nil {
		t.Fatalf("Second InjectMetadata failed: %v", err)
	}

	// Verify metadata
	var c, s, p string
	err = db.db.QueryRow("SELECT customer, server, project FROM metadata").Scan(&c, &s, &p)
	if err != nil {
		t.Fatalf("QueryRow failed: %v", err)
	}
	if c != "cust" || s != "srv" || p != "proj" {
		t.Errorf("Metadata mismatch, got: %v %v %v", c, s, p)
	}

	// Test WriteChunk empty
	err = db.WriteChunk([]byte{})
	if err != nil {
		t.Fatalf("WriteChunk empty failed: %v", err)
	}
}
