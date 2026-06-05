# Bug Report: Search Results Flattening Multiple Lines into Wrong Order

## Description
The user reported that the search results pane was displaying the matching string and the prompt in the wrong order (e.g., `kubernetes root@k8s-master:~#` instead of `root@k8s-master:~# kubernetes`). 
This was caused by the FTS engine retrieving entire I/O chunks. Since a single TCP or pipe chunk can contain multiple consecutive lines of terminal output (e.g., the command output `kubernetes` followed by the next shell prompt `root@...`), the `cleanSnippet` function was stripping the newline boundaries and merging them into a single string. This created a visually confusing line where the prompt appeared *after* the output.

## Expected Behavior
- Refactor the `cleanSnippet` function in `cmd/ads/search_interactive.go` to split the FTS chunk into discrete lines using `\n`.
- Extract and display *only* the specific line that contains the `\033[31m` FTS highlight marker. 
- This ensures the UI presents a clean, isolated row of output without bleeding into the adjacent terminal prompts that happened to arrive in the same packet.
- This feature triggers a patch version bump to `v1.9.2`.
