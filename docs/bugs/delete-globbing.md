# Feature Request: Globbing Support for Session Deletion

## Problem Statement
Users naturally accrue dozens of auto-generated sessions (`ads-<timestamp>`) during standard usage. Cleaning these up one by one via `ads delete <exact-name>` is tedious and anti-pattern for a CLI tool. The user requested `*` globbing support to wipe out matching sessions efficiently (e.g., `ads delete "ads-*"`).

## Expected Behavior
1. `ads delete <pattern>` should interpret the argument as a glob pattern.
2. It should fetch all sessions from the metadata database and run a `filepath.Match` against their names.
3. It should bulk delete the SQLite binaries and the metadata records for all matching sessions.
4. If no sessions match, it should exit gracefully with a warning instead of a hard error.

## Actions Taken
- Created isolated Git branch `feature/delete-globbing`.
- Refactored `deleteCmd` to use `filepath.Match` and bulk iteration over `db.ListSessions()`.
- Added support for both exact matching (since globbing naturally falls back to exact string match) and wildcards.
- Triggered minor version bump to `v4.6.0`.
