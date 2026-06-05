# Feature/Bug Report: Interactive Search Pane

## Description
The user reported that the current search results (piped into `less -R`) are difficult to read and visually unappealing. They requested an interactive, full-text search pane that opens "on top of the existing one".

## Proposed Solution
Instead of relying on external tools or raw command prompts, we will build a native Go Terminal UI (TUI) directly into the `ads` binary using the `bubbletea` framework. 
We will then update the `Prefix + s` Tmux binding to utilize `tmux display-popup`, which natively renders our Go TUI in a beautiful floating window perfectly centered over the user's active session.
