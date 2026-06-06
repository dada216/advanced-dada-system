# Bug Report: Zsh PROMPT Literal Byte Evaluation Failure

## Problem Statement
The user reported that the `command_history` table was still remaining completely empty, despite correctly running `ads hook install zsh` and restarting their terminal.

Upon rigorous manual PTY interrogation using a custom Go test harness (`test_zsh_pty.go`), it was discovered that Zsh was emitting the OSC 133 `B` marker into the terminal as a literal string (i.e. `\033]133;B\007`), rather than executing it as an invisible ANSI escape sequence (i.e. `\x1b]133;B\x07`).

Because Zsh's `$PROMPT` variable does not natively evaluate standard backslash escape codes (like `\033`) by default without special syntax, the literal `B` sequence never triggered the parser's state machine. Because `state=2` was never achieved, the `commandBuf` was never populated, resulting in an empty `cmdText`, silently bypassing the database `InsertCommand()` function.

## Expected Behavior
The `ads hook zsh` output must ensure the OSC 133 `B` sequence is evaluated by Zsh's interpreter as an actual escape byte (`\x1b`).

## Actions Taken
- Created isolated Git branch `fix/zsh-prompt-hook`.
- Corrected the Zsh `PROMPT` appending logic in `cmd/ads/hook.go`. The syntax `PROMPT="${PROMPT}%{\033]133;B\007%}"` was replaced with the Bash-compatible ANSI-C quoting syntax `PROMPT=$PROMPT$'%{\e]133;B\a%}'`, which forces Zsh to evaluate the escape bytes natively.
- Triggered patch version bump to `v4.10.1`.
