**Executive Critique**
The design has clearly improved across iterations. `architecture_revision000.md` builds the system around `tmux -CC`; `architecture_revision001.md` moves to a simpler raw PTY/SSH recording proxy. That move is directionally correct.

The best current architecture is:

```text
Client terminal / xterm.js
        |
        v
Daemon: session manager + byte broadcaster + recorder
        |
        v
Local PTY or remote SSH PTY
```

The daemon should own sessions from birth, record byte streams, scan for OSC 133 markers, and let clients render terminal output. It should not intercept typed `ssh` commands, should not use tmux Control Mode as the foundation, and should not maintain server-side VT state for the core product.

The biggest issue now is that `architecture_revision001.md` still overstates several guarantees and under-specifies the hard operational details. Raw byte capture is a good simplification, but it does not automatically solve replay, reattach, resize, SSH feature parity, redaction, multi-client fan-out, or durability.

**What The Design Gets Right**
The move away from `tmux -CC` is the strongest improvement. `architecture_revision000.md` makes tmux Control Mode foundational at `design-iterations/architecture_revision000.md:15`, but the critiques correctly identify that this turns the daemon into a terminal-emulator-adjacent system with protocol parsing, octal decoding, flow control, and input translation. `architecture_revision001.md` instead uses raw PTY bridging at `design-iterations/architecture_revision001.md:15`, which is much more feasible.

The design correctly rejects SSH interception. The critiques are right that detecting typed `ssh` commands inside a PTY stream is a fragile state-machine problem. Explicitly launched remote sessions through profiles are the right boundary.

The database-per-session SQLite model is sound. The split between a meta DB and per-session DBs at `design-iterations/architecture_revision001.md:42` is a good fit for archival, customer isolation, backup, and eventual search federation.

OSC 133 is the right semantic segmentation mechanism. The design’s use of prompt/input/output/end markers at `design-iterations/architecture_revision001.md:27` is appropriate, especially for later LLM retrieval.

The plugin architecture is directionally sane, but should be deferred. The HashiCorp-style multi-process model at `design-iterations/architecture_revision001.md:82` is a good eventual shape, but it should not be in the core MVP.

Ansible Runner artifact capture is the right integration strategy. Capturing structured Runner events instead of scraping stdout at `design-iterations/architecture_revision001.md:76` is the correct design.

**Major Design Problems**
1. The default containerized daemon model conflicts with local terminal capture.

`architecture_revision001.md` says the software runs inside Podman and bind-mounts the host Podman socket at `design-iterations/architecture_revision001.md:9`. That is risky and probably the wrong default for the terminal recorder.

If the daemon runs inside a container, “local shell” means a shell inside the container unless you deliberately pierce namespaces, mount host paths, proxy user identity, and forward TTY semantics. That is not the same thing as recording the user’s real desktop shell. Also, bind-mounting the Podman socket grants broad control over the user’s container engine. The claim that this does not compromise the host security posture is too strong.

Recommendation: make the daemon host-native by default. Podman should be an execution backend, not the runtime boundary for the core daemon.

2. “Mathematically perfect replay” is overclaimed.

`architecture_revision001.md:19` says raw byte capture guarantees a perfect replay exactly as the user experienced it. That is only true under constraints.

Replay fidelity depends on terminal size, resize events, terminal type, timing, input/output ordering, alternate screen behavior, and the renderer used during replay. A raw byte stream is the right primitive, but it is not enough by itself.

The design must record:

- stdout/stderr or backend output bytes
- optional input bytes
- monotonic timestamps
- wall-clock timestamps
- terminal size at session start
- resize events
- `TERM`, shell, profile, host, command/backend metadata
- exit status and session termination reason
- sequence numbers to preserve ordering

Without resize events, replay of `vim`, `htop`, progress bars, full-screen TUIs, and wrapped output will diverge.

3. Multi-client attach is under-designed.

The latest critique correctly notes that multiple clients cannot independently read from `s.output()` at `llm/critiques/architectural_critique3.md:286`. This is not a small afterthought; it is central.

The daemon needs a single reader per backend and a broadcaster that fans out to:

- recorder
- active CLI attach client
- TUI client
- WebSocket clients
- optional live processors such as OSC scanner or redactor

The recording path must never be blocked by a slow WebSocket client. Client queues should be bounded. Slow clients should drop frames, disconnect, or switch to catch-up mode. The recorder should have its own bounded queue and explicit failure policy. Silent drops, like the pseudocode implies at `llm/critiques/architectural_critique3.md:166`, are unacceptable for an audit/logging product.

4. Reattach semantics are not defined.

If the daemon does not maintain VT state, a newly attached client only sees future bytes unless the daemon replays history from the beginning or from a recent checkpoint. That may be acceptable, but the product must say so.

There are three workable options:

- MVP: attach shows future output only, with command history/search available separately.
- Better: attach replays the last N seconds or last N bytes into the client before live streaming.
- Advanced: store periodic terminal snapshots for fast seek/reattach.

The design says clients bring their own terminal emulators, which is good, but then it must accept that the server cannot magically provide current screen state unless it stores snapshots or replays prior bytes.

5. SSH feature parity is a large hidden project.

`architecture_revision001.md:21` says remote sessions use standard SSH libraries and request a remote PTY. That is right, but a usable SSH replacement needs more than `ssh.Dial`.

The design must explicitly cover:

- `~/.ssh/config`
- `known_hosts` verification
- SSH agent auth
- encrypted private keys
- keyboard-interactive auth
- ProxyJump / bastion hosts
- host key algorithms and legacy hosts
- keepalives
- terminal modes
- PTY resize propagation
- reconnect behavior
- SSH exit status and signal handling

The earlier critique already calls out `known_hosts`, SSH config parsing, and agent forwarding at `llm/critiques/architectural_critique.md:308`. These should be promoted into the actual design.

6. The remote exit-code claim is inaccurate.

`architecture_revision001.md:25` says the design accurately captures remote exit codes. A raw interactive SSH shell does not give per-command exit codes by itself. You only get the shell/session exit status when the shell terminates.

Per-command exit codes require shell integration, such as OSC 133 `D` markers. That means the design has not removed tracking scripts entirely; it has moved them into shell hooks. That is fine, but the document should be precise.

7. OSC 133 shell hooks are central but brittle.

OSC 133 is the right idea, but the design treats hook injection as easier than it is.

Things that will break or degrade semantic capture:

- unsupported shells
- custom prompt frameworks
- nested shells
- `sudo -s`
- `su`
- typed `ssh` inside a local shell
- raw-mode programs
- commands that disable echo
- shell history settings
- malformed or partial escape sequences
- OSC markers split across read chunks
- terminal apps that emit OSC sequences themselves

The correct guarantee is: raw byte recording is always available; semantic command segmentation is best-effort and shell-dependent.

8. Command history cannot be reliably extracted from output echo alone.

The schema includes `command_history` at `design-iterations/architecture_revision001.md:49`, but the design does not explain how command text is captured robustly.

If you only record PTY output, command text appears because the terminal echoes typed characters in canonical mode. That fails for password prompts, raw-mode apps, multiline edits, readline corrections, bracketed paste, hidden input, and shell editing sequences.

Recommendation: have shell hooks emit a structured command-start event containing the command string where possible. Store that separately from raw PTY bytes. Do not pretend raw echo reconstruction is reliable command history.

9. Secret handling is unsafe as written.

`architecture_revision001.md:70` says credentials are pulled from the OS keyring and injected as environment variables. That is convenient but dangerous.

Environment variables can leak through:

- child processes
- `/proc/<pid>/environ`
- crash dumps
- shell history
- `env`
- logs
- recorded terminal output
- plugin access
- Ansible artifacts

Also, “protected, non-swappable memory” is not realistic once values are injected into a process environment.

Recommendation: avoid env injection for high-value secrets where possible. Prefer SSH agent, askpass, short-lived temp files with strict permissions, scoped credentials, or backend-specific credential helpers. Add recorder-side redaction for known secret values before writing to SQLite.

10. Search over raw terminal bytes will be poor unless normalized.

The design says FTS5 indexes output at `design-iterations/architecture_revision001.md:49`. Raw terminal output includes ANSI color, cursor movement, backspaces, progress bars, alternate screen output, and binary-ish data. Indexing that directly will produce noisy search.

Recommendation: store raw chunks as BLOBs, but index a sanitized text projection. Strip ANSI/OSC sequences, normalize backspaces, optionally ignore alternate-screen full-screen app output, and index command/output semantic chunks rather than every raw byte chunk.

11. SQLite federation needs constraints.

The federated query layer at `design-iterations/architecture_revision001.md:58` says it can dynamically attach session DBs. That is workable for small targeted sets, but SQLite has attachment limits and attaching thousands of databases is not a scalable search strategy.

Recommendation: use the meta DB to narrow candidates first. For broad search, query session DBs in bounded worker pools or maintain a separate aggregate search index. Do not rely on attaching a large number of DBs at once.

12. Backup and archival need SQLite-specific handling.

The storage abstraction at `design-iterations/architecture_revision001.md:64` treats session databases like movable files. With WAL mode, a live SQLite database may include `db`, `db-wal`, and `db-shm`. Copying or moving only the main file can corrupt or lose recent writes.

Recommendation: archival must use the SQLite backup API, `VACUUM INTO`, or a checkpoint-and-copy protocol. Define when a session DB is sealed, when WAL is checkpointed, and what exact artifact gets uploaded.

13. The Web interface needs an explicit security model.

The embedded WebSocket/REST interface at `design-iterations/architecture_revision001.md:93` exposes live shells. That is extremely sensitive.

The design should specify:

- bind address defaults to localhost
- authentication token
- origin checks
- CSRF protections for REST endpoints
- WebSocket authorization
- TLS expectations if exposed beyond localhost
- audit logging for attach/control actions

A browser terminal connected to a local daemon is effectively remote code execution as the user.

14. The plugin model needs capabilities, not just process isolation.

The plugin architecture at `design-iterations/architecture_revision001.md:82` isolates crashes, but not authority. A plugin that can query all sessions, read secrets, or attach to terminals is extremely powerful.

Recommendation: define plugin capabilities early, even if plugins ship later. Example capabilities: read session metadata, read redacted chunks, perform search, request LLM context, write annotations. Do not give plugins raw terminal control by default.

15. The product scope is too broad for early delivery.

The design includes terminal recording, remote SSH, Podman orchestration, Ansible Runner, plugins, LLM RAG, time tracking, TUI, CLI, Web, storage tiering, and Hetzner-specific backends. That is a platform, not an MVP.

The architecture should explicitly separate core from later services.

**Recommended Target Architecture**
The core should be host-native and Go-based unless there is a strong reason to choose Rust. The critique files already point toward Go libraries, and Go has strong fits for PTY, SSH, SQLite, Bubble Tea, gRPC, and plugin work.

Recommended v1 core:

```text
cmd/ads
internal/daemon
internal/session
internal/session/local_pty.go
internal/session/remote_ssh.go
internal/session/broadcaster.go
internal/recording/recorder.go
internal/recording/osc133.go
internal/database/meta.go
internal/database/session_db.go
internal/search/fts.go
internal/profile/profile.go
internal/shell/bash.sh
internal/shell/zsh.sh
```

The critical runtime path should be:

```text
backend reader
  -> redactor
  -> OSC scanner
  -> recorder queue
  -> client broadcaster
```

The backend interface should look conceptually like:

```go
type Backend interface {
    Read([]byte) (int, error)
    Write([]byte) (int, error)
    Resize(cols, rows int) error
    Wait() error
    Close() error
}
```

Local PTY and remote SSH should implement the same interface.

**Recommended MVP**
Phase 0 should prove the hard parts with throwaway spikes:

- local PTY with raw attach, resize, and replay
- remote SSH PTY with `known_hosts`, SSH agent, and resize
- OSC 133 scanning across chunk boundaries
- SQLite batched recording under high output volume
- broadcaster with slow-client behavior

Phase 1 should ship a useful local recorder:

- host-native daemon
- UDS control API
- local PTY sessions
- CLI `start`, `attach`, `list`, `kill`, `replay`
- per-session SQLite
- batched writes
- resize event capture
- raw output replay
- basic FTS over sanitized output

Phase 2 should add remote sessions:

- `crypto/ssh`
- `~/.ssh/config`
- `known_hosts`
- agent auth
- ProxyJump if needed
- remote PTY resize
- profile metadata
- best-effort shell hooks

Phase 3 should add richer UX:

- TUI dashboard
- search UI
- session tags
- optional web/xterm.js

Phase 4 and later:

- storage archival
- tmux-as-normal-subprocess persistence
- Ansible Runner
- plugins
- LLM RAG
- time tracking
- Podman orchestration

**Bottom Line**
The raw recording-proxy model from `architecture_revision001.md` and `architectural_critique3.md` is the right foundation. Keep that.

The design should be revised to remove overclaims, demote containerization from “default runtime” to “managed backend,” make the broadcaster/recorder pipeline first-class, define replay/reattach semantics, specify SSH parity requirements, and add an explicit security/redaction model.

If those changes are made, the architecture becomes much more buildable: a small host-native daemon that owns PTY/SSH sessions, records raw bytes safely, extracts best-effort semantic markers, and grows into analytics, storage, web, plugins, and automation only after the core terminal recorder is reliable.