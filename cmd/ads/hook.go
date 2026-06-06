package main

import (
	"fmt"

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
PS1="\[\033]133;B\007\]$PS1"
PS0="\033]133;C\007"
`)
		} else if shell == "zsh" {
			fmt.Printf("%s\n", `
_ads_precmd() {
    local exit_code=$?
    printf "\033]133;D;%s\007" "$exit_code"
    printf "\033]133;A\007"
}
_ads_preexec() {
    printf "\033]133;C\007"
}

autoload -Uz add-zsh-hook
add-zsh-hook precmd _ads_precmd
add-zsh-hook preexec _ads_preexec

PROMPT="%{\033]133;B\007%}${PROMPT}"
`)
		} else {
			fmt.Printf("Shell '%s' is not supported yet.\n", shell)
		}
	},
}

func init() {
	rootCmd.AddCommand(hookCmd)
}
