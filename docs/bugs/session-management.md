# Feature: Session List and Rename Improvements

## Problem Statement
The user requested better session management capabilities. Specifically:
1. The `ads list` command does not distinguish between the currently active session and other sessions.
2. Randomly generated sessions (created via `ads launch`) clutter the main list of manually named sessions.
3. Users lack a native way to rename a randomly generated session (or any session) once they are inside it.

## Expected Behavior
1. `ads list` should mark the current session (using `$ADS_SESSION`) clearly.
2. `ads list` should separate auto-generated "unnamed" sessions (typically prefixed with `ads-`) into a distinct table.
3. A new command `ads rename <new-name>` should allow renaming the currently active session on the fly.

## Actions Taken
- Created isolated Git branch `feature/session-management`.
- Added `RenameSession` to `internal/meta/session.go`.
- Added `ads rename` command.
- Refactored `ads list` to parse and format sessions dynamically.
- Bumped version to `v4.4.0`.
