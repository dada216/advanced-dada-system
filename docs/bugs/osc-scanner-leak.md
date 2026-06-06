# Bug Report: Empty Command History and OSC Scanner Memory Leak

## Problem Statement
The user reported that the `command_history` table in the SQLite session DB was completely empty. Upon architectural investigation, a fatal flaw was discovered in `OSCScanner.Write()`. 
The previous implementation relied on an overly simplified regex parser (`regexp.MustCompile`) combined with a naive `bytes.LastIndexByte` chunking logic for partial escape sequences. If the scanner encountered standard ANSI formatting codes (like `\x1b[0m` used in syntax highlighting) but NO `OSC 133` markers within the same chunk, it would "hold" the trailing escape sequence in the buffer. Because standard ANSI codes never satisfied the `OSC 133` regex, the buffer would perpetually loop and hold the data indefinitely, growing the stream infinitely and starving the `commandBuf`. 
As a result, commands were never cleanly extracted, and a severe memory leak occurred during highly-formatted terminal output (e.g., Zsh syntax highlighting, `ls --color`).

## Expected Behavior
1. The semantic parser MUST explicitly index `\x1b]133;` directly as a byte sequence rather than relying on expensive, global regular expressions.
2. The chunking logic must properly evaluate if trailing bytes are a mathematically valid *prefix* of the `\x1b]133;` sequence, rather than generically halting on any `\x1b` character.

## Actions Taken
- Created isolated Git branch `fix/osc-scanner-rewrite`.
- Designed a `mockInserter` test suite in `internal/ansi/scanner_test.go` that accurately reproduced the `\x1b[K` starvation bug and the memory leak.
- Completely rewrote `internal/ansi/scanner.go` to use a custom, O(N) byte indexer and prefix-matching logic for partial sequences.
- Triggered minor version bump to `v4.7.0`.
