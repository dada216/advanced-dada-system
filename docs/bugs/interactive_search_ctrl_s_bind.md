# Feature Report: Tmux Global Search Keybinding

## Description
The user requested a way to launch the interactive search pane without needing to press the Tmux prefix key (`Ctrl + B`) beforehand. They specifically asked to map the action directly to `Ctrl + S`.

## Expected Behavior
- Update the default session configuration template in `internal/meta/db.go` to use the `-n` (no prefix) flag for the Tmux keybinding.
- Change the bound key from `s` to `C-s` (`Ctrl + S`), allowing immediate popup invocation.
- This feature triggers a minor version bump to `v1.8.0`.
