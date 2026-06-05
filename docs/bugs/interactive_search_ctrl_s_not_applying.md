# Bug Report: Global Keybind Not Applying to Active Sessions

## Description
The user reported that the `v1.8.0` change to `Ctrl+S` (`bind-key -n C-s`) was not taking effect, and they still had to use the old `Prefix + s` shortcut. 

This occurred because `ads run` was only generating and sourcing the Tmux configuration template (`tmux.conf`) when *creating* a new detached Tmux session. If a session already existed and was alive in the background, `ads run` would simply attach to it without re-sourcing the config. Thus, the active Tmux server retained the old `Prefix + s` configuration in memory.

## Expected Behavior
- Extract the Tmux profile rendering and `source-file` execution logic in `internal/orchestrator/run.go` so that it runs universally, regardless of whether the session is new or already exists.
- This ensures that every time a user runs `ads run <name>`, the latest configuration from the database is forcefully hot-reloaded into the running Tmux server before attaching.
- This triggers a patch version bump to `v1.8.1`.
