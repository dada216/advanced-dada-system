# Bug Report: Konsole Launch Quoting & Login Shell Regression

## Problem Statement
1. The Konsole profile generator embedded a complex `sh -c` initialization string. Konsole's internal parser incorrectly mangles the single and double quotes, causing evaluation errors (`accepts 1 arg(s), received 0`) if a user copies or references the command in different shells.
2. The previous fix forcing `ads-shell` to spawn a login shell (`-zsh` / `-bash`) inadvertently broke environments for users who rely on interactive *non-login* shells to bootstrap their configurations (e.g. `exec zsh` via `~/.bashrc`). Since login shells skip `.bashrc`, they were incorrectly dropped into a vanilla bash session despite `/etc/passwd` resolution.

## Expected Behavior
1. `ads` should offer a native `launch` command to automate session naming and execution, removing the need for error-prone `sh -c` string escaping in the KDE profile.
2. `ads-shell` should launch an interactive non-login shell by default, accurately mirroring the exact default behavior of Linux terminal emulators like Konsole and GNOME Terminal.

## Actions Taken
- Created isolated Git branch `fix/konsole-launch-quoting`.
- Added the `ads launch` subcommand to handle atomic creation and initialization of a random timestamped session.
- Reverted the `-` prefix on `argv[0]` inside `ads-shell`, restoring standard non-login interactive execution environments.
- Triggered patch/minor bump to `v4.2.1`.
