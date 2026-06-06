# Bug Report: SHELL Env Override Masking Actual Login Shell

## Problem Statement
Despite parsing `/etc/passwd` in recent versions, `ads-shell` incorrectly prioritized the `$SHELL` environment variable over the native system's user config. When a terminal emulator (like Konsole or a graphical desktop session) explicitly overrides or inherits an incorrect `$SHELL` (often `/bin/bash`), `ads-shell` silently inherits it and skips parsing `/etc/passwd`. This results in users being persistently dropped into `bash` instead of their true login shell (e.g. `zsh`).

## Expected Behavior
1. `ads-shell` MUST treat the native OS user configuration (`getent passwd` or `/etc/passwd`) as the ultimate source of truth, taking absolute priority.
2. The `$SHELL` environment variable should only be used as a last-resort fallback if native resolution fails (e.g., containerized setups without standard user databases).

## Actions Taken
- Created isolated Git branch `fix/ads-shell-getent`.
- Reversed the resolution order in `getUserShell()` to prioritize `getent passwd` and `/etc/passwd`.
- Triggered patch bump to `v4.2.2`.
