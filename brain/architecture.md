# Noodle Architecture & Vision

## Elevator Pitch

Noodle is an open-source agent orchestration framework written in Go. The core insight is that agents already know how to read and write files — so the entire system's API is just files. Skills (markdown files with YAML frontmatter) are the only extension point. An LLM decides what to work on, other LLMs do the work and review it, and the system learns from each cycle.

## The Three Big Ideas

### 1. Everything is a file

All state lives in `.noodle/` — orders, session events, status, mise context. There's no SDK, no RPC protocol, no integration layer. An agent can read `orders.json` and understand the system's state because it's just JSON. This makes the system trivially debuggable (you can `cat` any file to see what's happening) and naturally composable — agents coordinate by writing files, not by calling APIs.

### 2. Skills as the only extension point

A skill is a markdown file (`SKILL.md`) that tells an agent how to do something. When a skill has `noodle:` frontmatter, it becomes a schedulable task type. This is the *only* way to extend Noodle — no plugin system, no hooks API, no custom Go code. Skills are to Noodle what components are to React. You compose them to build your workflow.

### 3. LLM judgment / Go mechanics split

Go handles everything deterministic: the main loop, file watching, session spawning, worktree management, health monitoring. LLMs handle everything that requires judgment: deciding what to work on next, doing the actual coding work, reviewing quality. This split means the framework is predictable and testable while the agents are flexible and adaptive.

---

## Architecture Walkthrough

### The Main Loop (`loop/loop.go`)

The core is a single Go event loop that runs cycles. Each cycle:

1. **Rebuild skill registry** if any skill files changed (fsnotify watcher)
2. **Process control commands** (human can steer via `control.ndjson`)
3. **Collect completed sessions** — check which agents finished
4. **Run monitor** — detect stuck sessions, check heartbeats
5. **Process pending retries** — re-attempt stages that failed to dispatch
6. **Build cycle brief** — gather all context into `mise.json`
7. **Consume orders** — atomically promote `orders-next.json` into `orders.json`
8. **Normalize and validate** — apply routing defaults, strip invalid stages
9. **Bootstrap schedule** — if no orders exist and no sessions are running, spawn a schedule agent
10. **Plan spawns** — find dispatchable stages, respect concurrency limits, avoid busy targets
11. **Spawn sessions** — launch agents in tmux (or remote VMs), persist stage status as `active` before spawning
12. **Stamp status** — write current state to `status.json`

### The Work Orders Model

Work is organized as **orders** containing ordered **stages**. An order is a pipeline — for example, `execute → quality → reflect`. The loop advances stages mechanically: when a stage completes and merges, the next stage in the order becomes dispatchable. No LLM runs between stages.

```
Order "implement-feature-29"
  Status: active
  ├── Stage 0: execute  [completed]
  ├── Stage 1: quality  [active]     ← currently running
  └── Stage 2: reflect  [pending]    ← dispatches after quality merges

  OnFailure:
  ├── Stage 0: debugging [pending]   ← runs if any main stage fails
  └── Stage 1: execute   [pending]
```

If a stage fails, the order's `OnFailure` pipeline kicks in — the order transitions to `"failing"` status and the recovery stages dispatch. If there's no `OnFailure`, remaining stages are cancelled and the order is removed.

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
                     quality verdict?         no verdict
                     (.noodle/quality/)        or accepted
                            |                      |
                     reject → fail stage      merge to main
                     (triggers OnFailure)          |
                                              advance order
                                              to next stage
```

**Mise** ("mise en place") is the context-gathering step. It builds a brief containing the backlog, active sessions, recent history, available capacity, plans, and routing config. The schedule agent reads this and writes `orders-next.json` — full orders with all their stages laid out upfront.

**Cooks** run in isolated git worktrees — never the primary checkout. Each cook gets its own branch. This means you can run multiple agents in parallel without conflicts. Inter-stage context flows through main — each stage merges to main on success, and the next stage's worktree is created from main, picking up the previous stage's output.

**Quality** reviews completed work. In `auto` autonomy mode, the loop checks for a quality verdict file (`.noodle/quality/<session-id>.json`) before merging. A rejection triggers the order's failure routing. In `approve` mode, all completed stages park for human review regardless of verdict.

### Key Go Packages

| Package | Responsibility |
|---------|---------------|
| `loop/` | Main cycle orchestration, stage lifecycle, order advancement |
| `internal/orderx/` | Canonical `Order`/`Stage`/`OrdersFile` types and atomic I/O |
| `internal/taskreg/` | Task type registry (maps skill frontmatter to dispatchable types) |
| `mise/` | Context gathering (brief builder) |
| `dispatcher/` | Session spawning (tmux, remote) |
| `skill/` | Skill resolution and frontmatter parsing |
| `monitor/` | Session health (stuck detection, heartbeats) |
| `worktree/` | Git worktree lifecycle |
| `adapter/` | Backlog sync (shell scripts that bridge external systems) |
| `event/` | NDJSON append-only event logging |
| `server/` | Web UI backend with SSE streaming |
| `config/` | `.noodle.toml` loading and validation |

### Skills

Skills become schedulable by adding frontmatter:

```yaml
---
name: deploy
description: Deploy after successful execution on main
noodle:
  permissions:
    merge: false
  schedule: "After a successful execute completes on main branch"
---
```

Task types that need codebase context declare it in frontmatter — the loop reads `domain_skill` from the registry at dispatch time and bundles the referenced skill:

```yaml
---
name: execute
description: Pick up a work item and implement it
noodle:
  domain_skill: go-best-practices
  schedule: "When a backlog item is ready for implementation"
---
```

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

React + TypeScript + TanStack Router. The Go server streams real-time updates via SSE. It's a kanban board (queue -> active -> review -> done) with a Slack-style chat panel to inspect session logs. The UI is embedded in the Go binary.

---

## Design Deep-Dives

### Why files instead of a database or message queue?

**The core argument:** Agents already know how to read and write files. Every LLM can do it natively — it's the one capability every agent shares regardless of provider. A database requires an SDK. A message queue requires a protocol. Files require nothing.

**Concrete benefits:**

- **Zero integration cost.** A new provider (Claude, Codex, Gemini, whatever) works immediately. No client library, no API adapter. If it can read a file, it can participate in Noodle.
- **Composition is free.** One agent's output is another's input. The schedule agent writes `orders-next.json`, the loop merges it into `orders.json`, stages dispatch from there. No pub/sub, no consumer groups — just files.
- **Trivially debuggable.** You can `cat .noodle/orders.json` and see exactly what's queued — every order, every stage, their statuses. You can `cat .noodle/sessions/{id}/canonical.ndjson` and replay an entire session. Try doing that with a Kafka topic.
- **Crash-safe without transactions.** Every state mutation uses atomic rename — write to `.tmp`, `os.Rename()` to the real path. POSIX guarantees atomicity. Readers always see complete valid JSON or the previous version. No WAL, no fsync dance.

**But doesn't that have race conditions?**

The key insight is **single-writer discipline**. The loop is the only process that writes `orders.json`. Schedule agents write to a separate file (`orders-next.json`) — a staging area. The loop merges incoming orders into the existing file, skipping duplicates by order ID for crash safety, then deletes `orders-next.json`. There's no TOCTOU window because the loop reads the bytes once, validates in-memory, then does a single atomic write.

Control commands use a different pattern — `control.ndjson` is append-only with file locking (`flock()`), and each command gets an ID tracked in a `processedIDs` map for idempotent replay. The loop truncates the file after consuming all commands and writes acknowledgments to a separate `control-ack.ndjson`.

**What about performance at scale?**

The `.noodle/` directory is small. Orders file has maybe 10-20 orders. Session directories accumulate but events are per-session files, not a single growing log. The real bottleneck is agent execution time (minutes), not file I/O (microseconds). If you're running 20 agents in parallel, the files they read are tiny JSON blobs — this never becomes the bottleneck.

Also, fsnotify means the loop doesn't poll files blindly — it triggers cycles on specific file changes (`orders-next.json`, `control.ndjson`) and falls back to a ticker for reliability. It's event-driven with a polling safety net.

**How does Noodle handle schema evolution?**

Right now: you just change the struct and the JSON changes with it. No backward compatibility by default — that's a stated principle. No `omitempty` shims, no legacy fallbacks. This works because the loop is the single writer/reader of most files, and agents are stateless between sessions. A running agent reads `mise.json` once at the start. If the schema changes between sessions, the next agent gets the new schema.

---

### How does Noodle handle concurrency?

**The short answer:** Git worktrees for code isolation, busy-target tracking for coordination, merge locks for serialization.

**Worktree lifecycle:**

1. **Create** — `git worktree add .worktrees/<name> -b noodle/<name>`. Each cook gets a dedicated branch and working directory. Dependencies are installed in the worktree. Claude settings are symlinked for shared config.
2. **Execute** — The agent works entirely within the worktree. It can't touch the primary checkout. The tmux session's working directory is set to the worktree path.
3. **Merge** — Acquire a PID-based file lock (`.worktrees/.merge-lock`). Rebase the worktree branch onto main. Fast-forward merge into main. Only one merge happens at a time.
4. **Cleanup** — Remove the worktree directory, delete the branch, `git worktree prune`.

**Busy-target tracking is multi-layered:**

The loop maintains six maps that prevent duplicate work on the same order:

| Map | Purpose |
|-----|---------|
| `activeByTarget` | Currently dispatched orders (keyed by order ID) |
| `activeByID` | Currently running sessions (keyed by session ID) |
| `adoptedTargets` | Sessions from a previous loop run that are still alive |
| `pendingReview` | Completed stages parked for human approval or merge conflict resolution |
| `pendingRetry` | Stages whose dispatch failed (runtime unavailable, at max concurrency) |
| `failedTargets` | Permanently failed orders (blocks scheduler from recreating them) |

When planning spawns, every order is checked against all maps. If its ID appears anywhere, it's skipped — with one exception: orders in `"failing"` status (running their OnFailure pipeline) are exempt from the `failedTargets` check, because recovery stages must be allowed to dispatch.

**What happens if two agents edit the same file?**

They can't. Each agent is in its own worktree — a full copy of the repo on its own branch. Git handles the merge. If there's a conflict, it surfaces during the rebase step of the merge flow. The loop catches the `MergeConflictError` and parks the order for pending review with a reason string — the human resolves the conflict via the web UI. This is a recoverable state, not a permanent failure.

**What about the merge lock? Isn't that a bottleneck?**

Merges are fast — rebase + fast-forward on a local repo is sub-second. The lock prevents interleaving, not parallelism. Agents work in parallel for minutes; the merge serialization point is milliseconds. It's like a checkout line at a grocery store — the shopping takes time, the checkout doesn't.

**What happens when the loop crashes mid-cycle?**

Session adoption. On startup, the loop scans `.noodle/sessions/` and cross-references with `tmux list-sessions`. Any session that's still alive in tmux but not tracked by the loop gets "adopted" into `adoptedTargets`. The loop monitors it until completion, then handles it normally. Sessions that were running but whose tmux session died get their metadata updated to `exited`. Orphaned tmux sessions (named `noodle-*` but not in the session directory) get killed.

The orders file is also crash-safe. The `consumeOrdersNext` function merges incoming orders by ID — if a crash happens after writing `orders.json` but before deleting `orders-next.json`, the next cycle sees the same orders and skips duplicates. No data is lost or double-dispatched.

This means a loop crash loses zero work. Running agents keep running. The new loop picks them up.

---

### What prevents infinite loops or runaway costs?

**Four layers of protection:**

**1. Structural guards in the loop:**

- **Schedule-skip optimization.** The loop only spawns a schedule agent when there are no orders and no sessions running. If the loop restarts with orders already present, it dispatches their next stages directly — no redundant 60-120s scheduling pass.
- **Max retries.** Default 3. Each retry increments an attempt counter. On exhaustion, the stage fails and the order's failure routing kicks in.
- **Failed target tracking.** Once an order is in `failedTargets`, it's never re-dispatched by the loop and the scheduler receives it as context to avoid recreating the same work. Persisted to disk, survives loop restarts.

**2. Failure routing:**

- **OnFailure pipelines.** Orders carry optional `OnFailure` stages — a recovery pipeline that runs when any main stage fails. For example, `[debugging, execute, quality]` to investigate, retry, and re-review. If the OnFailure pipeline itself fails, the order is terminally removed. No nested recovery — one level of fallback.
- **Quality verdict gate.** Before merging, the loop checks `.noodle/quality/<session-id>.json`. A rejection (`accept: false`) triggers the same failure routing as a stage failure. The quality agent's feedback becomes the failure reason, flowing into recovery context.

**3. Health monitoring:**

- **Stuck detection.** If a session is alive but idle for >120s (no new events, no file modifications), it's marked "stuck" with red health. The monitor checks every 5 seconds using dual observation — tmux process liveness AND heartbeat timestamps.
- **Context budget warnings.** When token usage hits 80% of the context window (200K tokens), health goes yellow. This is observability, not a hard kill — but it surfaces to the UI so you can intervene.

**4. Human control:**

- **Autonomy dial.** Set to `approve` and every completed stage parks in `pendingReview` instead of auto-merging. You review in the web UI and approve/reject/request-changes.
- **Control commands.** `pause`, `drain`, `stop-all`, `kill` — all available via `control.ndjson` or the web UI. `drain` finishes active sessions and exits. `stop-all` kills everything immediately.
- **Dynamic concurrency.** `set-max-cooks` can be changed at runtime. Set it to 0 and nothing new spawns.

**Why no global cost budget?**

There's no hard global cost cap — intentionally. Per-session cost is tracked (every `result` event in the canonical NDJSON stream includes `cost_usd`, `tokens_in`, `tokens_out`, accumulated in `meta.json`), but a hard cap creates a worse failure mode than the problem it solves: your system stops working mid-task with no clean recovery. The autonomy dial and manual controls give you the same safety with more nuance. You can see costs in the UI and decide whether to continue.

**What about recovery context on retry?**

When a session fails, the loop runs `recover.CollectRecoveryInfo()` which reads the session's event log. It extracts: files touched, ticket progress, last actions, and the failure reason. This gets passed to the retry session as a "resume context" in the prompt — so the retried agent knows what the previous attempt did and where it failed. It's not starting from scratch.

---

### Why Go?

**The split is deliberate: Go handles mechanics, LLMs handle judgment.**

Go's strengths map perfectly to what the framework needs:

- **Fast startup.** `noodle start` is instant. No JVM warmup, no Python import chain. This matters when the loop runs cycles every few seconds.
- **Single binary.** `go install github.com/poteto/noodle@latest`. No runtime dependencies. The web UI is embedded in the binary via `embed.FS`.
- **Native concurrency.** Goroutines for monitoring tmux sessions, watching files, processing events. Each session spawns a `monitorPane()` goroutine (checks tmux health every 2s) and a `monitorCanonicalEvents()` goroutine (reads NDJSON output every 500ms).
- **Cross-platform.** macOS, Linux, Windows. No bash 4+ features in the codebase — that's a stated constraint.
- **Process management.** `os/exec`, `os.Rename()`, `flock()` — Go is great at the things Noodle needs: spawn processes, watch files, shuffle JSON around.

**What Go doesn't do:** Talk to LLMs. There's no Anthropic SDK, no OpenAI client, no HTTP calls to model APIs. Noodle shells out to CLI tools (`claude`, `codex`) via `exec.Command()`. The output is piped through `noodle stamp` which adds timestamps and parses it into canonical events.

The pipeline looks like:

```
claude -p --output-format stream-json < prompt.txt 2>&1 \
  | noodle stamp --output raw.ndjson --events canonical.ndjson
```

**Why not use the API directly? Isn't shelling out fragile?**

Three reasons:

1. **Provider independence.** Adding a new provider means adding a new `buildXCommand()` function that constructs CLI args. No new HTTP client, no new auth flow, no new streaming parser. The `parse/` package detects the provider from the NDJSON line format and normalizes to canonical events.
2. **The CLI tools handle auth, retries, rate limits, streaming.** That's code I don't have to write or maintain. Claude Code handles its own API key, retry logic, and connection management.
3. **Permission model.** `--permission-mode bypassPermissions` lets the agent run autonomously. `--dangerously-skip-permissions` for full access. These are CLI features, not API features.

**How are prompts composed without an SDK?**

At dispatch time, the dispatcher assembles the prompt from three pieces:

1. **Preamble** — standard Noodle context (state file locations, conventions)
2. **Skill bundle** — the skill's `SKILL.md` (frontmatter stripped) plus all `references/*.md` files concatenated
3. **User prompt** — the task-specific instructions from the stage's prompt field

For Claude: preamble + skill go via `--append-system-prompt`, the task prompt goes via stdin. For Codex: preamble is inlined into the prompt (Codex handles skills natively via its own config).

Task types that declare `domain_skill` in their frontmatter get a second skill bundled — the domain skill is resolved from the registry and appended to the system prompt alongside the methodology skill. This is fully declarative — no hardcoded task key checks in Go.

---

### How is Noodle different from other agent frameworks?

**Three differentiators:**

**1. Files are the API, not tools.**

Most frameworks (LangChain, CrewAI, AutoGen) build around tool-calling APIs or custom protocols. An agent calls `schedule_task(params)` and the framework routes it. In Noodle, an agent writes `orders-next.json` and the loop picks it up. There's no function call boundary — the agent is just writing a file it can already see and understand.

This means agents can inspect and modify their own orchestration. The schedule agent can read `orders.json` and see active orders. A cook can read `mise.json` and see what other agents are doing. Transparency is the default.

**2. LLM-powered scheduling, not rule-based.**

Most frameworks use deterministic routing — "after task A, run task B" or "if condition X, dispatch Y." Noodle's schedule agent reads the full project context (backlog, plans, session history, capacity, task type registry) and makes a judgment call. It can synthesize follow-up tasks it's never seen before, prioritize based on recent failures, and route work to the right model.

The scheduler creates full orders with all stages laid out upfront — `execute → quality → reflect` — and the loop advances them mechanically. The LLM decides *what* work to do and *how to structure the pipeline*; Go handles the stage-by-stage execution. The schedule skill has a situational awareness table: empty orders triggers a full survey; a quality rejection triggers a rescoped retry; all items blocked means schedule reflect or meditate to use the slot productively. This is judgment, not rules.

**3. The brain creates a genuine learning loop.**

After a cook session, the reflect skill captures learnings and routes them to the right place — brain notes for knowledge, skill updates for behavior changes, todos for follow-up work. The key principle is "encode lessons in structure": a lint rule, a frontmatter flag, or a runtime check beats a textual instruction that gets ignored.

After several reflect cycles, meditate does a full vault audit — spawns subagents to find stale notes, extract cross-cutting principles, and update skills. This isn't a gimmick. The skills literally get better over time because agents rewrite them based on accumulated experience.

**Does the learning loop actually work?**

Look at the brain vault. `brain/codebase/` has 15+ files of operational knowledge — things like "tmux shutdown causes a race with the summary buffer" or "worktree prune is equivalent to the cleanup patch." These were written by reflect agents after real sessions. The delegation directory has patterns extracted from multi-agent coordination failures — "prevent subdelegation," "specify verification boundary," "include domain quirks."

The backlog itself tells the story. 40+ completed items ranging from the initial architecture redesign through fixture framework redesign, hot-reload skill resolution, structured logging, web UI. Each one went through the cook -> quality -> reflect cycle.

---

### Notable engineering challenges

Three problems that required the most careful design:

#### Session lifecycle and adoption

The problem: the loop is a long-running process managing agent sessions in tmux. What happens when the loop crashes? When tmux dies? When a session hangs?

The solution required:

- **Dual observation** — tmux process checks (`tmux has-session`) AND heartbeat files. Either can fail independently.
- **Session adoption** — on startup, scan sessions directory, cross-reference with live tmux sessions, adopt anything still running. This means zero work is lost on loop restart.
- **Stuck detection** — "alive but not making progress" is harder than "dead." The monitor tracks last activity time (max of last event timestamp and file modification time) and compares against a threshold.
- **Exit status disambiguation** — a tmux session exiting could mean success (agent finished) or failure (crash). The canonical NDJSON is parsed for an `exited` event with status. If there's no clean exit event, it's treated as a failure.
- **Cancellation vs failure** — if the loop intentionally kills a session (steer command), that's not a failure and shouldn't trigger retry logic. The session state tracks whether the exit was expected.

#### Orders promotion and crash safety

The problem: schedule agents run for 60-120 seconds. During that time, the loop is still cycling — monitoring sessions, processing controls, stamping status. If the schedule agent writes to `orders.json` directly, it races with the loop's own state mutations. And the loop can crash at any point — between writing `orders.json` and deleting `orders-next.json`, between marking a stage active and spawning its session.

The solution: **staging area + idempotent merge**. Schedule writes to `orders-next.json`. The loop merges incoming orders into the existing `orders.json` by order ID — duplicates are skipped, not rejected. Then it deletes `orders-next.json`. If the loop crashes after writing but before deleting, the next cycle sees the same orders and deduplicates them. No data is lost or double-dispatched.

Stage status persistence adds another layer: `spawnCook()` sets `Stage.Status = "active"` and writes to `orders.json` *before* spawning the tmux session. If the loop crashes after persisting but before spawning, the stage is marked active with no running session — session adoption reconciles this on restart.

This pattern cascades: control commands use append-only NDJSON with acknowledgments. Status is written atomically after each cycle. Every state file has a clear ownership model.

#### Skill hot-reload and graceful degradation

The problem: skills are just files on disk. They can appear, disappear, or change at any time. An order might reference a skill that no longer exists. A scheduled task type might become unresolvable.

The solution was a four-layer defense:

1. **fsnotify watcher** with 200ms debounce detects skill directory changes and triggers registry rebuilds.
2. **Belt-and-suspenders check** at dispatch time — if the skill isn't in the registry, force a rebuild before giving up.
3. **Orders validation** — `NormalizeAndValidateOrders` strips stages with unknown task types and logs warnings to `ActionNeeded`.
4. **Bootstrap fallback** — if the schedule skill itself is missing, spawn a bootstrap agent to create one. After 3 failures, give up (`bootstrapExhausted = true`) and keep running without scheduling.

The key insight: a missing skill is degraded operation, not a crash. The loop continues with whatever it can resolve. Same principle applies to the backlog sync script — if it's missing or broken, the brief gets an empty backlog with a warning instead of crashing the cycle.
