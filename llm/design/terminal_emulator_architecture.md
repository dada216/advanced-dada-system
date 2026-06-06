# Alternative Architecture: Terminal Emulator Integration

## The Core Question

The current ADS design says: *"I'll wrap your session in tmux, pipe the output to a recorder, and you'll interact with tmux."* This works, but it adds a layer: the user runs their terminal emulator → which runs `ads run` → which runs `tmux` → which runs the shell. The user's terminal emulator becomes a dumb viewport into tmux.

The question is: **can we eliminate the tmux middleman and integrate recording directly into the terminal emulator itself?**

The answer is yes, but through several fundamentally different paths, each with different depths of integration.

---

## Approach 1: Konsole D-Bus + PTY Sidecar

### How Konsole Works

Konsole is a KPart-based KDE application. It exposes a **D-Bus interface** under `org.kde.konsole`:

```
org.kde.konsole.Session
  ├── sendText(QString text)
  ├── setProfile(QString name) 
  ├── setTitle(int role, QString title)
  ├── environment() → QStringList
  ├── processId() → int
  └── foregroundProcessId() → int

org.kde.konsole.Window
  ├── newSession(QString profile) → int
  ├── currentSession() → int
  └── setCurrentSession(int id)
```

### The Problem

D-Bus gives you **control** (create sessions, send text, switch profiles) but **not observation** of the output stream. There's no `onOutput(bytes)` signal. You can't subscribe to the raw byte stream flowing from the PTY to the display.

### The Sidecar Approach

Since Konsole gives you the child process PID via D-Bus, you *could* build a sidecar that:

1. Uses D-Bus to create a new Konsole session (tab/window)
2. Gets the PTY slave path from `/proc/<pid>/fd/0`
3. Opens the PTY master side for reading (this is where it gets ugly — you can't just read another process's PTY master)

**Verdict: This doesn't work cleanly.** D-Bus integration with Konsole is useful for session management (creating tabs, setting profiles) but not for I/O capture. You'd need a separate capture mechanism.

### Hybrid: Konsole D-Bus for UX + `script(1)`-style PTY wrapper for capture

```
┌──────────────────────────────────────┐
│ Konsole                              │
│  ┌──────────────────────────────────┐│
│  │ Tab: "production-server"         ││
│  │                                  ││
│  │  ads-shell ──→ PTY pair ──→ bash ││
│  │      │                           ││
│  │      └──→ SQLite writer          ││
│  └──────────────────────────────────┘│
│  ┌──────────────────────────────────┐│
│  │ Tab: "dev-local"                 ││
│  └──────────────────────────────────┘│
└──────────────────────────────────────┘
```

- **`ads new myserver`** → registers session in `meta.db`, creates a Konsole profile via D-Bus that sets the shell to `ads-shell --session <uuid>`
- **`ads-shell`** → a PTY proxy: allocates a PTY pair, forks the real shell on the slave side, reads all master I/O, writes to SQLite, forwards to the real terminal's stdout
- **Konsole profiles** map 1:1 to ADS sessions — so the user just opens a Konsole tab and it's automatically recorded
- **`ads search`** / **`ads llm`** → unchanged CLI tools querying the same SQLite databases

### Trade-offs

| Aspect | Current (tmux) | Konsole D-Bus + PTY Proxy |
|---|---|---|
| Terminal rendering | tmux does it (good but not native) | Konsole does it (native, GPU-accelerated) |
| Session persistence | tmux detach/reattach | ❌ Lost if Konsole closes (no detach) |
| Split panes | tmux native | Konsole native (but no per-pane recording) |
| Emulator lock-in | None (any terminal can run tmux) | Konsole-specific for D-Bus parts |
| Window management | tmux windows | KDE window/tab management |
| OSC 133 | Parsed by ads-recorder | Konsole already parses these natively! |
| Remote sessions | tmux + native ssh | Konsole's native SSH support + proxy |

> [!IMPORTANT]
> **The killer issue: no session persistence.** tmux gives you detach/reattach for free. If you close a Konsole window, the session is gone. This is fine if recording is the only goal (you're capturing for later analysis, not for session resumption), but it's a significant UX regression if users rely on persistent sessions.

---

## Approach 2: kitty Kittens (The Most Programmable Path)

### Why kitty

kitty is the most extensible terminal emulator on Linux. It has:

- **Kittens**: Python plugins that run inside kitty's process
- **Remote control**: A Unix socket protocol for external process control
- **`kitty @ pipe`**: Pipe window/tab content to external programs
- **`--listen-on`**: Creates a Unix socket for scripting
- **Custom keyboard protocols** and escape sequence handling
- **Session management**: Can save/restore window layouts from config files
- **Built-in SSH integration** (`kitten ssh`) with automatic file transfer

### Architecture

```
┌─────────────────────────────────────────────┐
│ kitty                                       │
│  ┌─────────────────────────────────────────┐│
│  │ OS Window                               ││
│  │  ┌─────────┐ ┌─────────┐ ┌───────────┐ ││
│  │  │ Tab 1   │ │ Tab 2   │ │ Tab 3     │ ││
│  │  │ (local) │ │ (ssh)   │ │ (search)  │ ││
│  │  └────┬────┘ └────┬────┘ └───────────┘ ││
│  │       │            │                    ││
│  │  ads-kitten: intercepts output stream   ││
│  │       │            │                    ││
│  │       ▼            ▼                    ││
│  │   SQLite writer (per-tab session DB)    ││
│  └─────────────────────────────────────────┘│
└─────────────────────────────────────────────┘
         │
         ▼
  ~/.local/share/ads/
    ├── meta.db
    └── sessions/
        ├── <uuid1>.db
        └── <uuid2>.db
```

### Implementation Sketch

**1. The ADS kitten (`ads.py`)**:

```python
# ~/.config/kitty/kittens/ads.py
from kitty.boss import Boss
import sqlite3, os

def on_window_created(boss: Boss, window):
    """Called when a new kitty window/tab is created."""
    session_uuid = create_session_in_meta_db(window.title)
    window.ads_session_uuid = session_uuid
    window.ads_db = open_session_db(session_uuid)

def on_data_received(boss: Boss, window, data: bytes):
    """Called for every chunk of output data."""
    window.ads_db.write_chunk(data)
    # Parse OSC 133 markers inline
    parse_osc133(data, window.ads_db)
```

**The problem**: kitty's kitten API **does not have an `on_data_received` hook**. Kittens are interactive programs (like `kitten icat` for image display), not passive I/O interceptors. You can't subscribe to the output stream of another window from a kitten.

**2. The remote control approach**:

```bash
# Start kitty with remote control enabled
kitty --listen-on unix:/tmp/kitty-ads.sock

# From external process, get window content
kitty @ --to unix:/tmp/kitty-ads.sock get-text --match id:1 --extent all
```

`kitty @ get-text` can pull the current scrollback, but it's a **polling** mechanism, not streaming. You'd have to poll repeatedly, diff against previous content, and extract new lines. This is fragile and inefficient.

**3. The `kitty @ pipe` approach** (most promising):

```bash
# Pipe all output from a window to an external command
kitty @ pipe --match id:1 --to ads-recorder --session <uuid>
```

Wait — this is essentially `tmux pipe-pane` but for kitty. The difference is that kitty's pipe gives you the **rendered text** (post-VT processing), not the raw byte stream. This is actually better for FTS since you don't need ANSI stripping, but worse for raw replay.

### Trade-offs

| Aspect | Current (tmux) | kitty Integration |
|---|---|---|
| Plugin language | N/A (external process) | Python (kittens) |
| I/O capture | Raw bytes via pipe-pane | Rendered text or polling |
| Session persistence | tmux detach/reattach | ❌ No detach equivalent |
| GPU rendering | No | Yes (OpenGL) |
| SSH integration | Native ssh binary | `kitten ssh` (automatic TERMINFO) |
| Image support | Limited (sixel) | Native (icat protocol) |
| Emulator lock-in | None | kitty only |
| Community/packages | tmux is everywhere | kitty is widespread but not universal |

> [!WARNING]
> **kitty's author (Kovid Goyal) explicitly does not want kitty to become a platform for plugins that intercept I/O streams.** The kitten API is designed for interactive tools, not passive monitoring. You'd be working against the design philosophy.

---

## Approach 3: WezTerm Event-Driven Lua (The Most Promising Plugin Path)

### Why WezTerm

WezTerm is the terminal emulator **designed to be programmable**. Its entire configuration is a Lua program, and it has an event system that can hook into almost everything:

- **Pane output events** (you can subscribe to output data!)
- **Process lifecycle events**
- **Custom key bindings** that trigger Lua functions
- **Multiplexer domains** (local, SSH, serial — with built-in mux)
- **Built-in session persistence** via its multiplexer
- **Runs on Linux, macOS, Windows**

### Architecture

```lua
-- ~/.config/wezterm/wezterm.lua
local wezterm = require 'wezterm'
local ads = require 'ads'  -- your custom module

wezterm.on('pane-output', function(pane, output)
    -- Called for every chunk of output data!
    local session = ads.get_session(pane:pane_id())
    if session then
        session:write_chunk(output)
        session:scan_osc133(output)
    end
end)

wezterm.on('pane-spawned', function(pane)
    ads.create_session(pane)
end)

wezterm.on('pane-exited', function(pane)
    ads.seal_session(pane)
end)
```

**Wait — does WezTerm actually have a `pane-output` event?**

Checking: WezTerm has these relevant events:
- `update-status` — fires on status bar updates
- `window-config-reloaded`
- `user-var-changed` — fires when user vars (set via OSC 1337) change
- `bell` — fires on BEL character

**It does NOT have a generic `pane-output` event.** However, it has:

- **`user-var-changed`**: User variables set via escape sequences. You could inject breadcrumbs from shell hooks via `printf '\033]1337;SetUserVar=ads_cmd=%s\007' $(echo -n "ls" | base64)` — this gives you structured command metadata.
- **Multiplexer pane capture**: WezTerm's built-in mux can be queried for scrollback content.
- **`wezterm cli get-text`**: Similar to kitty, retrieves pane content.

### The Real WezTerm Approach: Multiplexer + CLI

WezTerm has a **built-in multiplexer** that provides tmux-like functionality (detach/reattach, split panes, tabs) natively. Combined with its CLI tool:

```bash
# Start WezTerm with mux server
wezterm-mux-server --daemonize

# Connect from any terminal
wezterm connect unix

# From the CLI, capture pane content
wezterm cli get-text --pane-id 0 --start-line -1000
```

The approach would be:
1. **Use WezTerm's mux as the session manager** (replaces tmux)
2. **Shell hooks** (`PROMPT_COMMAND` / `PS0`) inject OSC 1337 user vars with command metadata
3. **A background process polls `wezterm cli get-text`** periodically and writes to SQLite
4. **Or**: Use `wezterm cli send-text` / custom escape sequences to trigger recording

### Trade-offs

| Aspect | Current (tmux) | WezTerm Mux |
|---|---|---|
| Session persistence | tmux detach/reattach | WezTerm mux detach/reattach ✅ |
| Terminal rendering | tmux (good) | WezTerm (GPU, excellent) |
| Config language | tmux.conf | Lua (very powerful) |
| I/O capture | pipe-pane (streaming) | Polling or shell hooks (less clean) |
| Split panes | tmux native | WezTerm native |
| SSH | Native ssh | WezTerm SSH domains (built-in mux over SSH) |
| Cross-platform | Linux/macOS | Linux/macOS/Windows |
| Emulator lock-in | None | WezTerm only |

> [!NOTE]
> WezTerm's multiplexer gives you **session persistence AND native terminal rendering** — the two things that are in tension in the other approaches. The trade-off is the I/O capture is less clean than tmux pipe-pane.

---

## Approach 4: PTY Proxy Layer (Emulator-Agnostic)

This is the approach that works with **any** terminal emulator, including Konsole.

### How It Works

Instead of modifying the terminal emulator, you insert a transparent recording layer between the emulator and the shell:

```
Terminal Emulator (Konsole, kitty, anything)
    │
    ├── PTY master ←──→ PTY slave → ads-shell (proxy)
    │                                    │
    │                                    ├── PTY master ←──→ PTY slave → /bin/bash
    │                                    │
    │                                    └── SQLite writer
    │                                         ├── io_stream (raw)
    │                                         ├── fts_index (stripped)
    │                                         └── command_history (OSC 133)
    │
    └── Rendered display to user
```

**`ads-shell`** is a Go binary that:
1. Allocates a new PTY pair
2. Forks the real shell (`bash`) on the slave side
3. Sits in a read loop on the master side
4. **Forwards all I/O bidirectionally** (terminal ↔ shell)
5. **Tees all output** to the SQLite writer
6. **Parses OSC 133** markers inline
7. **Handles SIGWINCH** to resize the inner PTY

### Implementation Sketch (Go)

```go
package main

import (
    "io"
    "os"
    "os/exec"
    "os/signal"
    "syscall"

    "github.com/creack/pty"
)

func main() {
    sessionUUID := os.Args[1]
    db := openSessionDB(sessionUUID)
    defer db.Close()

    // Start real shell on a new PTY
    shell := exec.Command("/bin/bash", "--rcfile", adsRCFile())
    ptmx, err := pty.Start(shell)
    if err != nil { panic(err) }
    defer ptmx.Close()

    // Handle window resize
    ch := make(chan os.Signal, 1)
    signal.Notify(ch, syscall.SIGWINCH)
    go func() {
        for range ch {
            pty.InheritSize(os.Stdin, ptmx)
        }
    }()
    ch <- syscall.SIGWINCH // initial size sync

    // Bidirectional copy with recording
    go func() {
        // User input → shell (and optionally record input too)
        io.Copy(ptmx, os.Stdin)
    }()

    // Shell output → user + SQLite
    tee := io.TeeReader(ptmx, db.StreamWriter())
    io.Copy(os.Stdout, tee)
}
```

### Integration with Konsole

On your KDE/Fedora 44 setup:

1. **Create a Konsole profile** called "ADS Recorded" that sets the command to:
   ```
   ads-shell <session-uuid>
   ```
   Or better, set it to `ads-attach <session-name>` which looks up/creates the session and launches the proxy.

2. **The CLI `ads new`** creates a Konsole profile via `konsoleprofile` or by writing to `~/.local/share/konsole/<name>.profile`:
   ```ini
   [General]
   Command=ads-shell --session <uuid>
   Name=ADS: production-server
   Icon=utilities-terminal
   ```

3. **Opening a new Konsole tab** with this profile automatically starts a recorded session.

4. **`ads list`** / **`ads search`** / **`ads edit`** work exactly as today.

### Why This Is The Strongest Approach

```
 ┌────────────────────────────────────────────────────────┐
 │                    User's Desktop                      │
 │                                                        │
 │  ┌──────────┐  ┌──────────┐  ┌──────────┐             │
 │  │ Konsole  │  │ kitty    │  │ SSH from │             │
 │  │ (local)  │  │ (laptop) │  │ iPad     │             │
 │  └────┬─────┘  └────┬─────┘  └────┬─────┘             │
 │       │              │              │                   │
 │       └──────────────┼──────────────┘                   │
 │                      │                                  │
 │               ads-shell (PTY proxy)                     │
 │                      │                                  │
 │              ┌───────┴───────┐                          │
 │              │               │                          │
 │          Real Shell    SQLite Writer                    │
 │          (/bin/bash)   (per-session DB)                 │
 │                                                        │
 └────────────────────────────────────────────────────────┘
```

- **Works with ANY terminal emulator** — Konsole, kitty, GNOME Terminal, Alacritty, even SSH clients
- **No emulator lock-in**
- **Perfect I/O capture** — you see every byte in both directions
- **OSC 133 parsing** works naturally since you see the raw stream
- **Can record input too** (typed commands, not just output)
- **SIGWINCH forwarding** means resize works correctly

### The Trade-off: You're Back to a Custom PTY Wrapper

The original architecture doc explicitly rejected this:

> *"earlier attempts to solve this problem by building bespoke terminal emulators, complex Pseudo-Terminal (PTY) proxies... proved excessively brittle"*

But there's a difference between a **PTY proxy** and a **VT state machine**:
- A PTY proxy is a transparent pipe — it doesn't interpret escape sequences for rendering
- A VT state machine tries to track cursor position, screen state, alternate screens, etc.

The proxy approach is **much simpler** than building a terminal emulator. It's essentially what `script(1)` does, and `script` has worked reliably for decades. The Go `creack/pty` library handles the OS-level PTY mechanics.

**The real risks are:**
1. **Raw mode / terminal attribute forwarding** — `vim`, `less`, etc. change terminal modes. The proxy must forward these correctly.
2. **Signal propagation** — `SIGWINCH`, `SIGTSTP`, `SIGINT` must be forwarded to the child
3. **Job control** — `Ctrl+Z` / `fg` / `bg` through a proxy is tricky
4. **Performance** — an extra copy of every byte adds latency (but it's negligible for interactive use)

---

## Approach 5: Hybrid — PTY Proxy + Konsole Integration + Optional tmux

The most practical path combines the strengths:

```
                    ┌─────────────────────────┐
                    │   ads CLI               │
                    │                         │
                    │   ads new <name>        │──→ meta.db
                    │   ads run <name>        │
                    │   ads search <query>    │
                    │   ads edit <profile>    │
                    │   ads llm summarize     │
                    └─────────┬───────────────┘
                              │
                    ┌─────────▼───────────────┐
                    │   Session Launcher      │
                    │                         │
                    │   Detects environment:  │
                    │   ├── Konsole? → D-Bus  │
                    │   ├── kitty? → socket   │
                    │   ├── tmux? → pipe-pane │
                    │   └── Other → PTY proxy │
                    └─────────┬───────────────┘
                              │
              ┌───────────────┼───────────────┐
              │               │               │
        ┌─────▼─────┐  ┌─────▼─────┐  ┌──────▼────┐
        │ Konsole   │  │ tmux      │  │ PTY Proxy │
        │ backend   │  │ backend   │  │ backend   │
        │           │  │           │  │           │
        │ D-Bus for │  │ pipe-pane │  │ Inline    │
        │ tab mgmt  │  │ for I/O   │  │ tee to    │
        │ + PTY     │  │ capture   │  │ SQLite    │
        │ proxy for │  │           │  │           │
        │ capture   │  │           │  │           │
        └───────────┘  └───────────┘  └───────────┘
              │               │               │
              └───────────────┼───────────────┘
                              │
                    ┌─────────▼───────────────┐
                    │   Shared Data Layer     │
                    │                         │
                    │   meta.db (sessions,    │
                    │     profiles, tags)     │
                    │                         │
                    │   sessions/<uuid>.db    │
                    │     ├── io_stream       │
                    │     ├── fts_index (FTS5)│
                    │     └── command_history │
                    └─────────────────────────┘
```

### How It Works

1. **`ads new <name>`** — Creates session in `meta.db` (unchanged)
2. **`ads run <name>`** — Detects the current environment:
   - **Inside Konsole?** → Creates a new Konsole tab via D-Bus, sets the shell to `ads-shell --session <uuid>`. User stays in Konsole natively. No tmux.
   - **Inside kitty?** → Uses `kitty @ launch` to create a new window running `ads-shell`. No tmux.
   - **Inside tmux already?** → Uses the current tmux backend (pipe-pane). Unchanged.
   - **Plain terminal / SSH?** → Falls back to tmux backend or runs PTY proxy directly.
3. **`ads-shell`** — The PTY proxy recorder, used by the Konsole and kitty backends
4. **`ads search` / `ads llm`** — Unchanged, query the same SQLite databases

### Detection Logic

```go
func detectBackend() Backend {
    // Check if we're inside Konsole
    if os.Getenv("KONSOLE_DBUS_SESSION") != "" {
        return &KonsoleBackend{}
    }
    // Check if we're inside kitty
    if os.Getenv("KITTY_LISTEN_ON") != "" {
        return &KittyBackend{}
    }
    // Check if tmux is available and we want session persistence
    if _, err := exec.LookPath("tmux"); err == nil {
        return &TmuxBackend{}  // current behavior
    }
    // Fallback: inline PTY proxy
    return &PTYProxyBackend{}
}
```

### Konsole-Specific Niceties

Since you're on KDE/Fedora 44, the Konsole backend can:

- **Auto-generate Konsole profiles**: Write `.profile` files to `~/.local/share/konsole/` so ADS sessions appear in Konsole's profile dropdown
- **Set tab titles** via D-Bus to show session name + status
- **Use Konsole's native split-pane** features (horizontal/vertical splits within the same window)
- **Leverage Konsole's built-in OSC 133 support**: Konsole already understands semantic shell integration markers — it highlights the prompt, shows exit codes in the scrollbar. Your recording layer captures the same markers.
- **Integrate with KDE Activities**: Different ADS profiles could be associated with different KDE Activities

---

## Data Model (Shared Across All Backends)

The SQLite data model stays the same regardless of backend:

```sql
-- meta.db (unchanged)
CREATE TABLE sessions (
    uuid TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL DEFAULT 'local',     -- local | remote | konsole | kitty
    status TEXT NOT NULL DEFAULT 'created', -- created | running | sealed
    backend TEXT NOT NULL DEFAULT 'tmux',   -- tmux | konsole | kitty | pty-proxy
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    remote_user TEXT,
    remote_host TEXT,
    remote_port INTEGER,
    profile_name TEXT NOT NULL DEFAULT 'default'
);

-- per-session .db (unchanged)  
CREATE TABLE io_stream (...);
CREATE VIRTUAL TABLE fts_index USING fts5(text);
CREATE TABLE command_history (...);
CREATE TABLE metadata (...);
```

The only addition is a `backend` column so the system knows which mechanism to use for reattach.

---

## Comparison Matrix

| Feature | tmux (current) | Konsole + PTY Proxy | kitty + PTY Proxy | WezTerm Mux | Pure PTY Proxy |
|---|---|---|---|---|---|
| **Session persistence (detach/reattach)** | ✅ Native | ❌ Not possible | ❌ Not possible | ✅ Native mux | ❌ Not possible |
| **Terminal rendering quality** | Good (256-color, some glitches) | Excellent (GPU, native) | Excellent (GPU) | Excellent (GPU) | Native to host emulator |
| **Works from SSH** | ✅ | ❌ Need X11/Wayland | ❌ Need X11/Wayland | ✅ (mux server) | ✅ |
| **Emulator lock-in** | None | Konsole | kitty | WezTerm | None |
| **I/O capture fidelity** | Raw bytes (pipe-pane) | Raw bytes (PTY proxy) | Raw bytes (PTY proxy) | Polling (lossy) | Raw bytes (inline) |
| **OSC 133 support** | Parsed by recorder | Parsed by recorder + Konsole native | Parsed by recorder | User vars only | Parsed by recorder |
| **Image protocol support** | Sixel only | iTerm2/kitty | Native | Native | Depends on host |
| **Split pane recording** | Per-pane ✅ | Per-tab only | Per-window only | Per-pane (polling) | Single session |
| **Window management** | tmux windows | KDE/Konsole tabs | kitty tabs/windows | WezTerm tabs | Host emulator |
| **Remote sessions** | ssh inside tmux | ssh inside ads-shell | kitten ssh | WezTerm SSH domain | ssh inside ads-shell |
| **Implementation effort** | Done ✅ | Medium (PTY proxy + D-Bus) | Medium (PTY proxy) | High (Lua + polling) | Medium (PTY proxy) |
| **Dependencies** | tmux | Konsole + Go PTY | kitty + Go PTY | WezTerm | Go PTY only |

---

## My Recommendation

### For your KDE/Fedora 44 setup specifically:

**Go with Approach 5 (Hybrid) with the PTY proxy as the primary capture mechanism and Konsole integration for UX.**

Here's why:

1. **The PTY proxy is the right core abstraction.** It works everywhere, captures everything, and is simpler than it sounds — `creack/pty` + `io.TeeReader` handles 90% of the complexity. The risk called out in the architecture doc (about "brittle PTY proxies") applies to proxies that try to *interpret* the stream (VT state machines). A transparent tee is fundamentally simpler.

2. **Konsole D-Bus integration is pure UX polish.** Use it to create tabs, set titles, manage profiles — but don't rely on it for capture. The PTY proxy does that.

3. **Keep tmux as a backend option** for SSH-only servers and session persistence. Some workflows genuinely need detach/reattach.

4. **The data model doesn't change.** Same `meta.db`, same per-session SQLite, same FTS5, same `command_history`. The recording method is an implementation detail that doesn't affect the query/search/LLM layer at all.

### What you lose vs. current design:

- **Session persistence** when using the Konsole backend (no detach/reattach). Mitigate by: keeping tmux as a fallback backend, or using Konsole's session restore feature (which reopens tabs with the same commands on startup).

### What you gain:

- **Native terminal rendering** — no tmux rendering layer, full GPU acceleration, proper font rendering, ligatures, image protocols
- **Native KDE integration** — tabs in Konsole, KDE window management, Activities, notifications
- **No tmux dependency for local sessions** — one less moving part
- **Input recording** — the PTY proxy sees both directions, so you can record what the user typed (not just output). This is huge for audit/compliance.
- **Emulator flexibility** — same `ads-shell` works if you switch to kitty, Alacritty, or anything else tomorrow

### Implementation Order:

1. Build `ads-shell` (the PTY proxy recorder) — ~300 lines of Go
2. Verify it works with Konsole manually (just set a profile's command to `ads-shell`)
3. Add `ads run --backend=pty` to use it instead of tmux
4. Add Konsole D-Bus integration for tab creation/management
5. Keep `ads run --backend=tmux` as default for backward compatibility
6. Gradually make PTY proxy the default as confidence grows

> [!TIP]
> The PTY proxy approach also opens the door to **input recording** and **keystroke-level timing**, which enables replay at original speed — a feature that's impossible with the current output-only `pipe-pane` capture.
