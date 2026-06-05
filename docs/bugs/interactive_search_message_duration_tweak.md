# Bug Report: Tmux Display Message Duration Tweak

## Description
The user reported that the 3-second display duration for the Tmux feedback toast was slightly too long and preferred a 2-second duration for optimal pacing.

## Expected Behavior
- Modify the `-d 3000` flag to `-d 2000` in the `tmux display-message` invocation.
- Bump the release version to `v1.4.1`.
