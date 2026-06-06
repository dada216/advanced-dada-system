package ansi

import (
	"testing"
	"time"
)

type mockInserter struct {
	commands []string
}

func (m *mockInserter) InsertCommand(commandText string, startTs, endTs time.Time, exitCode int) error {
	m.commands = append(m.commands, commandText)
	return nil
}

func TestOSCScanner(t *testing.T) {
	mock := &mockInserter{}
	scanner := NewOSCScanner(mock)

	// Simulate ads hook bash output
	// 1. Prompt prints (A ends previous, B starts input)
	_, _ = scanner.Write([]byte("\x1b]133;A\x07"))
	_, _ = scanner.Write([]byte("\x1b]133;B\x07bash-5.1$ "))

	// 2. User types "echo hello"
	_, _ = scanner.Write([]byte("echo hello\r\n"))

	// 3. Preexec (C starts execution)
	_, _ = scanner.Write([]byte("\x1b]133;C\x07"))

	// 4. Command outputs
	_, _ = scanner.Write([]byte("hello\r\n"))

	// 5. Command ends
	_, _ = scanner.Write([]byte("\x1b]133;D;0\x07"))

	if len(mock.commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(mock.commands))
	}

	if mock.commands[0] != "bash-5.1$ echo hello" {
		t.Errorf("unexpected command text: %q", mock.commands[0])
	}

	// Test 2: Chunked writes with random ANSI escape sequences (which trigger lastEsc bug)
	mock.commands = nil
	_, _ = scanner.Write([]byte("\x1b]133;A\x07"))
	_, _ = scanner.Write([]byte("\x1b]133;B\x07bash-5.1$ "))
	_, _ = scanner.Write([]byte("ls "))
	_, _ = scanner.Write([]byte("\x1b[K")) // some escape sequence
	_, _ = scanner.Write([]byte("-la\r\n"))
	_, _ = scanner.Write([]byte("\x1b]133;C\x07"))
	_, _ = scanner.Write([]byte("output\r\n"))
	_, _ = scanner.Write([]byte("\x1b]133;D;0\x07"))

	if len(mock.commands) != 1 {
		t.Fatalf("Test 2 expected 1 command, got %d", len(mock.commands))
	}
	if mock.commands[0] != "bash-5.1$ ls -la" {
		t.Errorf("Test 2 unexpected command text: %q", mock.commands[0])
	}

	// Test 3: The starvation/leak bug
	_, _ = scanner.Write([]byte("\x1b]133;C\x07"))
	// Simulating a huge output with ANSI colors but no OSC 133
	_, _ = scanner.Write([]byte("\x1b[0m"))
	for i := 0; i < 10000; i++ {
		_, _ = scanner.Write([]byte("line of output\n"))
	}
	if scanner.stream.Len() > 100 {
		t.Fatalf("Stream buffer leaked! Size is %d", scanner.stream.Len())
	}
}
