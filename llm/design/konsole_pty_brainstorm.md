# Brainstorm: Konsole + PTY Proxy Architecture

## The Core Idea in One Sentence

**`ads run foo` creates a Konsole tab whose shell is `ads-shell --session <uuid>`, which transparently proxies a PTY pair, recording everything to SQLite.**

No tmux. Konsole IS the multiplexer.

---

## The Two New Binaries

### `ads` (orchestrator) — mostly unchanged
- `ads new`, `ads list`, `ads delete`, `ads search`, `ads edit`, `ads llm` — all stay
- `ads run <name>` — instead of spawning tmux, opens a **Konsole tab** via D-Bus (or just launches `ads-shell` directly if already in a terminal)
- `ads hook bash` — unchanged (OSC 133 injection)

### `ads-shell` (replaces `ads-recorder`)
- Single binary, does both the PTY proxy AND the recording
- No separate recorder process — the proxy IS the recorder
- Simpler process tree: `konsole → ads-shell → bash` (was: `konsole → tmux → bash` + `ads-recorder` on the side)

---

## `ads-shell` Internals

```
stdin (from Konsole) ──→ ads-shell ──→ PTY master ──→ bash (on PTY slave)
                              │
                              ├──→ SQLite: io_stream (raw output bytes)
                              ├──→ SQLite: fts_index (ANSI-stripped)
                              ├──→ OSC 133 scanner → command_history
                              │
stdout (to Konsole)  ←── ads-shell ←── PTY master ←── bash output
```

Key behaviors:
- **Bidirectional transparent proxy** — user sees no difference from a normal shell
- **Output recording**: every chunk read from PTY master → `io_stream` + stripped → `fts_index`
- **Input recording** (NEW!): every chunk from stdin → new `input_stream` table? Or just tag direction in `io_stream`?
- **OSC 133 parsing**: inline, same scanner as today
- **SIGWINCH forwarding**: resize the inner PTY when Konsole resizes
- **SIGTERM/SIGHUP**: clean shutdown, flush buffers, close DB
- **Exit propagation**: when bash exits, ads-shell exits with the same code → Konsole closes the tab naturally

### Raw mode handling
- Konsole puts the outer PTY in raw mode for interactive programs
- ads-shell must set its inner PTY to match — `tcgetattr` on outer, `tcsetattr` on inner
- The `creack/pty` Go library handles most of this
- Key: must forward terminal attribute changes, not just data

### Chunk batching
- Instead of one transaction per 4KB chunk (current problem), buffer and commit every 100ms or 64KB, whichever comes first
- Use a goroutine with a ticker for time-based flushing

---

## Konsole Integration

### Profile Generation

`ads new foo` could write a Konsole profile:

```ini
# ~/.local/share/konsole/ADS-foo.profile
[General]
Name=ADS: foo
Command=ads-shell --session <uuid>
Icon=utilities-terminal
Environment=ADS_SESSION=<uuid>

[Appearance]
ColorScheme=Breeze
Font=Hack,12

[Scrolling]
HistoryMode=2
HistorySize=999999
```

Then Konsole's profile dropdown shows "ADS: foo" and clicking it starts a recorded session automatically.

### D-Bus Session Launch

`ads run foo` from a terminal could:

```go
// Open a new Konsole window/tab with the ADS profile
conn, _ := dbus.SessionBus()
obj := conn.Object("org.kde.konsole-<pid>", "/Windows/1")
call := obj.Call("org.kde.konsole.Window.newSession", 0, "ADS-foo")
sessionID := call.Body[0].(int32)
```

Or simpler — just exec:
```bash
konsole --profile "ADS-foo" &
```

### Tab Title via D-Bus

While ads-shell is running, update the Konsole tab title:
```go
// Set tab title to show session name + running duration
obj.Call("org.kde.konsole.Session.setTitle", 0, 1, "ADS: foo [2h15m]")
```

### What if not in Konsole?

If someone runs `ads run foo` from a plain xterm or SSH:
- Just exec `ads-shell` directly in the current terminal — works fine, just no tab management
- Or launch Konsole: `exec konsole --profile "ADS-foo"`

---

## Session Lifecycle (Without tmux)

| Action | Current (tmux) | New (Konsole) |
|---|---|---|
| Create | `ads new foo` → meta.db | Same |
| Start | `ads run foo` → tmux new-session + pipe-pane + attach | `ads run foo` → Konsole tab with `ads-shell` |
| Detach | `Ctrl-b d` → tmux detaches, session lives | ❌ **Not possible** — close tab = end session |
| Reattach | `ads run foo` → tmux attach | ❌ **Not possible** — must start new session |
| Seal | tmux detach triggers status=sealed | ads-shell exit triggers status=sealed |
| Search | Same | Same |
| Delete | Same | Same |

### The Persistence Question

**Accept it: sessions are ephemeral.** When the Konsole tab closes, the session is sealed. The *recording* persists forever in SQLite, but the *live shell* is gone.

This is actually fine for most use cases:
- **Audit/compliance**: You have the full recording. That's what matters.
- **Search/analysis**: Fully intact.
- **LLM summarization**: Works on sealed sessions anyway.
- **Session resume**: The user just opens a new session. Their command history is in bash_history. The terminal state is lost, but that's normal — it's like closing and reopening a terminal window.

If someone truly needs persistence, they can run tmux *inside* ads-shell. The recording still works — tmux output flows through the PTY proxy. Best of both worlds.

---

## CLI Changes

### Unchanged
- `ads new <name>` — same
- `ads list` — same  
- `ads delete <name>` — same
- `ads search <query>` — same (but can ditch the plugin architecture, call search directly?)
- `ads search-interactive` — same TUI
- `ads edit <profile>` — same
- `ads profile *` — same
- `ads hook bash` — same
- `ads llm *` — same
- `ads plugin *` — same
- `ads auth test` — **drop it** (no remote session concept in v1, or keep for SSH-inside-ads-shell)

### Changed
- `ads run <name>` — launches Konsole tab or execs ads-shell directly
- Could add: `ads attach <name>` — opens a new ads-shell for an existing session UUID (continues recording to the same .db, new io_stream rows)

### New
- `ads shell <name>` — just execs ads-shell inline (for use outside Konsole, or piping into Konsole profile)
- `ads konsole install` — generates Konsole profiles for all existing sessions? Or a single "ADS Launcher" profile?
- `ads konsole sync` — re-syncs Konsole profiles with meta.db

---

## What to Keep From Current Codebase

| Package | Keep? | Notes |
|---|---|---|
| `internal/config` | ✅ | XDG path resolution — unchanged |
| `internal/meta` | ✅ | meta.db CRUD — add `backend` column? |
| `internal/sessiondb` | ✅ | Per-session SQLite — unchanged, maybe add input_stream |
| `internal/ansi` | ✅ | Strip + OSC 133 scanner — unchanged |
| `internal/search` | ✅ | Federated FTS5 search — unchanged |
| `internal/orchestrator` | 🔄 | **Rewrite**: replace tmux logic with Konsole D-Bus + ads-shell exec |
| `internal/plugin` | ❓ | Do we still need HashiCorp go-plugin? See below |
| `cmd/ads` | 🔄 | Modify `run` command, add Konsole integration commands |
| `cmd/ads-recorder` | ❌ | **Replace with `cmd/ads-shell`** — PTY proxy + recorder in one |
| `cmd/ads-plugin-search` | ❓ | If ditching plugin architecture, move search back inline |
| `cmd/ads-plugin-llm` | ❓ | If ditching plugin architecture, move LLM call inline |

### The Plugin Question

The HashiCorp go-plugin system adds complexity. With the tmux architecture, isolating search/LLM into separate processes made sense (crash isolation from the recorder). With ads-shell:
- The recorder IS the shell proxy — it can't crash without the user noticing anyway
- Search and LLM are query-time operations, not recording-time
- Could just call them directly from the `ads` binary — no plugin subprocess needed
- **Simpler = better here**

Counter-argument: plugins allow third-party extensions. But nobody has written a third-party plugin yet, and YAGNI.

**Suggestion**: Drop go-plugin for now. `ads search` calls `internal/search` directly. `ads llm` calls an `internal/llm` package directly. If plugin extensibility is needed later, add it back with a clearer interface.

---

## New Capabilities Unlocked

### 1. Input Recording
The PTY proxy sees BOTH directions. You can record what the user typed:

```sql
CREATE TABLE io_events (
    id INTEGER PRIMARY KEY,
    ts DATETIME DEFAULT CURRENT_TIMESTAMP,
    direction TEXT NOT NULL,  -- 'in' or 'out'
    data BLOB NOT NULL
);
```

This enables:
- **Keystroke-level replay** at original speed
- **Audit**: "what did the operator type?" — critical for compliance
- **Command extraction** even without OSC 133 (heuristic: input between \r characters)

### 2. Session Multiplexing
ads-shell can be launched multiple times for the same session UUID → multiple recording segments in the same .db. Like "chapters" of a session.

### 3. Environment Capture
At startup, ads-shell can snapshot the environment (`env`, `uname -a`, `hostname`, terminal size, shell version) into a `session_info` table:

```sql
CREATE TABLE session_info (
    key TEXT PRIMARY KEY,
    value TEXT
);
-- hostname, kernel, shell, term_size, user, cwd, etc.
```

### 4. Konsole-Native Scrollback Search
Since Konsole has its own scrollback buffer AND ads has FTS5 on the same data, you could bind `Ctrl+Shift+F` (Konsole's search) for current-tab search and `Ctrl+S` (custom) for cross-session federated ADS search.

---

## Open Questions

1. **Do we need the `ads-shell` binary, or could `ads` itself be the shell proxy?**
   - `ads shell --session <uuid>` as a subcommand instead of a separate binary
   - Pro: single binary to install
   - Con: `ads` is heavy (cobra, all commands); the shell proxy should be minimal

2. **Remote sessions?**
   - Option A: `ads-shell` just runs `ssh user@host` instead of `bash` as the child process
   - Option B: Drop remote support, let users just type `ssh` inside a recorded session
   - Option B is simpler and actually gives better recording (you see the ssh command itself)

3. **Konsole profile per-session vs. generic "ADS" profile?**
   - Per-session: clutters Konsole's profile list
   - Generic: one "ADS" profile that prompts for which session to run (or a session picker TUI)
   - Hybrid: one launcher profile + `ads run` manages the rest

4. **Should `ads run` block or background?**
   - Block: `ads run foo` execs ads-shell, user is now in the session (current terminal becomes the session)
   - Background: `ads run foo` opens a new Konsole tab and returns immediately
   - Both are useful — flag? `ads run foo` blocks, `ads open foo` opens Konsole tab?

5. **Do we keep the interactive search TUI inside tmux popup?**
   - No tmux = no `display-popup`
   - Alternative: run it as a Konsole split pane, or in a floating Konsole window
   - Or just run it fullscreen with alt-screen (Bubbletea already does this)

6. **What about the managed tmux config (scrollback, mouse, keybindings)?**
   - Scrollback: Konsole handles natively (infinite scrollback option)
   - Mouse: Konsole handles natively
   - Keybindings: Konsole profiles handle this
   - The whole `tmux_profiles` table in meta.db becomes... Konsole profiles on disk?
   - Or keep it as an abstract "profile" concept that generates Konsole profiles?
