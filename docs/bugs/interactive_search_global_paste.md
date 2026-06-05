# Feature Report: Global Paste Keybinding

## Description
The user requested that the Tmux buffer paste functionality (which defaults to `Prefix + ]`) be mapped globally, removing the prefix requirement just like the search pane shortcut. 

## Expected Behavior
- Update `internal/meta/db.go` to add a global paste keybind.
- We map this to `Ctrl + ]` (`bind-key -n C-] paste-buffer`). We *must not* map it to literally just `]` (`bind-key -n ]`), because doing so would completely intercept the closing bracket character, making it impossible for the user to type `]` in their terminal, bash scripts, or code editors.
- This feature triggers a minor version bump to `v1.9.0`.
