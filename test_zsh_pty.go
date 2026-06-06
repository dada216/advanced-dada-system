package main

import (
	"fmt"
	"github.com/creack/pty"
	"os/exec"
	"time"
)

func main() {
	cmd := exec.Command("zsh", "--no-rcs")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		fmt.Println("Error starting pty:", err)
		return
	}
	defer ptmx.Close()

	hook := `
PROMPT="TEST_PROMPT%{\033]133;B\007%}"
`
	ptmx.Write([]byte(hook + "\n"))
	time.Sleep(500 * time.Millisecond)

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				fmt.Printf("READ (%%q): %q\n", string(buf[:n]))
			}
			if err != nil {
				return
			}
		}
	}()

	ptmx.Write([]byte("echo my_zsh_test_command\n"))
	time.Sleep(1 * time.Second)
	ptmx.Write([]byte("exit\n"))
	cmd.Wait()
}
