# AGENTS.md — Advanced Dada System (ADS)

## Project Status

v0.4.0 implemented! Local sessions, raw recording, FTS5 dual-write, federated search, and native bubbletea interactive TUI are functioning.

## Agent Instructions / Coding Policies

- **VERSIONING & RELEASES**:
  1. Semantic Versioning (`vMAJOR.MINOR.PATCH`) MUST be used.
  2. Bug reports MUST automatically trigger a MINOR version bump (e.g. `v0.4.0` -> `v0.5.0`), exactly one minor bump per bug report.
  3. Successive future versions MUST be laid out by the user under `llm/design/` files. The agent MUST read these files when discussing new versions.
  4. Every bumped version MUST have a summary file tracked under `docs/updates/vX.Y.Z.md` describing improvements, bugfixes, and design choices.
  5. Major version bumps require extensive documentation.
  6. The `Makefile` includes a `make release` command to create a Git tag for the latest version.
- **BUG TRACKING & GIT BRANCHING**:
  1. All bug reports MUST be documented as `.md` files in the `docs/bugs/` directory BEFORE writing any code.
  2. Fixes MUST be developed in a dedicated isolated Git branch (e.g., `git checkout -b fix/<bug-name>`).
  3. Once the fix is built and verified via the `Makefile` pipeline, commit the fix to that specific branch.
  4. Switch back to the main branch (`git checkout main`) and cleanly merge the fix branch (`git merge --no-ff fix/<bug-name>`) to integrate it into the mainline history.
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