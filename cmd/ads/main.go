package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ads",
	Short: "Advanced Dada System - Terminal Analytics Platform",
	Long:  "ads is a CLI tool to manage and analyze recorded terminal sessions.",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
