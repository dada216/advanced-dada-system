# Bug Report: Tmux Display Message Duration Too Short

## Description
The user reported a minor bug where the `tmux display-message` feedback (which appears after copying a snippet from the interactive search pane) disappears too quickly, appearing only as a brief flash.

## Expected Behavior
- The `tmux display-message` command should be invoked with the `-d` flag set to a higher duration (e.g., `3000` milliseconds) to ensure the message remains visible long enough for the user to comfortably read it.
- This minor bugfix will trigger a version bump from `v1.3.0` to `v1.4.0`.
