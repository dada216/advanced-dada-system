# Bug Report: Zsh Theme Prompt Overwrite

## Problem Statement
The user reported that `command_history` was still empty despite the previous fix for literal byte evaluation (`v4.10.1`). 

Upon investigation using an integration test harness (`test_zsh_theme.go`), it was discovered that advanced Zsh themes (such as `starship`, `powerlevel10k`, `spaceship`) dynamically reconstruct and overwrite the `$PROMPT` variable during the Zsh `precmd` hook execution on every prompt cycle.

Because `ads hook zsh` simply appended the OSC 133 `B` marker to the end of the user's `.zshrc`, the static evaluation occurred during shell initialization. Once the interactive shell started and the user executed a command, the theme's own `precmd` logic completely obliterated the `B` marker, resetting the prompt to its custom theme string. 

Without the `B` marker (Start of User Input), the `ads-shell` PTY proxy scanner never transitioned into the keystroke capturing state, resulting in a completely silent failure.

## Expected Behavior
The OSC 133 `B` marker must be guaranteed to exist at the end of the final rendered `$PROMPT` before user input begins, regardless of what Zsh themes are active.

## Actions Taken
- Created isolated Git branch `fix/zsh-theme-hook`.
- Removed the static `$PROMPT` append logic from the Zsh hook payload.
- Moved the `B` marker injection logic *directly inside* the `_ads_precmd` function itself. Since `ads hook install zsh` appends `eval "$(ads hook zsh)"` to the end of `.zshrc`, `add-zsh-hook precmd _ads_precmd` is guaranteed to be registered last, ensuring `_ads_precmd` executes *after* any theme's prompt logic.
- Included an idempotency check (`if [[ "$PROMPT" != *$'%{\e]133;B\a%}'* ]]`) to avoid infinitely appending the marker on every cycle.
- Bumped version to `v4.10.2`.
