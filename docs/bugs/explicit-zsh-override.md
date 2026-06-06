# Bug Report: System OS Identity vs Desired User Shell

## Problem Statement
Despite the rigorous parsing of `/etc/passwd` to definitively determine the system's login shell, it turns out that some users explicitly desire to use a completely different shell (like `zsh`) for their ADS sessions, even though their primary OS identity in `/etc/passwd` strictly dictates `bash`. The current architecture strictly enforces `/etc/passwd` identity, making it impossible to launch an alternative shell natively without completely altering the host OS account configuration via `chsh`.

## Expected Behavior
1. `ads-shell`, `ads run`, and `ads launch` MUST accept a `--shell <string>` flag to allow explicit terminal emulator overrides, bypassing OS-level identity checks.
2. The KDE Konsole profile generation script (`install-konsole.sh`) should explicitly append `--shell zsh` as formally requested by the user, explicitly injecting their customized workflow without breaking default functionality for others.

## Actions Taken
- Created isolated Git branch `fix/explicit-zsh-override`.
- Added the `--shell` flag across the entire CLI architecture.
- Appended `--shell zsh` to the Konsole profile shortcut generation.
- Triggered minor bump to `v4.3.0` due to new CLI flags/features.
