# AGENTS.md — Advanced Dada System (ADS)

## Project Status

v0.1 implemented! Local sessions, raw recording, FTS5 dual-write, and federated search are functioning.

## Agent Instructions / Coding Policies

- **CODING STYLE**: You must strictly adhere to the guidelines codified in `llm/design/coding_style.md` (derived from Effective Go).
- **MANDATORY**: ALWAYS run the `Makefile` verification pipeline (`make tidy`, `make lint`, `make build`, and `make test`/`make test-integration`) before committing code or declaring a milestone complete.
- Do not bypass `golangci-lint` errors. If formatting (`gofmt`) or code logic fails, correct it proactively.
- All code commits must be atomic and strictly adhere to the architecture laid out in `llm/design/architecture.md`.

## Architecture

Two CLI binaries (may ship as a single executable with subcommands):

- **`ads`** — Orchestrator: manages config, launches sessions, queries data
- **`ads-recorder`** — Headless, ephemeral stdin reader; one process per active session; writes raw + stripped output to a per-session SQLite DB

No centralized daemon. Failure in one recorder is isolated.

## Key Design Decisions (non-obvious)

- **`tmux pipe-pane`** is the sole capture mechanism — no custom PTY wrapper or VT state machine
- **Native `ssh` binary** for remote sessions — no in-process SSH client (inherits `~/.ssh/config`, agent forwarding, ProxyJump, etc.)
- **Database-per-session** SQLite files in `~/.local/share/ads/sessions/<uuid>.db` + a central `meta.db` — avoids monolithic DB contention; deleting a session is `unlink()`
- **OSC 133 markers** injected via shell hooks for semantic chunking (prompt/input/output); best-effort, degrades gracefully to raw temporal indexing
- **Dual-write in recorder**: raw bytes → `io_stream` table; ANSI-stripped text → `fts_index` virtual table (FTS5)

## Tech Stack

- Go with `spf13/cobra` for CLI
- SQLite via CGo (`mattn/go-sqlite3`) — requires C toolchain for builds
- `hashicorp/go-plugin` (gRPC over Unix sockets) for future plugin system
- Native `tmux` and `ssh` binaries (must be installed on host)

## Session Lifecycle

1. **`ads new <name>`** / **`ads new --remote <name>`** — opens ephemeral config shell; persists to `meta.db`
2. **`ads run <name>`** — spawns `tmux` session, starts `ads-recorder` via `pipe-pane`, attaches user

## Data Paths

- `~/.local/share/ads/meta.db` — session index, configs, tags
- `~/.local/share/ads/sessions/<uuid>.db` — per-session recording data