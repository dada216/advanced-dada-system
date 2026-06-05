# Bug Report: Interactive Search Lacks Prefix Matching and Clipboard Support

## Description
The user reported that when typing in the interactive search pane, partial matches (e.g., `piri` for `piripi`) are not returning results in real-time. This is because SQLite FTS5 requires explicit prefix syntax (`*`) for partial token matching.
Additionally, the user expects that hitting `Enter` on a selected result will automatically copy that result into the clipboard so it can be immediately pasted into the active session.

## Expected Behavior
- The federated search engine should automatically append wildcard syntax (`"query"*`) to provide real-time prefix matching as the user types.
- Hitting `Enter` in the TUI should grab the selected snippet, strip any injected ANSI color codes, and invoke `tmux set-buffer` so the text is instantly available in the tmux clipboard (`Prefix + ]`).
- This fix will bump the semantic version to `v1.1.0`.
