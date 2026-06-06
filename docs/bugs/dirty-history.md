# Bug Report: Dirty Command History (Prompt Leaking into User Input)

## Problem Statement
The user requested an analysis of why the `command_history` table contained a "dirty" history. Upon writing a comprehensive PTY integration test suite (`internal/sessiondb/db_test.go`), it was discovered that the shell's literal prompt text (e.g., `bash-5.1$ `) was being permanently recorded into the SQLite database as part of the user's executed command text.

This occurred due to a semantic misunderstanding of the OSC 133 specification in `ads hook`:
The previous hook definitions injected `\033]133;B\007` (Start of User Input) at the *beginning* of `$PS1` (Bash) and `$PROMPT` (Zsh). Because `B` was emitted before the actual prompt text was rendered, the scanner (`ads-shell`) naturally assumed the shell's literal prompt string was actually part of the user's keystrokes, fatally poisoning the search index and LLM context.

## Expected Behavior
1. The `B` marker must denote the absolute end of the prompt and the absolute start of the user's input buffer.
2. The `A` marker denotes the start of the prompt.
3. Therefore, the prompt text must render strictly between `A` and `B`.

## Actions Taken
- Created isolated Git branch `fix/clean-command-history`.
- Authored `TestIntegrationCommandHistory` in `internal/sessiondb/db_test.go` to spawn a real PTY, attach Bash, execute the hook dynamically, run a command, and assert the resulting DB output. This serves as a permanent regression test.
- Modified `cmd/ads/hook.go` to append the `B` marker strictly at the end of the shell prompt string (`PS1="$PS1\[\033]133;B\007\]"`).
- Re-ran the integration test, verifying that the `command_history` extracted `echo my_secret_test_command` perfectly clean.
- Triggered patch version bump to `v4.8.1`.
