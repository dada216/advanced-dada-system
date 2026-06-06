package main
import (
	"fmt"
	"os/exec"
	"path/filepath"
)
func main() {
	cmd := exec.Command("/bin/bash")
	cmd.Args[0] = "-" + filepath.Base("/bin/bash")
	fmt.Printf("Path: %s, Args: %v\n", cmd.Path, cmd.Args)
}
