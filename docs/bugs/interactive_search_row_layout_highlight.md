# Feature Report: Interactive Search Row Layout and Highlight Overhaul

## Description
The user reported that the search result layout was suboptimal. Specifically, they requested:
1. The line number (`row X`) should be moved to the far right edge of the terminal row.
2. The left side should cleanly display the session name and the matching string.
3. The matching string should return the *entire* text line with the match highlighted, rather than a truncated 10-token `snippet()`.

## Expected Behavior
- Update the FTS5 query in `internal/search/engine.go` to use the `highlight()` function instead of `snippet()`. This guarantees the entire line of text is returned with the matching tokens wrapped in ANSI color codes.
- Update `cmd/ads/search_interactive.go` to split the row rendering. The left side will contain the cursor, session name, and matched line, while the right side will flush the `row X` indicator against the right terminal bound using Lipgloss width calculations.
- This UI layout overhaul triggers a version bump to `v1.6.0`.
