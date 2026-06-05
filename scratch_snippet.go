package main

import (
	"fmt"
	"strings"
)

func cleanSnippet(s string) string {
	lines := strings.Split(s, "\n")
	var matchLine string
	for _, line := range lines {
		if strings.Contains(line, "\033[31m") {
			matchLine = line
			break
		}
	}
	if matchLine == "" {
		matchLine = s
	}

	s = matchLine
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", "    ")
	s = strings.ReplaceAll(s, "\b", "")
	return strings.TrimSpace(s)
}

func main() {
	rawChunk := "\r\n\033[31mkubernetes\033[0m\r\nroot@k8s-master:~# "
	fmt.Printf("Before: %q\n", strings.ReplaceAll(strings.ReplaceAll(rawChunk, "\n", " "), "\r", ""))
	fmt.Printf("After: %q\n", cleanSnippet(rawChunk))
}
