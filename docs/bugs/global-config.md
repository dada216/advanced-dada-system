# Feature Request: Global Configuration and Default Shell state

## Problem Statement
While the `--shell` flag added in `v4.3.0` allowed overriding the OS identity (`/etc/passwd`), it was inherently stateless. A user creating a session manually via `ads run local0` from their shell would not inherit the `--shell zsh` parameter utilized by their Konsole GUI shortcut. Consequently, they would be unexpectedly dropped back into `bash`. This required users to painstakingly append `--shell zsh` to every manual invocation, violating CLI ergonomics.

## Expected Behavior
1. `ads` MUST support a persistent global configuration architecture.
2. An `ads config` command should allow getting/setting configuration properties.
3. The `ads-shell` PTY proxy must natively intercept `default_shell` from this configuration *before* falling back to `/etc/passwd`. 

## Actions Taken
- Created isolated Git branch `feature/global-config`.
- Implemented robust `config.json` schema via `internal/config/config.go`.
- Added the `ads config [key] [value]` CLI interface.
- Integrated `cfg.DefaultShell` directly into `ads-shell`'s resolution tree.
- Triggered minor version bump to `v4.8.0`.
