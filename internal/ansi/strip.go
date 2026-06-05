package ansi

import (
	"regexp"
)

var (
	csiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	oscRegex = regexp.MustCompile(`\x1b\].*?(?:\x07|\x1b\\)`)
)

func Strip(data []byte) string {
	clean := oscRegex.ReplaceAll(data, nil)
	clean = csiRegex.ReplaceAll(clean, nil)
	return string(clean)
}
