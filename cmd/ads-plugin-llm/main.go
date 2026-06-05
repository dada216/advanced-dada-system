package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/advanced-dada-system/ads/internal/config"
	"github.com/advanced-dada-system/ads/internal/plugin"
	hashicorpplugin "github.com/hashicorp/go-plugin"
)

type LLMPlugin struct{}

type openRouterReq struct {
	Model    string `json:"model"`
	Messages []msg  `json:"messages"`
}

type msg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (s *LLMPlugin) RunTask(args map[string]string) (string, error) {
	prompt, ok := args["prompt"]
	if !ok {
		return "", fmt.Errorf("missing 'prompt' argument")
	}

	apiKey := os.Getenv("ADS_OPENROUTER_KEY")
	if apiKey == "" {
		appDir, err := config.InitAppDataDir()
		if err == nil {
			keyPath := filepath.Join(appDir, "secrets", "openrouter.key")
			if b, err := os.ReadFile(keyPath); err == nil {
				apiKey = string(bytes.TrimSpace(b))
			}
		}
	}

	if apiKey == "" {
		return "", fmt.Errorf("missing OpenRouter API key. Set ADS_OPENROUTER_KEY env var or save it in ~/.local/share/ads/secrets/openrouter.key")
	}

	reqBody := openRouterReq{
		Model: "openai/gpt-3.5-turbo",
		Messages: []msg{
			{Role: "system", Content: "You are an expert Linux system administrator analyzing terminal session output."},
			{Role: "user", Content: prompt},
		},
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(b))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/advanced-dada-system/ads")
	req.Header.Set("X-Title", "Advanced Dada System")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("no choices returned from API")
	}

	firstChoice := choices[0].(map[string]interface{})
	message := firstChoice["message"].(map[string]interface{})
	content := message["content"].(string)

	return content, nil
}

func main() {
	hashicorpplugin.Serve(&hashicorpplugin.ServeConfig{
		HandshakeConfig: plugin.HandshakeConfig,
		Plugins: map[string]hashicorpplugin.Plugin{
			"service": &plugin.ServicePlugin{Impl: &LLMPlugin{}},
		},
	})
}
