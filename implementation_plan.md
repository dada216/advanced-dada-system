# Implementation Plan: Advanced Dada System (ADS)

This implementation plan is based on the revised, CLI-first, multiplexer-driven architecture (`architecture_revision002.md`). It breaks the development into distinct, measurable phases designed for a solo developer, starting with high-risk technical spikes and building up to the complete platform.

## Phase 0: Technical Spikes (The Unknowns)
**Goal:** Validate the core mechanics of `tmux pipe-pane` and raw byte recording.
**Duration:** ~1 week

1. **Spike A: Manual tmux pipeline**
   * Manually launch a `tmux` session.
   * Run `tmux pipe-pane -o 'cat >> session.log'` and interact with the terminal (run `vim`, resize the window).
   * Verify that the terminal behaves normally and the `.log` file accurately captures raw bytes including ANSI codes.
2. **Spike B: Barebones Recorder (`ads-recorder-spike`)**
   * Write a tiny Go program that reads from `os.Stdin` and prints chunk sizes to `stdout`.
   * Hook it to `tmux pipe-pane` to verify data flows correctly and the process terminates when the pane closes.
3. **Spike C: ANSI Stripping**
   * Identify or build a fast Go library/regex to strip ANSI codes (`\x1b[...]`) from a byte slice to prove clean text can be extracted for FTS5.

---

## Phase 1: Meta-Database and Project Skeleton
**Goal:** Establish the `ads` CLI orchestrator and the central state.
**Duration:** 1-2 weeks

1. **CLI Scaffolding**
   * Initialize a Go project using `spf13/cobra` for the CLI.
   * Create base commands: `ads new`, `ads list`, `ads run`, `ads search`.
2. **Meta-Database (SQLite)**
   * Use `mattn/go-sqlite3` or `modernc.org/sqlite` to initialize `meta.db` in `~/.local/share/ads/`.
   * Create schemas for:
     * `sessions` (uuid, name, type [local/remote], status, created_at)
     * `config` (session_uuid, key, value)
3. **Phase 1 Config Shell (`ads new`)**
   * Implement `ads new <name>`.
   * Provision a record in `meta.db`.
   * Spawn a subshell where the user can run `ads set key value`.
   * Persist the config to `meta.db` when the shell exits.

---

## Phase 2: Local Sessions & Raw Recording
**Goal:** Launch a local session, pipe output, and save to a per-session SQLite database.
**Duration:** 2-3 weeks

1. **Session Database Schema**
   * Define the schema for the per-session DBs (e.g., `~/.local/share/ads/sessions/<uuid>.db`).
   * Tables: `metadata`, `io_stream` (timestamp, chunk), `fts_index` (virtual table).
2. **The Recorder Process (`ads-recorder`)**
   * Implement the headless Go binary that receives the `--session <uuid>` flag.
   * Reads from `os.Stdin` in chunks.
   * Writes raw chunks to the `io_stream` table.
   * Passes chunks through the ANSI-stripper and writes the sanitized text to `fts_index`.
3. **The Orchestrator (`ads run`)**
   * Implement local session launching.
   * `ads` executes `tmux new-session -d -s <uuid> bash`.
   * `ads` executes `tmux pipe-pane -t <uuid> -o 'ads-recorder --session <uuid>'`.
   * `ads` attaches to the `tmux` session.

---

## Phase 3: Remote SSH Orchestration
**Goal:** Extend the system to support remote execution securely.
**Duration:** 1-2 weeks

1. **Configuring Remote Sessions (`ads new --remote`)**
   * Add interactive testing: `ads auth test`.
   * The CLI attempts to run `ssh -o BatchMode=yes <user>@<host> exit` to confirm the native SSH agent is working and the host is reachable.
2. **Executing Remote Sessions (`ads run`)**
   * Modify the orchestrator logic for remote sessions.
   * `ads` executes `tmux new-session -d -s <uuid> 'ssh <user>@<host>'`.
   * The same `tmux pipe-pane` logic attaches the recorder.
   * Verify session resilience (disconnecting the network, re-running `ads run` to reattach).

---

## Phase 4: Semantic Output Structuring (OSC 133)
**Goal:** Introduce best-effort semantic chunking for LLM preparation.
**Duration:** 2 weeks

1. **Shell Integration Hooks**
   * Embed `bash-preexec` or equivalent scripts into the Go binary.
   * On `ads run`, ensure the target shell sources these hooks to emit OSC 133 markers.
2. **OSC 133 Scanner in `ads-recorder`**
   * Build a byte scanner that looks for `\x1b]133;` markers in the incoming stream.
   * Update the session DB schema to include a `command_history` table (timestamp, command_zone, text).
   * When markers are detected, segment the incoming chunks into Prompts, Inputs, and Outputs.
3. **Graceful Degradation Testing**
   * Ensure the recorder doesn't crash or fail to record if markers are missing (e.g., when the user runs `vim`).

---

## Phase 5: Federated Query Layer (`ads search`)
**Goal:** Make the captured data useful through CLI search.
**Duration:** 1 week

1. **Federated Query Logic**
   * Implement `ads search "error string"`.
   * Query the Meta-DB to find all session UUIDs (optionally filtering by `--tag`).
   * Spin up worker goroutines to query the `fts_index` of the matching session databases.
2. **Output Formatting**
   * Return results to `stdout` in a clean, parseable format (JSONlines or grep-like syntax) so users can pipe them to other tools.

---

## Phase 6: Extensibility (Plugins & LLMs)
**Goal:** Connect the platform to external analytical engines.
**Duration:** 3+ weeks

1. **Plugin Architecture**
   * Integrate `hashicorp/go-plugin`.
   * Define the gRPC interface for plugins (e.g., `GetSessionOutput(uuid) string`).
2. **LLM Integration Plugin**
   * Build an official plugin `ads-plugin-llm`.
   * This plugin scans completed sessions, pulls the OSC 133-tagged command outputs, and queries a local LLM (like Ollama) or cloud API to summarize errors.
3. **Ansible Runner Integration**
   * Create an `ads ansible` subcommand that wraps Ansible Runner.
   * Capture the JSON event streams and dump them directly into the target session's SQLite database.

---

## Technical Stack Overview
* **Primary Language:** Go (Golang)
* **Databases:** SQLite (using CGo `mattn/go-sqlite3` for robust FTS5 support)
* **CLI Framework:** `spf13/cobra`
* **Multiplexer:** Native `tmux`
* **Remote Protocol:** Native `ssh` binary
* **Plugin System:** `hashicorp/go-plugin` (gRPC)
