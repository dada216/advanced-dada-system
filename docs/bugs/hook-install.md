# Feature Request: Automated Hook Installation for Shells

## Problem Statement
The user reported that the `command_history` table in the SQLite database remained completely empty during normal usage. Upon architectural analysis, the root cause was discovered: the user had not manually added the `eval "$(ads hook zsh)"` script to their `~/.zshrc`. 
Because `ads-shell` functions purely as an invisible PTY proxy, it is mathematically impossible for it to guess the boundaries of user input without the active cooperation of the target shell (via OSC 133 semantic markers). Because the shell wasn't configured, no markers were emitted, and the scanner state machine remained locked at state `0`, successfully starving the `command_history` engine.

## Expected Behavior
1. Users should not be expected to manually copy-paste shell evaluation strings.
2. `ads` MUST provide an automated configuration mechanism (similar to `starship` or `direnv`).

## Actions Taken
- Created isolated Git branch `feature/hook-install`.
- Extended the `ads hook` command tree with a new `install [shell]` subcommand.
- Configured dynamic `os.OpenFile(..., os.O_APPEND)` logic to safely append the correct evaluation string to `~/.bashrc` or `~/.zshrc`.
- Triggered minor version bump to `v4.10.0`.
