# Bug Report: Interactive Search Results Layout and Navigation

## Description
The user reported that the `ads search-interactive` pane renders text horribly when the underlying FTS5 snippet contains raw newline (`\n` or `\r`) characters from the terminal capture. This breaks the UI layout, causing multi-line staggering.
Furthermore, the user requested that the layout mirror a `zsh` history search (like `fzf`), requiring cleanly formatted single-line outputs and interactive selection/navigation capabilities.

## Expected Behavior
- Newlines in snippets should be stripped and replaced with spaces.
- The results should be displayed in a cleanly formatted, selectable list rather than raw appended strings.
- We should have an integration test simulating Tmux interactive keypresses to prove real-time federated updates across multiple sessions.
