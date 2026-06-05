# Feature Report: Tmux-Native Color Scheme

## Description
The user reported that the initial bubblegum/pink color scheme for the interactive search pane clashed with standard terminal aesthetics. They requested a color palette that perfectly mimics the default Tmux styles (which typically rely heavily on greens, blacks, yellows, and classic terminal grays) to make the TUI feel like a native extension of Tmux.

## Expected Behavior
- Update the Lipgloss styles in `cmd/ads/search_interactive.go` to use a classic Tmux palette.
- The `titleStyle` will mimic the default Tmux status bar (Green background, Black text).
- The `cursorStyle` will use a bright terminal green.
- The `sessionStyle` will use a classic terminal yellow.
- This aesthetic overhaul triggers a version bump to `v1.5.0`.
