package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/advanced-dada-system/ads/internal/meta"
	"github.com/spf13/cobra"
)

var pluginRootCmd = &cobra.Command{
	Use:   "plugin [subcommand]",
	Short: "Manage plugin configurations",
}

var pluginEditCmd = &cobra.Command{
	Use:   "edit [plugin-name]",
	Short: "Edit configuration for a specific plugin",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pluginName := args[0]
		if pluginName != "llm" {
			fmt.Printf("Editing configuration for plugin '%s' is not supported yet.\n", pluginName)
			return
		}

		db, err := meta.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening DB: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		currentKey, _ := db.GetPluginConfig("llm", "api_key")
		currentModel, _ := db.GetPluginConfig("llm", "model")

		reader := bufio.NewReader(os.Stdin)

		maskedKey := "None"
		if currentKey != "" {
			maskedKey = currentKey[:4] + "..." + currentKey[len(currentKey)-4:]
		}

		fmt.Printf("OpenRouter API Key [%s]: ", maskedKey)
		keyInput, _ := reader.ReadString('\n')
		keyInput = strings.TrimSpace(keyInput)
		if keyInput == "" {
			keyInput = currentKey
		}

		if keyInput == "" {
			fmt.Println("API Key is required to fetch models.")
			os.Exit(1)
		}

		fmt.Println("Fetching available models from OpenRouter...")
		req, _ := http.NewRequest("GET", "https://openrouter.ai/api/v1/models", nil)
		req.Header.Set("Authorization", "Bearer "+keyInput)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			fmt.Printf("Failed to fetch models: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var result struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Printf("Failed to parse models: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\nAvailable Models (showing first 20):")
		max := 20
		if len(result.Data) < max {
			max = len(result.Data)
		}
		for i := 0; i < max; i++ {
			fmt.Printf("  %d. %s\n", i+1, result.Data[i].ID)
		}

		modelDisplay := "openai/gpt-3.5-turbo"
		if currentModel != "" {
			modelDisplay = currentModel
		}

		fmt.Printf("\nSelected Model [%s]: ", modelDisplay)
		modelInput, _ := reader.ReadString('\n')
		modelInput = strings.TrimSpace(modelInput)
		if modelInput == "" {
			modelInput = modelDisplay
		}

		_ = db.SetPluginConfig("llm", "api_key", keyInput)
		_ = db.SetPluginConfig("llm", "model", modelInput)

		fmt.Println("Plugin configuration saved successfully.")
	},
}

var pluginListCmd = &cobra.Command{
	Use:   "list [plugin-name]",
	Short: "List configurations for a specific plugin",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pluginName := args[0]
		if pluginName != "llm" {
			fmt.Printf("Listing configuration for plugin '%s' is not supported yet.\n", pluginName)
			return
		}

		db, err := meta.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening DB: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		currentKey, _ := db.GetPluginConfig("llm", "api_key")
		currentModel, _ := db.GetPluginConfig("llm", "model")

		fmt.Printf("--- Configuration for Plugin: %s ---\n", pluginName)
		
		maskedKey := "None"
		if currentKey != "" && len(currentKey) > 8 {
			maskedKey = currentKey[:4] + "..." + currentKey[len(currentKey)-4:]
		}
		fmt.Printf("API Key: %s\n", maskedKey)
		
		modelDisplay := "None (defaults to openai/gpt-3.5-turbo)"
		if currentModel != "" {
			modelDisplay = currentModel
		}
		fmt.Printf("Model: %s\n", modelDisplay)
	},
}

func init() {
	pluginRootCmd.AddCommand(pluginEditCmd)
	pluginRootCmd.AddCommand(pluginListCmd)
	rootCmd.AddCommand(pluginRootCmd)
}
