# Bug Report: Interactive Search Row Highlight Coverage and Brightness

## Description
The user reported two issues with the active row highlight in the search pane:
1. The light green highlight (`119`) was too bright and visually jarring.
2. The background highlight dropped out before reaching the right edge of the terminal.

The coverage drop-out occurs because internal Lipgloss styles and the FTS5 red highlight inject `\033[0m` (ANSI reset) sequences. These sequences wipe out all formatting, prematurely terminating the background color set by the parent row style before the terminal reaches the far right edge.

## Expected Behavior
- Change the background highlight to a darker, more muted green (ANSI `22`) to blend better with terminal aesthetics.
- Instead of relying on a Lipgloss parent style (which cannot safely traverse internal ANSI resets), post-process the selected line to explicitly intercept all internal `\033[0m` sequences and automatically re-apply the background color (`\033[48;5;22m`). This guarantees a continuous, unbroken color bar across the entire row.
- This tweak triggers a patch version bump to `v1.7.1`.
