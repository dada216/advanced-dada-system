package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var hookCmd = &cobra.Command{
	Use:   "hook [shell]",
	Short: "Generate shell integration scripts for OSC 133 markers",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		shell := args[0]
		if shell == "bash" {
			fmt.Printf("%s\n", `
_ads_prompt_command() {
    local exit_code=$?
    printf "\033]133;D;%s\007" "$exit_code"
    printf "\033]133;A\007"
}
PROMPT_COMMAND="_ads_prompt_command; $PROMPT_COMMAND"
PS1="$PS1\[\033]133;B\007\]"
PS0="\033]133;C\007"
`)
		} else if shell == "zsh" {
			fmt.Printf("%s\n", `
_ads_precmd() {
    local exit_code=$?
    printf "\033]133;D;%s\007" "$exit_code"
    printf "\033]133;A\007"
    if [[ "$PROMPT" != *$'%{\e]133;B\a%}'* ]]; then
        PROMPT=$PROMPT$'%{\e]133;B\a%}'
    fi
}
_ads_preexec() {
    printf "\033]133;C\007"
}

autoload -Uz add-zsh-hook
add-zsh-hook precmd _ads_precmd
add-zsh-hook preexec _ads_preexec
`)
		} else {
			fmt.Printf("Shell '%s' is not supported yet.\n", shell)
		}
	},
}

var hookInstallCmd = &cobra.Command{
	Use:   "install [shell]",
	Short: "Automatically append the hook source command to your .bashrc or .zshrc",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		shell := args[0]
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}

		var rcFile string
		var hookStr string
		if shell == "bash" {
			rcFile = homeDir + "/.bashrc"
			hookStr = "\neval \"$(ads hook bash)\"\n"
		} else if shell == "zsh" {
			rcFile = homeDir + "/.zshrc"
			hookStr = "\neval \"$(ads hook zsh)\"\n"
		} else {
			fmt.Fprintf(os.Stderr, "Shell '%s' is not supported for auto-installation.\n", shell)
			os.Exit(1)
		}

		f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", rcFile, err)
			os.Exit(1)
		}
		defer f.Close()

		if _, err := f.WriteString(hookStr); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to %s: %v\n", rcFile, err)
			os.Exit(1)
		}

		fmt.Printf("Successfully injected ADS hooks into %s!\n", rcFile)
		fmt.Printf("Please restart your terminal or run: source %s\n", rcFile)
	},
}

func init() {
	hookCmd.AddCommand(hookInstallCmd)
	rootCmd.AddCommand(hookCmd)
}
