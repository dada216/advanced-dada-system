package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "v4.6.0"
var Codename = "Pangolin"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the current ADS version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Advanced Dada System (ADS) version %s \"%s\"\n", Version, Codename)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
