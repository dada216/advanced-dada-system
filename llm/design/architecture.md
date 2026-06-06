# Architectural Blueprint: CLI-First, Multiplexer-Driven Terminal Analytics Platform

## 1. Executive Summary

The contemporary infrastructure landscape demands a paradigm shift in how systems administrators, DevOps engineers, and security analysts interface with terminal environments. Traditional terminal emulators and logging mechanisms are insufficient for modern compliance, automation, and analytical needs. However, earlier attempts to solve this problem by building bespoke terminal emulators, complex Pseudo-Terminal (PTY) proxies, or monolithic background daemons proved excessively brittle and operationally heavy.

This document outlines a radically simplified, decentralized architectural design for a terminal session management and analytics platform. It returns to core Unix philosophy: composing simple, battle-tested tools. The platform discards custom PTY wrappers, Virtual Terminal (VT) state machines, and centralized daemons in favor of standard `tmux` pipelines and the native `ssh` binary.

By utilizing `tmux pipe-pane` as the core capture mechanism, the architecture guarantees mathematically perfect rendering and seamless terminal resizing without a single line of custom emulation code. It implements a strict database-per-session model using SQLite for high-performance, segmented auditing. Semantic context is preserved for Large Language Models (LLMs) via Operating System Command (OSC) 133 markers injected via shell hooks. The entire system is operated exclusively through a robust Command Line Interface (CLI), establishing an omniscient, AI-ready data layer that is uncompromisingly stable.

---

## 2. Decentralized Execution Topology

To achieve absolute stability and eliminate single points of failure, the architecture avoids a centralized, always-on background daemon. Instead, the application consists of two lightweight, purpose-built CLI binaries (which may be shipped as a single executable with different subcommands):

1. **The Orchestrator (`ads`)**: The user-facing CLI tool responsible for managing configuration, launching sessions, and querying data.
2. **The Pipeline Recorder (`ads-recorder`)**: A headless, ephemeral background process spawned dynamically per-session. It reads raw bytes from standard input (`stdin`), parses them, and writes them to SQLite.

Because there is no centralized daemon managing multiple PTYs, a failure in one recording pipeline is completely isolated. If a recorder crashes, the user's terminal session remains active and uncorrupted; only the logging for that specific pane stops.

---

## 3. Two-Phase Session Lifecycle

The architecture enforces a strict separation between configuring a session and actively running it. This prevents polluted execution environments and ensures that SSH connections and credentials are comprehensively validated before recording begins.

### Phase 1: Configuration (`ads new`)
When a user provisions a new session, the Orchestrator provisions an isolated SQLite database and drops the user into an ephemeral configuration shell.
* Local session: `ads new <session-name>`
* Remote session: `ads new --remote <session-name>`

Within this shell, dedicated `ads` subcommands are injected into the `$PATH`. The user configures target hosts, tags, and tests credentials (e.g., `ads auth test --use-agent`). This interactive phase guarantees that the native SSH agent is configured correctly. Once configured, the user exits the shell, persisting the verified configuration to the Meta-Database.

### Phase 2: Execution and Recording (`ads run`)
To launch a session, the user executes `ads run <session-name>`.
The Orchestrator executes the following sequence:
1. Spawns standard `tmux` in the background (`tmux new-session -d -s <name>`).
2. Inside `tmux`, it launches the target environment: either a local bash shell with `--rcfile` hooks, or the native `ssh` binary pointing to the remote host.
3. It immediately initiates `tmux pipe-pane -o 'ads-recorder --session <name>'`.
4. It attaches the user's current terminal to the `tmux` session (`tmux attach -t <name>`).

---

## 4. Native SSH Orchestration and Context Boundaries

By invoking the operating system's native `ssh` binary rather than implementing an SSH client in code (e.g., via `crypto/ssh`), the architecture inherits full SSH feature parity for free.

The platform automatically supports:
* `~/.ssh/config` parsing and aliases
* `known_hosts` strict host key checking
* Complex authentication (hardware security keys, `ed25519-sk`, keyboard-interactive prompts)
* SSH Agent forwarding (`-A`)
* ProxyJump and Bastion hosts (`-J`)

Because remote sessions are defined as explicit configurations during Phase 1, there is no need to monitor active local shells for typed `ssh` commands. Local and remote contexts are cleanly isolated from birth. Furthermore, this approach abandons the unsafe practice of injecting secrets via environment variables; the native `ssh` binary directly leverages the user's SSH agent or OS keyring securely.

---

## 5. The Multiplexer Capture Pipeline

The most critical functional requirement is the precise capture of terminal input and output. The architecture delegates terminal emulation entirely to `tmux`.

When the Orchestrator initiates `tmux pipe-pane`, `tmux` streams every raw byte outputted by the shell directly into the `ads-recorder` process via standard input.
* **Zero Emulation Bugs:** Because the user interacts directly with `tmux`, complex applications like `vim`, `htop`, and split-panes work flawlessly.
* **Free Persistence:** `tmux` intrinsically supports session detachment. If a network drop occurs, the session and the recorder continue running. The user simply types `ads run <name>` to seamlessly reattach.
* **Accurate Resizing:** `SIGWINCH` resize events are handled natively by `tmux` and the user's host terminal emulator. The architecture does not need to intercept or forward these signals.

---

## 6. Semantic Output Structuring via OSC 133

To enable LLMs and analytical tools to parse raw terminal output intelligently, the text stream must be structurally categorized.

The platform utilizes Operating System Command (OSC) 133 escape sequences, injected via shell integration scripts (e.g., `bash-preexec` or native Zsh hooks).

| OSC Sequence | Semantic Function |
| :--- | :--- |
| `OSC 133; A ST` | Start of Prompt |
| `OSC 133; B ST` | Start of User Input |
| `OSC 133; C ST` | Start of Command Output |
| `OSC 133; D ST` | End of Command Output (includes exit code) |

As the `ads-recorder` receives raw bytes from `tmux`, a lightweight, asynchronous scanner monitors the stream for these specific OSC 133 markers. When detected, the recorder segments the output and tags it within the SQLite database.

**Graceful Degradation:** The architecture recognizes that shell hooks are brittle (e.g., bypassed by `sudo su` or raw-mode applications). The OSC 133 scanner is explicitly designed as a *best-effort* semantic layer. If markers are absent, the platform falls back to raw temporal indexing, ensuring the session remains fully recorded and searchable.

---

## 7. The Database-Per-Session SQLite Model

Storing heavy, continuous terminal streams in a monolithic database creates severe concurrency bottlenecks. The architecture strictly enforces a Database-per-Session paradigm utilizing SQLite in Write-Ahead Log (WAL) mode.

1. **The Meta-Database**: A centralized SQLite file storing high-level configurations, tags, server profiles, and an index of all session database UUIDs.
2. **Session Databases**: A dedicated `.db` file for every execution. Deleting a session is an `unlink()` system call, avoiding expensive SQL purges.

### The ANSI-Stripping FTS Pipeline
Raw terminal output contains ANSI color codes and cursor movements, making standard SQL Full-Text Search (FTS) extremely noisy.
To solve this, the `ads-recorder` implements a dual-write pipeline:
1. The exact, raw bytes are written to the `io_stream` table for perfect visual replay.
2. An asynchronous middleware strips all ANSI sequences and terminal control characters, writing the sanitized plaintext to the `fts_index` virtual table. This ensures pristine, highly accurate keyword searching across millions of lines of history.

---

## 8. Federated Query and Analytics

With data distributed across discrete SQLite files, the `ads` CLI implements a Federated Query Layer.

When a user executes a search (`ads search --tag production "Out of memory"`), the CLI:
1. Queries the Meta-Database to identify the relevant session UUIDs.
2. Spawns bounded, concurrent worker goroutines to query the targeted session databases dynamically.
3. Aggregates and returns the results to standard output.

Because the interface is a CLI, users can pipe these aggregated results natively into `grep`, `jq`, or `awk`.

---

## 9. Modular Extensibility and Automation

### HashiCorp Go-Plugin Architecture
To mitigate operational risks from dynamically loaded libraries, the platform employs a multi-process plugin model over gRPC (via Unix Domain Sockets), similar to Terraform. Plugins run as isolated binaries, capable of subscribing to the Meta-Database or querying session databases without jeopardizing the core CLI's stability.

### LLM and Time-Tracking Services
Background analytical services can periodically scan newly sealed session databases.
* **LLM Analysis:** Driven by the OSC 133 semantic chunks, LLM plugins can extract errors, summarize sessions, or generate incident playbooks with minimal token overhead and low hallucination risk.
* **Time Tracking:** Plugins can analyze temporal gaps between `OSC 133; B` (Input) markers to generate accurate billing and compliance reports.

### Ansible Runner Integration
For automated configuration management, the platform delegates entirely to the Ansible Runner API. Runner executes playbooks within isolated container execution environments. The `ads` CLI suppresses raw standard output, instead capturing the highly structured Runner Artifact Job Events (JSON streams) and logging them directly into the session databases for querying.

---

## 10. Coding Agent Policies & Development Standards

To ensure code quality, maintainability, and seamless collaboration with AI coding assistants, the project strictly adheres to the following development policies:

### Makefile-Driven Workflows
All build, test, and linting operations must be encapsulated within a `Makefile`. This provides a unified interface for both human developers and autonomous agents. Agents should rely on executing `make` targets (e.g., `make build`, `make lint`, `make test`) rather than constructing complex shell commands manually.

### Golangci-lint v2 Enforcement
Code quality is enforced using `golangci-lint` (v2). 
- The `Makefile` must include a `lint` target that executes `golangci-lint run`.
- Agents must run the `lint` target and resolve all reported issues before concluding a task or committing code.
- The configuration for `golangci-lint` should be explicitly defined in a `.golangci.yml` file to enforce consistent rules across all environments.

### Gitflow and Commits
Agents must follow standard gitflow practices for branching and committing. Commits should be atomic, well-documented, and created using predefined project conventions. Agents should utilize internal skills (like `mastering-git-cli`) to handle VCS operations correctly.

### Privileged Host Execution and Auditing
When an agent requires execution of commands directly on the user's host (e.g., executing system-level package upgrades via `dnf`), the execution MUST be routed exclusively through the `scripts/host-exec.sh` wrapper script. This guarantees:
1. **Security & Auditing:** Every host-level command is rigorously intercepted and recorded in a human-readable format to `docs/security/host_exec.log`.
2. **Namespace Injection:** The agent container leverages `--privileged` and `--pid=host` to perform an `nsenter` breakout. This eliminates the need for complex SSH agent configurations or `sudoers` password interventions, allowing autonomous, transparent, and strictly audited host interactions.

---

## 11. Conclusion

By abandoning the monolithic daemon model and discarding custom terminal emulation in favor of `tmux pipe-pane` and native `ssh`, this architecture achieves absolute stability, zero-emulation-bug rendering, and free session persistence. 

It tightly couples these battle-tested Unix primitives with a modern, high-performance SQLite-per-session data layer. The result is a radically simplified, highly resilient CLI platform perfectly positioned to act as the foundational, AI-ready analytical layer for next-generation systems administration.
