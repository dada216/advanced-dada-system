# Bug Report: Tmux Scroll Behavior and Configurable History

## Description
The user reported that the scroll behavior in the default tmux session is not adherent to requirements. Specifically:
1. The mouse wheel should scroll backwards naturally. The current custom `WheelUpPane` and `WheelDownPane` bindings in `defaultProfile` interfere with tmux's native scrolling and copy-mode behavior.
2. The scrollback history should be infinite but configurable. The current hardcoded limit is `100000`, and there is no CLI way to edit the raw `config_text` for a profile to change this limit.

## Proposed Fix
1. Remove the custom `WheelUpPane` and `WheelDownPane` bindings from `defaultProfile` in `internal/meta/db.go`. Tmux version > 2.1 natively handles scrolling perfectly with just `set -g mouse on`.
2. Increase the default `history-limit` to `999999999` (practically infinite).
3. Implement a new CLI command `ads profile edit-config <name>` that fetches the `config_text` of a profile, opens it in the user's `$EDITOR` (or `nano`/`vi` fallback) for modification, and saves the changes back to the database. This satisfies the "configurable" requirement.
