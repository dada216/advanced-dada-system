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
_ads_precmd() {
    local exit_code=$?
    printf "\033]133;D;%s\007" "$exit_code"
    printf "\033]133;A\007"
}
_ads_preexec() {
    printf "\033]133;C\007"
}
autoload -Uz add-zsh-hook
add-zsh-hook precmd _ads_precmd
add-zsh-hook preexec _ads_preexec
PROMPT="${PROMPT}%{\033]133;B\007%}"
`
	ptmx.Write([]byte(hook + "\n"))
	time.Sleep(500 * time.Millisecond)

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				fmt.Printf("READ: %q\n", string(buf[:n]))
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
