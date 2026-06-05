# Feature Report: Interactive Search Row Highlighting and Copy Feedback

## Description
The user requested two usability enhancements for the Interactive Search pane:
1. **Full-Row Highlighting**: When navigating with arrow keys, the highlighted selection should stretch across the entire width of the terminal pane, rather than abruptly cutting off at the end of the text string.
2. **Tmux Display Feedback**: When a user hits `Enter` to copy a command to the clipboard, a brief, non-intrusive popup message should appear verifying what was copied. This message should not appear if the user exits via `Esc` or `Ctrl+C`.

## Expected Behavior
- Utilize `lipgloss` `.Width(m.width)` bounds to dynamically pad the selected list item to the maximum terminal width, creating a solid background bar across the screen.
- On `tea.KeyEnter`, trigger a native `tmux display-message` hook that flashes the copied snippet in the host's Tmux status line after the popup successfully closes.
- As requested by the user, this represents a feature milestone and prompts a minor version bump from `v1.2.0` to `v1.3.0`.
