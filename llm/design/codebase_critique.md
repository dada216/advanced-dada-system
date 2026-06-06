# ADS Codebase Critique & Improvement Roadmap

## Overview

I've read every design document and every Go source file. The project is at **v3.3.1** with ~2,200 lines of Go across 4 binaries and 7 internal packages. The overall Unix-philosophy approach — `tmux pipe-pane` for capture, native `ssh`, database-per-session SQLite — is genuinely solid. The architecture avoids the usual trap of reimplementing terminal emulation. What follows is honest critique.

---

## 🟢 What's Good

1. **Architecture is well-chosen.** `tmux pipe-pane` + native `ssh` is correct. No PTY wrapper, no VT state machine, no daemon. This is the kind of design that survives contact with reality.

2. **Database-per-session is right.** Isolation, trivial deletion, no contention between recorders. WAL mode + `PRAGMA synchronous=NORMAL` is the right pragmatic call.

3. **The FTS5 dual-write pipeline works.** Raw BLOB in `io_stream` for replay, ANSI-stripped text in `fts_index` for search. Clean separation of concerns.

4. **HashiCorp go-plugin for extensibility.** Using multi-process gRPC plugins for search and LLM means the core recorder loop is crash-isolated from analytical workloads. Good call.

5. **OSC 133 scanner is well-designed.** State machine approach with graceful degradation. The `CommandInserter` interface decouples it from the DB nicely.

6. **Interactive TUI search** via Bubbletea is a nice touch for UX. Integrating it into tmux via `display-popup` bound to `C-s` is clever.

---

## 🔴 Critical Issues

### 1. The `cmd/ads/main.go` God File (302 lines, growing)

The main orchestrator entry point has **9 commands defined across 6 files** in package `main`, with scattered `init()` functions registering commands. This is the classic cobra anti-pattern.

**Problem**: Every new feature adds another file to `cmd/ads/`, all in the same `main` package, sharing global state (`rootCmd`, `isRemote`, `profileName`, `updateProfileName`). This doesn't scale.

**Fix**: Move each cobra command tree into its own package under `internal/cmd/` (e.g., `internal/cmd/session`, `internal/cmd/search`, `internal/cmd/llm`), with each exporting a single `func NewCmd() *cobra.Command`. The `main.go` becomes a pure assembler:

```go
func main() {
    root := &cobra.Command{Use: "ads"}
    root.AddCommand(session.NewCmd(), search.NewCmd(), llm.NewCmd(), ...)
    root.Execute()
}
```

### 2. Error Detection by String Matching

In [main.go L134](file:///app/projects/advanced-dada-system/cmd/ads/main.go#L134):
```go
if strings.Contains(err.Error(), "not found") {
```

And in [edit.go L30](file:///app/projects/advanced-dada-system/cmd/ads/edit.go#L30):
```go
if strings.Contains(err.Error(), "not found") {
```

**Problem**: This is fragile. If any upstream dependency changes error message wording, the auto-create flow silently breaks. This is one refactor away from a regression.

**Fix**: Define sentinel errors in `internal/meta`:
```go
var ErrSessionNotFound = errors.New("session not found")
var ErrProfileNotFound = errors.New("profile not found")
```
Then use `errors.Is()` at call sites.

### 3. The LLM Plugin Bypasses the `meta` Package Entirely

In [ads-plugin-llm/main.go L43-56](file:///app/projects/advanced-dada-system/cmd/ads-plugin-llm/main.go#L43-L56), the LLM plugin directly opens `meta.db` using raw `sql.Open()` and hand-crafts SQL queries to read `plugin_configs`. It completely bypasses `internal/meta.DB`.

**Problem**: Schema changes in `meta` will break the plugin silently. You now have two independent SQL consumers of the same schema with no shared contract.

**Fix**: Either pass the API key and model as arguments via the gRPC `RunTask(args)` call (the host already has access to `meta.DB`), or extract a shared read-only config accessor.

### 4. Unsafe JSON Decoding in LLM Plugin

In [ads-plugin-llm/main.go L118-120](file:///app/projects/advanced-dada-system/cmd/ads-plugin-llm/main.go#L118-L120):
```go
firstChoice := choices[0].(map[string]interface{})
message := firstChoice["message"].(map[string]interface{})
content := message["content"].(string)
```

**Problem**: Three unchecked type assertions in a row. If OpenRouter changes its response shape, or returns an error object, this panics — crashing the plugin process.

**Fix**: Use proper typed structs for the response:
```go
type openRouterResp struct {
    Choices []struct {
        Message struct {
            Content string `json:"content"`
        } `json:"message"`
    } `json:"choices"`
}
```

### 5. No Signal Handling in the Recorder

[ads-recorder/main.go](file:///app/projects/advanced-dada-system/cmd/ads-recorder/main.go) has no `SIGTERM`/`SIGINT` handler. When tmux kills the pipe-pane, the recorder may die mid-transaction.

**Problem**: Although WAL mode helps, abrupt termination without flushing the `OSCScanner` state means the last command in `command_history` may be lost, and the stream buffer may have unpersisted data.

**Fix**: Trap `SIGTERM` to break the read loop cleanly, flush the scanner, close the DB gracefully.

---

## 🟡 Structural Concerns

### 6. The Search Engine Is Sequential, Not Concurrent

[engine.go L50-83](file:///app/projects/advanced-dada-system/internal/search/engine.go#L50-L83) iterates sessions **sequentially**. The architecture doc explicitly calls for "bounded, concurrent worker goroutines." With many sessions, this becomes a bottleneck.

**Fix**: Use a bounded worker pool (`sync.WaitGroup` + semaphore channel) to query sessions concurrently, then aggregate results.

### 7. `config.InitAppDataDir()` Creates Directories as Side Effect

Every call to `InitAppDataDir()` does `os.MkdirAll()`. This means **reading config also writes to the filesystem**. Functions called `Init*` that are used as getters are confusing.

**Fix**: Split into `AppDataDir() string` (pure getter, reads `XDG_DATA_HOME`) and `EnsureAppDataDir() error` (called once at startup). Most callers only need the path.

### 8. The `meta.Open()` Migration Strategy Won't Scale

[meta/db.go L76-95](file:///app/projects/advanced-dada-system/internal/meta/db.go#L76-L95) runs `ALTER TABLE` statements on every `Open()` call, ignoring errors. This is fine for 3-4 migrations but becomes a maintenance hazard at 20+.

**Fix**: Add a `schema_version` table and a proper migration runner that tracks which migrations have been applied.

### 9. FTS5 Virtual Table Is Contentless Without Content Sync

The session schema declares:
```sql
CREATE VIRTUAL TABLE IF NOT EXISTS fts_index USING fts5(text);
```

Note this is **standalone** FTS5 (no `content=io_stream`). The original v0.1 design called for `content=io_stream, content_rowid=id`, which means FTS5 would reference `io_stream` rows directly. The current implementation inserts `rowid` manually to correlate them, which works — but it means **the FTS table stores a full copy of the stripped text**, doubling storage.

**Observation**: This is probably intentional (the `content=` approach has its own quirks with `DELETE` and `UPDATE`), but it should be documented as a conscious trade-off.

### 10. `WriteChunk()` Creates a Transaction Per Chunk

[sessiondb/db.go L87-123](file:///app/projects/advanced-dada-system/internal/sessiondb/db.go#L87-L123) opens a transaction for every 4KB chunk arriving from `tmux pipe-pane`. At high terminal throughput (e.g., `cat large_file`), this is a lot of transaction overhead.

**Fix**: Implement a buffered writer that batches chunks and commits every N chunks or every T milliseconds, whichever comes first. This is a classic write-amplification fix.

### 11. The ANSI Stripper Is Too Simple

[ansi/strip.go](file:///app/projects/advanced-dada-system/internal/ansi/strip.go) uses two regexes:
```go
csiRegex = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)
oscRegex = regexp.MustCompile(`\x1b\].*?(?:\x07|\x1b\\)`)
```

**Missing**: 
- DCS sequences (`\x1bP...\x1b\\`) — used by tmux for nested escape passthrough
- C0 control characters (`\x00`-`\x1f` except `\n`, `\r`, `\t`)
- SS2/SS3 sequences (`\x1bN`, `\x1bO`)

Real terminal output (especially from `vim`, `htop`, `less`) will produce these, leaving noise in the FTS index.

---

## 🔵 Testing & Quality Gaps

### 12. Test Coverage Is Minimal

The entire codebase has **one unit test file** ([engine_test.go](file:///app/projects/advanced-dada-system/internal/search/engine_test.go)) and **two integration scripts**. Zero tests for:
- `ansi.Strip()` — the most critical data transformation in the pipeline
- `OSCScanner` state machine — complex stateful logic with no tests
- `sessiondb.WriteChunk()` — the core write path
- `meta.DB` CRUD operations
- `orchestrator.Run()` — even basic mocking
- Plugin RPC marshaling

This is the #1 priority improvement.

### 13. No Linting for SQL Injection Risk

The search query in [engine.go L44-48](file:///app/projects/advanced-dada-system/internal/search/engine.go#L44-L48) builds SQL with string concatenation for the ANSI highlight markers:
```go
query := `SELECT ... highlight(fts_index, 0, "` + "\033[31m" + `", "` + "\033[0m" + `") ...`
```

This particular case is safe (the injected strings are constants, and the user input uses `?` parameterization), but the pattern is a code review red flag. Consider using a `const` for the query.

### 14. The `.golangci.yml` Is Minimal

Only 7 linters enabled. Missing high-value linters:
- `gosec` — security checks
- `gocritic` — performance and style
- `exhaustive` — exhaustive switch/enum checks
- `unused` — dead code detection
- `bodyclose` — HTTP response body leak detection (critical for the LLM plugin)

---

## 🟣 Architecture & Design Drift

### 15. Version Jump: v0.1 → v0.2 → v0.3 → **v3.3.1**

The design docs plan v0.1 through v0.4. The actual version is `v3.3.1`. There are no design docs for v1.0, v2.0, or v3.0–v3.3. The `v3.0.0_plugin_architecture.md` exists but there's no record of what happened between v0.3 and v3.0.

**Impact**: The `docs/updates/` directory should be checked for version summaries (per AGENTS.md policy), but the design doc gap makes it hard for new contributors (or future AI agents) to understand the evolution.

### 16. Plugin System Uses `net/rpc`, Not gRPC

The architecture doc says "gRPC over Unix Domain Sockets." The actual implementation uses `net/rpc` (Go's legacy RPC, which HashiCorp go-plugin supports as a simpler alternative to gRPC). This is fine for the current `RunTask(map[string]string) → string` interface, but it means:
- No protobuf schemas
- No streaming responses (relevant for LLM streaming)
- Limited to Go plugins (no polyglot extensibility)

If you plan to add LLM streaming or non-Go plugins, you'll need to migrate to the gRPC backend eventually.

### 17. The `RunTask` Interface Is Too Generic

```go
type Service interface {
    RunTask(args map[string]string) (string, error)
}
```

A single `map[string]string → string` interface for everything (search, LLM, future plugins) means:
- No type safety
- JSON serialization/deserialization at every boundary
- No structured error types
- Impossible to add streaming, progress reporting, or cancellation

**Fix**: Define per-plugin interfaces (e.g., `Searcher`, `Analyzer`) and use go-plugin's gRPC backend with proper protobuf messages.

---

## Concrete Improvement Priority List

| Priority | Improvement | Effort | Impact |
|---|---|---|---|
| **P0** | Add unit tests for `ansi.Strip`, `OSCScanner`, `sessiondb.WriteChunk` | 1 day | High — core data path is untested |
| **P0** | Sentinel errors in `meta` package, replace string matching | 2 hours | High — prevents subtle regressions |
| **P0** | Signal handler in `ads-recorder` | 1 hour | High — data loss risk |
| **P1** | Typed struct for OpenRouter API response | 30 min | Medium — crash prevention |
| **P1** | Pass LLM config via `RunTask` args, remove direct DB access | 2 hours | Medium — eliminates coupling |
| **P1** | Concurrent federated search | 3 hours | Medium — performance |
| **P2** | Batch writer for `WriteChunk` | 4 hours | Medium — throughput |
| **P2** | Better ANSI stripper (DCS, C0, SS2/SS3) | 3 hours | Medium — FTS quality |
| **P2** | Proper migration runner for `meta.db` | 3 hours | Medium — maintainability |
| **P3** | Refactor cobra commands into sub-packages | 4 hours | Low-Medium — code organization |
| **P3** | Expand `.golangci.yml` with `gosec`, `bodyclose`, etc. | 1 hour | Low — catches future bugs |
| **P3** | Design docs for v1.0–v3.3 gap | 2 hours | Low — documentation debt |

---

## Summary

The architecture is fundamentally sound — you've avoided the pitfalls that kill most terminal recording tools. The main risks are in the **details**: no tests for the critical data path, string-based error matching, unchecked type assertions, and a plugin interface that's too generic to evolve. The highest-ROI work is adding tests for `ansi` and `sessiondb`, defining sentinel errors, and adding signal handling to the recorder. Everything else is polish on a good foundation.
