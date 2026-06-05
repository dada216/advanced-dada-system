package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func HtmlToMd(html string) string {
	html = regexp.MustCompile(`(?s)<script.*?</script>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`(?s)<style.*?</style>`).ReplaceAllString(html, "")
	
	// Convert <pre> to code blocks
	html = regexp.MustCompile(`(?s)<pre.*?>(.*?)</pre>`).ReplaceAllString(html, "```go\n$1\n```\n")
	
	// Convert headers
	headerRe := regexp.MustCompile(`(?si)<h([1-4]).*?>(.*?)</h[1-4]>`)
	html = headerRe.ReplaceAllStringFunc(html, func(m string) string {
		matches := headerRe.FindStringSubmatch(m)
		prefix := strings.Repeat("#", int(matches[1][0]-'0'))
		
		// strip inner tags
		text := regexp.MustCompile(`(?s)<.*?>`).ReplaceAllString(matches[2], "")
		return "\n" + prefix + " " + text + "\n"
	})
	
	// Links
	html = regexp.MustCompile(`(?si)<a.*?href="(.*?)".*?>(.*?)</a>`).ReplaceAllString(html, "[$2]($1)")
	
	// Inline code
	html = regexp.MustCompile(`(?si)<code.*?>(.*?)</code>`).ReplaceAllString(html, "`$1`")
	
	// Paragraphs
	html = regexp.MustCompile(`(?si)<p.*?>(.*?)</p>`).ReplaceAllString(html, "$1\n\n")
	
	// Strip all remaining tags
	html = regexp.MustCompile(`(?s)<.*?>`).ReplaceAllString(html, "")
	
	// Unescape basics
	html = strings.ReplaceAll(html, "&lt;", "<")
	html = strings.ReplaceAll(html, "&gt;", ">")
	html = strings.ReplaceAll(html, "&amp;", "&")
	html = strings.ReplaceAll(html, "&quot;", "\"")
	html = strings.ReplaceAll(html, "&#39;", "'")
	
	return html
}

func main() {
	outPath := "llm/design/effective_go.md"
	if _, err := os.Stat(outPath); err == nil {
		fmt.Println("File already exists, skipping fetch.")
		return
	}

	fmt.Println("Fetching Effective Go...")
	resp, err := http.Get("https://go.dev/doc/effective_go")
	if err != nil {
		fmt.Printf("Failed to fetch: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read body: %v\n", err)
		os.Exit(1)
	}

	md := HtmlToMd(string(body))

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		fmt.Printf("Failed to create dir: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outPath, []byte(md), 0644); err != nil {
		fmt.Printf("Failed to write file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully saved to", outPath)
}
