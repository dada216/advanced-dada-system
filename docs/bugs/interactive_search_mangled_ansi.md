# Bug Report: Mangled Pastes and Search Results due to Bracketed Paste Mode

## Description
The user reported that copying text from search results out of a remote session injected extra junk characters like `2004l` and `2004h` into the pasted text (e.g. `2004lkubernetes 2004hroot@k8s-master:~#`).

This occurred because the `ads-recorder` ANSI stripping regex was insufficiently broad. It stripped standard ANSI colors (`\x1b[31m`), but failed to strip ECMA-48 CSI sequences containing intermediate bytes like `?`, which includes `\x1b[?2004h` and `\x1b[?2004l` (the terminal sequences for Bracketed Paste Mode). 
When the user pasted text into their terminal, these invisible sequences were intercepted by the recorder and written into the SQLite Full-Text Search `fts_index` database.
When retrieved via the interactive search and subsequently copied to the Tmux buffer, Tmux injected the raw sequence `\x1b[?2004l` into the host shell. The shell (Readline) failed to parse it, dropped the `\x1b[?` portion, and dumped the literal characters `2004l` onto the command line.

## Expected Behavior
- Broaden the ANSI CSI stripping regex in `internal/ansi/strip.go` to `\x1b\[[0-9;?]*[a-zA-Z]` so it properly intercepts bracketed paste and other advanced sequences for all future recordings.
- Patch `cmd/ads/search_interactive.go` to aggressively apply `ansi.Strip()` to the snippet immediately before writing to the Tmux buffer. This retroactively fixes any existing polluted data from being pasted into the shell.
- This feature triggers a patch version bump to `v1.9.1`.
