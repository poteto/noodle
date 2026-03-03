# Noodle Architecture & Vision

## Elevator Pitch

Noodle is an open-source agent orchestration framework written in Go. The core insight is that agents already know how to read and write files — so the entire system's API is just files. Skills (markdown files with YAML frontmatter) are the only extension point. An LLM decides what to work on, other LLMs do the work and review it, and the system learns from each cycle.

## The Three Big Ideas

### 1. Everything is a file

All state lives in `.noodle/` — orders, session events, status, mise context. There's no SDK, no RPC protocol, no integration layer. An agent can read `orders.json` and understand the system's state because it's just JSON. This makes the system trivially debuggable (you can `cat` any file to see what's happening) and naturally composable — agents coordinate by writing files, not by calling APIs.

### 2. Skills as the only extension point

A skill is a markdown file (`SKILL.md`) that tells an agent how to do something. When a skill has a `schedule:` field in its frontmatter, it becomes a schedulable task type. This is the *only* way to extend Noodle — no plugin system, no hooks API, no custom Go code. Skills are to Noodle what components are to React. You compose them to build your workflow.

### 3. LLM judgment / Go mechanics split

Go handles everything deterministic: the main loop, file watching, session spawning, worktree management, health monitoring. LLMs handle everything that requires judgment: deciding what to work on next, doing the actual coding work, reviewing quality. This split means the framework is predictable and testable while the agents are flexible and adaptive.

---

## Architecture Walkthrough

### The Main Loop (`loop/loop.go`)

The core is a single Go event loop that runs cycles. Each cycle has two phases:

**Maintenance phase** (`runCycleMaintenance`):

1. **Check registry errors** — bail if the skill registry has failed 3+ consecutive times
2. **Drain completions** — process completed sessions (includes adopted session monitoring)
3. **Drain merge results** — process completed merge operations
4. **Process control commands** — handle commands from `control.ndjson`
5. **Drain completions again** — second pass to catch completions triggered by control commands
6. **Drain merge results again** — second pass for merge results

**Main pipeline**:

7. **Rebuild skill registry** if stale (fsnotify watcher sets `registryStale` flag)
8. **Build cycle brief** — gather all context into `mise.json` (includes `refreshAdoptedTargets()`)
9. **Prepare orders for cycle**:
   - **Consume orders** — atomically promote `orders-next.json` into `orders.json`
   - **Normalize and validate** — apply routing defaults, strip invalid stages
   - **Bootstrap schedule** — if no orders exist and no sessions running, spawn a schedule agent
   - **Inject schedule on mise change** — add schedule order if `mise.json` changed mid-cycle
   - **Apply routing defaults** — fill in runtime/mode defaults
10. **Plan spawns** — find dispatchable stages, respect concurrency limits, avoid busy targets, check merge backpressure
11. **Spawn sessions** — launch agents as child processes, persist stage status as `active` before spawning
12. **Flush state** — write all in-memory state to disk
13. **Publish state** — update in-memory snapshot for the web UI
14. **Stamp status** — write current state to `status.json`

### The Work Orders Model

Work is organized as **orders** containing ordered **stages**. An order is a pipeline — for example, `execute → quality → reflect`. The loop advances stages mechanically: when a stage completes and merges, the next stage in the order becomes dispatchable. No LLM runs between stages.

```
Order "implement-feature-29"
  Status: active
  ├── Stage 0: execute  [completed]
  ├── Stage 1: quality  [active]     ← currently running
  └── Stage 2: reflect  [pending]    ← dispatches after quality merges
```

**Order statuses:** `active`, `completed`, `failed`.

**Stage statuses:** `pending`, `active`, `merging`, `completed`, `failed`, `cancelled`.

If a stage fails, the loop marks the order as `failed` in `orders.json` and forwards the failure to the active scheduler session via `controller.SendMessage`. The scheduler decides recovery — it can enqueue a new order with debugging/retry stages, or skip the work entirely. Failed orders stay in `orders.json` with `status: failed` so the scheduler sees them as context and dispatch skips them (only `active` orders are dispatchable).

### The Data Flow

```
mise.json --> [Schedule Agent] --> orders-next.json
                                       |
                                  loop merges into
                                       |
                                       v
                                  orders.json
                                       |
                               loop dispatches next
                               dispatchable stage
                                       |
                                       v
                             [Cook Agent in worktree]
                                       |
                                  completes
                                       |
                            ┌──────────┴──────────┐
                            v                      v
                     mode = supervised         mode = auto
                     or manual                      |
                            |                 merge to main
                     park for human                 |
                     review (pendingReview)    advance order
                                              to next stage
```

**Mise** ("mise en place") is the context-gathering step. It builds a brief containing the backlog, active sessions, recent history, available capacity, plans, and routing config. The schedule agent reads this and writes `orders-next.json` — full orders with all their stages laid out upfront.

**Cooks** run in isolated git worktrees — never the primary checkout. Each cook gets its own branch. This means you can run multiple agents in parallel without conflicts. Inter-stage context flows through main — each stage merges to main on success, and the next stage's worktree is created from main, picking up the previous stage's output.

**Mode** controls how much human oversight the loop enforces. In `auto` mode, completed stages merge automatically. In `supervised` mode, completed stages park in `pendingReview` for human approval. In `manual` mode, scheduling and dispatch also require explicit human action. Quality is a stage in the order pipeline — if it fails, the stage fails and the scheduler decides recovery.

### Key Go Packages

| Package | Responsibility |
|---------|---------------|
| `loop/` | Main cycle orchestration, stage lifecycle, order advancement |
| `internal/orderx/` | `Order`/`Stage`/`OrdersFile` types and atomic I/O |
| `internal/taskreg/` | Task type registry (maps skill frontmatter to dispatchable types) |
| `internal/state/` | Canonical state model (`State`, `OrderNode`, `StageNode`, `AttemptNode`) |
| `internal/ingest/` | Event normalization and idempotency (canonical event sequence IDs) |
| `internal/reducer/` | Deterministic state transitions and effect engine |
| `internal/dispatch/` | Order/stage transition planning (pure routing helpers) |
| `internal/mode/` | Unified mode control (`auto`/`supervised`/`manual`) and epoch-stamped effects |
| `internal/projection/` | Projects canonical state into files, snapshots, and WebSocket deltas |
| `internal/rtcap/` | Runtime capability contract (steerable, polling, remote-sync, heartbeat) |
| `internal/snapshot/` | UI/API snapshot construction from loop state |
| `internal/statever/` | Schema version tracking for state files |
| `mise/` | Context gathering (brief builder) |
| `dispatcher/` | Session lifecycle: process spawning, skill bundling, prompt composition, bidirectional streaming |
| `runtime/` | Runtime abstraction layer (`Runtime` interface with `Dispatch`, `Kill`, `Recover`) |
| `skill/` | Skill resolution and frontmatter parsing |
| `monitor/` | Session health (stuck detection, heartbeats, composite observers) |
| `worktree/` | Git worktree lifecycle |
| `adapter/` | Backlog sync (shell scripts that bridge external systems) |
| `event/` | NDJSON append-only event logging |
| `parse/` | Provider-agnostic event parsing (Claude/Codex NDJSON → canonical events) |
| `stamp/` | Timestamps NDJSON lines and emits canonical sidecar events |
| `server/` | Web UI backend with WebSocket streaming |
| `config/` | `.noodle.toml` loading and validation |
| `startup/` | First-run project structure initialization (`brain/`, `.noodle/`) |

### Skills

Skills become schedulable by adding a `schedule:` field to their frontmatter — that's the only required metadata:

```yaml
---
name: deploy
description: Deploy after successful execution on main
schedule: "After a successful execute completes on main branch"
---
```

Executing agents invoke domain skills directly via `Skill()` calls — there's no special frontmatter for domain skill bundling.

The minimal autonomous system is two skills:

- **schedule** — reads the backlog and decides what to work on next
- **execute** — picks up a queued item and does the work

From there you add more skills to make it smarter:

- **quality** — reviews completed work
- **reflect** — after each session, writes what it learned to the brain
- **meditate** — extracts higher-level principles from accumulated learnings and encodes them back into skills

### The Brain (Self-Improvement)

The `brain/` directory is an Obsidian vault that persists across sessions. After a cook session, the reflect skill captures learnings and routes them to the right place — brain notes for knowledge, skill updates for behavior changes, todos for follow-up work. The key principle is "encode lessons in structure": a lint rule, a frontmatter flag, or a runtime check beats a textual instruction that gets ignored.

After several reflect cycles, meditate does a full vault audit — spawns subagents to find stale notes, extract cross-cutting principles, and update skills. The skills literally get better over time because agents rewrite them based on accumulated experience.

### Web UI

React + TypeScript + TanStack Router. The Go server streams real-time updates via WebSockets — the server watches `.noodle/` via fsnotify and broadcasts snapshot updates to connected clients with 300ms debounce. The layout is feed-based: a sidebar with agents and orders as an expandable tree, a main chat feed panel (Slack-style session logs), and a context panel with metrics, files touched, and a stage rail showing pipeline progress. A review tab provides a diff viewer with approve/merge/reject actions. The UI is embedded in the Go binary via `embed.FS`.

---

## Design Essentials

### Concurrency Model

Git worktrees for code isolation, busy-target tracking for coordination, merge locks for serialization.

**Worktree lifecycle:** Create (dedicated branch + working directory) → Execute (agent works in worktree, never primary checkout) → Merge (PID-based file lock, rebase onto main, fast-forward) → Cleanup (remove worktree, delete branch, prune).

**Busy-target tracking:** The loop maintains maps (`activeCooksByOrder`, `adoptedTargets`, `adoptedSessions`, `pendingReview`) plus stage-status and failed-order derivation. An order is skipped if its ID appears in any source.

**Merge conflicts** surface during rebase and are forwarded to the scheduler, then parked for human review. Recoverable, not fatal.

### Crash Safety

**Orders promotion:** Schedule writes to `orders-next.json` (staging area). The loop merges into `orders.json` by order ID — duplicates are skipped for crash safety. Stage status is persisted as `active` *before* spawning, so restart + adoption reconciles orphaned stages.

**Session adoption:** On startup, scan sessions, check PID liveness, adopt anything still running. Loop crash loses zero work.

**Control commands:** Append-only `control.ndjson` with `flock()`, idempotent replay via `processedIDs`, acknowledgments to `control-ack.ndjson`. Web UI provides the same via HTTP API and WebSocket.

### Cost and Runaway Protection

- **Schedule-skip:** Only spawn scheduler when no orders and no sessions.
- **Failed order tracking:** `status: failed` orders stay in `orders.json` as context; only `active` orders dispatch.
- **Scheduler-driven recovery:** No automatic retry — the LLM decides recovery via `controller.SendMessage`.
- **Health monitoring:** Stuck detection (>120s idle), context budget warnings (80% of 200K tokens).
- **Mode dial:** `auto` / `supervised` / `manual` with monotonic epoch-stamped effects.
- **Control commands:** `pause`, `resume`, `drain`, `stop-all`, `kill`, `steer`, `merge`, `reject`, `set-max-concurrency`, etc.

### Provider Integration

Go doesn't talk to LLMs directly. Noodle spawns provider CLIs (`claude`, `codex`) as child processes. The `stamp` sidecar pipes stdout through `stamp.Processor` to produce `raw.ndjson` + `canonical.ndjson`. The `parse/` package normalizes provider-specific NDJSON formats to canonical events.

Prompts are assembled at dispatch: preamble (Noodle context) + skill bundle (`SKILL.md` + `references/*.md`) + task prompt. For Claude: `--append-system-prompt` + `--input-format stream-json`. For Codex: inlined via stdin.

Runtimes: `process` (default, local child processes), `sprites` (remote VMs), `cursor` (stub). Configured per-stage or via `.noodle.toml` routing defaults. `internal/rtcap` provides capability queries instead of runtime branching.

### Skill Hot-Reload

Four-layer defense: (1) fsnotify watcher with debounce → registry rebuild, (2) belt-and-suspenders check at dispatch → force rebuild, (3) `NormalizeAndValidateOrders` strips unknown task types, (4) bootstrap fallback → spawn bootstrap agent if schedule skill is missing (3 attempts, then degrade).

A missing skill is degraded operation, not a crash.
