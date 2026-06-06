# Bug Report: ZLE Redraw Wiping Command Buffer

## Problem Statement
The user reported that `command_history` was still empty despite the previous prompt injection fixes.

Analysis revealed a deeply ingrained edge case with the Zsh Line Editor (ZLE). Advanced plugins (e.g., `zsh-autosuggestions`, `fast-syntax-highlighting`, or transient prompt features in `starship`/`powerlevel10k`) often trigger a full prompt redraw the exact moment the user presses `Enter` to accept a line.

When ZLE redraws the prompt, it re-emits the OSC 133 `B` marker. By design, the `ads-shell` PTY proxy resets its keystroke buffer (`commandBuf`) every time it encounters a `B` marker (to prevent duplicating keystrokes during window resizes). However, because ZLE accepts the line and does *not* reprint the user's keystrokes after this final redraw, the buffer remained empty right before the `preexec` hook triggered the `C` marker (Execution Start). 

This resulted in `ads-shell` inserting an empty string into the `command_history` database.

## Expected Behavior
The exact string typed by the user must be captured and persisted to `command_history`, immune to ZLE redraws or PTY timing issues.

## Actions Taken
- Created isolated Git branch `fix/zsh-cmd-text-marker`.
- Leveraged a non-standard but highly robust extension of the OSC 133 specification: encoding the command text directly into the `C` marker's payload.
- Modified `internal/ansi/scanner.go` to parse `OSC 133;C;<command_text>\007`. If the command text is present, it explicitly overrides the `commandBuf`, entirely bypassing the flawed PTY heuristic.
- Updated `cmd/ads/hook.go` to emit `printf "\033]133;C;%s\007" "$1"` in the Zsh `preexec` hook. (Bash continues to use the standard heuristic).
- Bumped version to `v4.10.3`.
