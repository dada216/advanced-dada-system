package main

import (
	"fmt"
	"os"

	"github.com/advanced-dada-system/ads/internal/plugin"
	"github.com/spf13/cobra"
)

var llmCmd = &cobra.Command{
	Use:   "llm [subcommand]",
	Short: "AI analytics powered by OpenRouter",
}

var llmSummarizeCmd = &cobra.Command{
	Use:   "summarize [session-name]",
	Short: "Summarize a session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sessionName := args[0]
		prompt := fmt.Sprintf("Summarize the main events and any errors in the terminal session named %s.", sessionName)

		fmt.Printf("Analyzing session '%s' with OpenRouter...\n", sessionName)
		res, err := plugin.CallPlugin("ads-plugin-llm", map[string]string{"prompt": prompt})
		if err != nil {
			fmt.Fprintf(os.Stderr, "LLM Plugin Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n--- Summary ---")
		fmt.Println(res)
	},
}

var llmAskCmd = &cobra.Command{
	Use:   "ask [session-name] [question]",
	Short: "Ask a specific question about a session",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		sessionName := args[0]
		question := args[1]
		prompt := fmt.Sprintf("Regarding the terminal session named %s: %s", sessionName, question)

		fmt.Printf("Analyzing session '%s' with OpenRouter...\n", sessionName)
		res, err := plugin.CallPlugin("ads-plugin-llm", map[string]string{"prompt": prompt})
		if err != nil {
			fmt.Fprintf(os.Stderr, "LLM Plugin Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n--- Answer ---")
		fmt.Println(res)
	},
}

func init() {
	llmCmd.AddCommand(llmSummarizeCmd)
	llmCmd.AddCommand(llmAskCmd)
	rootCmd.AddCommand(llmCmd)
}
