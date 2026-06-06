# Bug Report: Zsh Support / Hardcoded Bash

## Problem Statement
The current implementation of the `ads-shell` PTY proxy and Konsole integration hardcodes `bash` as the shell environment. This completely breaks the experience for users who use `zsh` or any other custom shell as their primary login shell on their desktop. Furthermore, `ads hook` only supports outputting `bash` integration scripts, leaving `zsh` users without OSC 133 semantic chunking markers.

## Expected Behavior
1. The `ads-shell` proxy should respect the user's default shell (via the `$SHELL` environment variable) instead of forcing `bash`. If `$SHELL` is not set, it should fallback to `/bin/sh` or `bash`.
2. The `install-konsole.sh` script should use a generic `sh -c` or `$SHELL -c` for the initialization command, or just directly invoke the `ads` binary without hardcoding `bash`.
3. `ads hook zsh` should output a valid zsh configuration snippet that correctly integrates `precmd` and `preexec` hooks to inject OSC 133 terminal markers.

## Actions Taken
- Created isolated Git branch `fix/zsh-support`.
- This bug report triggers a MINOR version bump to `v4.1.0` according to the versioning policies.
