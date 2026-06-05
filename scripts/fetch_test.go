package main

import (
	"strings"
	"testing"
)

func TestHtmlToMd(t *testing.T) {
	input := `
		<html>
		<body>
			<h2 id="intro">Introduction</h2>
			<p>This is a <a href="https://go.dev">link</a> to Go.</p>
			<pre><code>fmt.Println("hello")</code></pre>
			<script>alert("hidden");</script>
		</body>
		</html>
	`
	expected := []string{
		"## Introduction",
		"[link](https://go.dev)",
		"```go\n`fmt.Println(\"hello\")`\n```",
	}
	
	result := HtmlToMd(input)
	
	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("Expected result to contain '%s', but got: \n%s", exp, result)
		}
	}
	
	if strings.Contains(result, "alert") {
		t.Errorf("Expected script tags to be stripped out")
	}
}
