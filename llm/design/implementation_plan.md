# Implementation Plan: Advanced Dada System (ADS) v0.1

**Scope:** First end-to-end usable release. Local sessions only, raw recording, basic search. No SSH, no OSC 133, no plugins. 

*Note: All development must adhere to the Coding Agent Policies defined in `architecture.md`, heavily leveraging Makefiles and `golangci-lint` v2.*

---

## What v0.1 Delivers

| Command | Behavior |
|---|---|
| `ads new <name>` | Create session in `meta.db`, print instructions |
| `ads list` | List sessions with status |
| `ads run <name>` | Spawn tmux, start `ads-recorder` via `pipe-pane`, attach |
| `ads search <query>` | FTS5 search across all session databases |
| `ads-recorder --session <uuid>` | Read stdin, dual-write to per-session SQLite |

**Explicitly deferred to post-v0.1:** `--remote`/SSH orchestration, OSC 133 shell hooks, `ads auth test`, plugin system, Ansible integration.

---

## Directory Layout

```text
cmd/
  ads/main.go            # orchestrator entrypoint
  ads-recorder/main.go   # recorder entrypoint
internal/
  meta/                  # meta.db schema + CRUD
  sessiondb/             # per-session .db schema + dual-write
  ansi/                  # ANSI escape stripping for FTS
  orchestrator/          # tmux lifecycle, pipe-pane wiring
  config/                # XDG paths, constants
Makefile                 # Centralized build, lint, and test definitions
.golangci.yml            # Linter configuration
go.mod
go.sum
```

Two binary targets in `go.mod`: `./cmd/ads` and `./cmd/ads-recorder`.

---

## Implementation Steps (ordered)

### Step 1: Project skeleton & Tooling

- `go mod init github.com/advanced-dada-system/ads`
- Add `spf13/cobra` and `mattn/go-sqlite3` dependencies
- Create `cmd/ads/main.go` and `cmd/ads-recorder/main.go` with cobra root commands
- Create `internal/config/` — resolve `~/.local/share/ads/` via XDG, ensure directories on startup
- **Tooling setup:** Define a `Makefile` wrapper for `go build`, `go test`, and implement a `lint` target using `golangci-lint` v2. Add `.golangci.yml`.
- Verify: Run `make build` and `make lint`, ensure both binaries compile and print help.
- (Optional) Commit: run the `mastering-git-cli` skill to commit the work according to the defined gitflow rules.

### Step 2: Meta-database + `ads new` / `ads list`

- `internal/meta/` — open/create `meta.db`, run migrations, CRUD for sessions table
- Schema:
  ```sql
  CREATE TABLE sessions (
      uuid   TEXT PRIMARY KEY,
      name   TEXT UNIQUE NOT NULL,
      type   TEXT NOT NULL DEFAULT 'local',  -- 'local' only in v0.1
      status TEXT NOT NULL DEFAULT 'created', -- created | running | sealed
      created_at DATETIME DEFAULT CURRENT_TIMESTAMP
  );
  ```
- `ads new <name>` — insert row, print session UUID and next-step hint
- `ads list` — select and display all sessions
- Verify: `make lint build`, then `ads new test-session && ads list` shows it.

### Step 3: Session database schema + `ads-recorder`

- `internal/sessiondb/` — open/create per-session `<uuid>.db`, run migrations
- Schema:
  ```sql
  CREATE TABLE io_stream (
      id   INTEGER PRIMARY KEY,
      ts   DATETIME DEFAULT CURRENT_TIMESTAMP,
      data BLOB NOT NULL
  );
  CREATE VIRTUAL TABLE fts_index USING fts5(text, content=io_stream, content_rowid=id);
  ```
- `internal/ansi/` — strip ANSI/OSC/control sequences, produce clean UTF-8 text for FTS
- `ads-recorder --session <uuid>` — open session DB, read `os.Stdin` in sized chunks, insert into `io_stream` and `fts_index` (stripped), flush on `io.EOF`
- SQLite opened in WAL mode
- Verify: `echo "hello \033[31mworld\033[0m" | ads-recorder --session <uuid>`, then query session DB shows raw blob in `io_stream` and `fts_index` contains clean "hello world"

### Step 4: `ads run` — tmux orchestration

- `internal/orchestrator/` — functions to:
  1. Look up session in `meta.db`
  2. Update status to `running`
  3. `exec.Command("tmux", "new-session", "-d", "-s", uuid, "bash")`
  4. `exec.Command("tmux", "pipe-pane", "-t", uuid, "-o", "ads-recorder --session " + uuid).Run()`
  5. `exec.Command("tmux", "attach", "-t", uuid).Run()` (blocks until detach)
  6. On attach return: update status to `sealed`
- Handle `ads run` for already-running sessions (reattach)
- Verify: `ads new demo && ads run demo` — opens tmux, all terminal output is recorded, `Ctrl-b d` detaches and seals session, session DB contains recorded data

### Step 5: `ads search`

- Query `meta.db` for all `sealed` session UUIDs (or `--name` filter)
- Open each session DB concurrently (bounded goroutine pool), run FTS5 `MATCH` against `fts_index`
- Print results as JSON lines: `{session, name, rowid, snippet}`
- Verify: Run a session producing known output, `ads search <word>` returns matching lines

### Step 6: Hardening + integration tests

- Graceful `ads-recorder` shutdown on `io.EOF` or `SIGTERM` — close DB cleanly
- `ads run` handles tmux session name collisions (error if session exists and is running)
- `ads delete <name>` — `unlink()` session DB + delete from `meta.db`
- Integration test script via `Makefile` (`make test-integration`): create session → run → produce output → detach → search → verify results
- Clean up cobra command help text, flags, error messages
- Final run of `make lint` to enforce `golangci-lint` v2 standards.

---

## Key Technical Risks

| Risk | Mitigation |
|---|---|
| `tmux pipe-pane` command quoting/path issues | Spike: verify `pipe-pane -o` invokes recorder binary correctly; use absolute path to `ads-recorder` |
| CGo build complexity (`mattn/go-sqlite3`) | Ensure `CGO_ENABLED=1` and C compiler in CI; consider `modernc.org/sqlite` as fallback |
| Recorder crash loses data | WAL mode + `PRAGMA synchronous=NORMAL`; recorder flushes every chunk, not just at EOF |
| FTS5 content= table sync with raw BLOB table | Use triggers on `io_stream` INSERT to populate `fts_index`; or write both in same transaction |

---

## What Comes After v0.1

- **v0.2:** SSH remote sessions (`ads new --remote`, `ads auth test`)
- **v0.3:** OSC 133 shell hooks + semantic `command_history` table
- **v0.4:** Plugin system (`hashicorp/go-plugin`), LLM integration
