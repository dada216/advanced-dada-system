# Bug Report: Shell Login Environment & Fallback

## Problem Statement
Users report that `ads-shell` launches a non-login instance of their shell (e.g. `bash` or `zsh`), causing profile scripts like `~/.zprofile` or `~/.bash_profile` to be skipped. This results in missing aliases, paths, and environment variables when opening a new ADS Konsole profile. Furthermore, if `os.Getenv("SHELL")` is empty or incorrectly inherited by KDE/Konsole, it falls back to `bash` rather than accurately determining the user's login shell from `/etc/passwd`.

## Expected Behavior
1. `ads-shell` should always launch local shells as **login shells** (by prepending a `-` to `argv[0]`).
2. If `SHELL` is unset or empty, `ads-shell` should parse `/etc/passwd` to correctly determine the user's true configured shell.

## Actions Taken
- Created isolated Git branch `fix/zsh-login-shell`.
- Minor version bump to `v4.2.0`.
- Added `/etc/passwd` fallback logic and modified `exec.Command`'s `Args[0]` to initiate a login shell.
