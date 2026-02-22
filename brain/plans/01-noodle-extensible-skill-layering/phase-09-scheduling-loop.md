Back to [[plans/01-noodle-extensible-skill-layering/overview]]

# Phase 9 — Scheduling Loop + Recovery

## Goal

Implement the core runtime loop: gather mise → sous chef writes queue → loop watches queue → enforce constraints → spawn cooks → review → mark done → recover failures.

The scheduling loop is intentionally simple. It does not score, prioritize, or make scheduling decisions. It watches `.noodle/queue.json` via fsnotify and mechanically spawns cooks from it, enforcing only hard constraints (concurrency limits, exclusivity, tickets). The queue file is the interface between the sous chef and the loop — the sous chef writes it, the loop reads it.

When a cook completes, the loop spawns a review (taster) by default. The taster reviews the work and outputs accept/reject. On accept, the loop ensures the verified worktree is merged to `main` (preferred: cook merges via `commit` skill; fallback: loop merges) and then calls the adapter's `done` script to mark the item complete. On reject, the loop respawns a cook with the taster's feedback.

When a cook fails (crashes, stuck, killed), the loop handles recovery: it analyzes what went wrong, builds resume context, and respawns with a retry suffix — up to a configurable max (default 3).

**Reference codebase:** The previous implementation has recovery tracking worth consulting. Read `.noodle/reference-path` for the location, then look at `recover/`. The `RecoveryInfo`, `BuildResumeContext`, and `RecoveryChainLength` patterns are worth adapting.

## Implementation Notes

This phase will be implemented by Codex. Keep it simple:

- **No over-engineering.** Build exactly what's described. No extra abstractions, no premature generalization, no "just in case" code.
- **No backwards compatibility.** This is a greenfield build — there's nothing to be backwards-compatible with.
- **No extreme concurrency patterns.** Use straightforward goroutines and mutexes where needed. No complex channel orchestration or worker pools unless the phase explicitly calls for them.
- **Tests should increase confidence, not coverage.** Test the critical flows that would break silently if wrong. Skip testing implementation details, trivial getters, or obvious wiring. Ask: "does this test help us iterate faster?"
- **Prefer markdown fixture tests for parser/protocol flows.** Keep input and expected output in the same anonymized `*.fixture.md` file under `testdata/`, and use this pattern wherever practical.


- **End-of-phase Claude review (required).** After implementing this phase, write the review prompt to a temp file and run a non-interactive Claude review with tools disabled and bypassed permissions, for example: `prompt_file="$(mktemp)"; printf '%s\n' "Review the changes for this phase. Report risks, regressions, and missing tests." > "$prompt_file"; claude -p --dangerously-skip-permissions --tools "" -- "$(cat "$prompt_file")" | tee .noodle/reviews/<phase-id>-review.log; rm -f "$prompt_file"`.
- **Observe Claude liveness in global logs while it runs.** Check the global `~/.claude` directory (project session `.jsonl` logs under `~/.claude/projects/`) and watch the active session log; as long as new log entries are being written, Claude is still working.
- **Stall criteria + completion gate.** Treat the review as stalled only when the active global `~/.claude/projects/` session log stops changing for more than 180s *and* the Claude process is still alive. Do not mark the phase complete until the Claude command exits and the global log contains the final assistant output, with blocking findings addressed.
- **Overview + phase completeness check (required).** Before closing the phase, review your changes against both the plan overview and this phase document. Verify every detail in Goal, Changes, Data Structures, and Verification is satisfied; track and resolve any mismatch before considering the phase done.
- **Worktree discipline (required).** Execute each phase in its own dedicated git worktree.
- **Commit cadence (required).** Commit as much as possible during the phase: at least one commit per phase, and preferably multiple commits for distinct logical changes.
- **Main-branch finalize step (required).** After all verification and review checks pass, merge the phase worktree to `main` and make sure the final verified state is committed on `main`.

## Changes

- **`loop/`** — New package. The scheduling loop: watches `.noodle/queue.json` via fsnotify, processes queue items, enforces constraints, manages cook lifecycle, tasting, completion, recovery. Writes `.noodle/mise.json` for the sous chef to read.
- **`recover/`** — New package. Recovery analysis (exit reason, last action, files changed), resume context building, retry chain tracking.
- **`cmd_start.go`** — `noodle start` CLI command. Starts the continuous scheduling loop. Runs until interrupted or queue is empty (configurable).
- **`command_catalog.go`** — Register `start` command.

## Startup Reconciliation

When `noodle start` boots, it reconciles leftover state from any previous run before entering the main loop:

1. **Scan tmux** — Find all tmux sessions matching Noodle's naming pattern. For each:
   - Has matching `meta.json` with `status: running` → **adopt** (resume monitoring, keep tickets)
   - Has no matching `meta.json` or `meta.json` shows `exited` → **kill** orphaned tmux session
2. **Scan state files** — For each session in `.noodle/sessions/`:
   - `meta.json` shows `running` but no matching tmux session → mark `exited`, remove tickets, add to recovery candidates
3. **Clean tickets** — Regenerate `.noodle/tickets.json` from adopted sessions only
4. **Validate queue** — Remove items from `.noodle/queue.json` that are already being worked on by adopted cooks
5. **Proceed** — Enter the main loop with adopted cooks counted against concurrency limits

This makes `noodle start` always safe to run — no manual cleanup after crashes.

## Loop States + Control File

The loop has three states, controlled by chef intervention commands (Phase 10):

- **Running** — Normal operation. Spawns cooks from the queue, enforces constraints.
- **Paused** — No new cooks spawn. Active cooks continue. Queue accumulates. Resumable.
- **Draining** — No new cooks spawn. Active cooks run to completion. Loop exits when all finish.

State transitions: `running ↔ paused`, `running → draining`, `paused → draining`. The TUI shows the current state in the header.

### Control channel

The TUI and external tools communicate with the running loop via `.noodle/control.ndjson` — a file-based protocol consistent with the rest of the architecture (no IPC, no sockets). The TUI appends commands directly; the loop watches the file via fsnotify and processes new lines.

```ndjson
{"id": "a1b2c3", "action": "pause", "at": "2026-02-22T18:30:00Z"}
{"id": "d4e5f6", "action": "resume", "at": "2026-02-22T18:35:00Z"}
{"id": "g7h8i9", "action": "drain", "at": "2026-02-22T18:36:00Z"}
{"id": "j0k1l2", "action": "skip", "item": "43", "at": "2026-02-22T18:37:00Z"}
{"id": "m3n4o5", "action": "kill", "name": "fix-auth-bug", "at": "2026-02-22T18:38:00Z"}
{"id": "p6q7r8", "action": "steer", "target": "fix-auth-bug", "prompt": "focus on tests", "at": "..."}
{"id": "s9t0u1", "action": "steer", "target": "sous-chef", "prompt": "bring forward security tasks", "at": "..."}
```

Every command has a unique `id` (short random string, generated by the writer). The file is append-only NDJSON. The loop tracks its read offset, processes new lines on fsnotify, and advances the offset. No commands are dropped — even if multiple commands arrive between loop cycles, they're all queued and processed in order. The loop truncates the file after processing all pending lines (resetting the offset to 0).

**Locking:** All readers and writers use Go's `syscall.Flock` (which maps to `flock(2)` on Linux and `fcntl(F_SETLK)` on macOS) on `.noodle/control.lock`. This is implemented in Go — not the shell `flock` command — so it works on both platforms. Writers acquire the lock, append a single line + `\n`, fsync, and release. The loop also acquires the lock during its read+truncate cycle: it reads all new lines, truncates the file, resets its offset, then releases the lock. This prevents the race where a writer appends between read and truncate — writers block until the truncate completes.

**Acknowledgment:** After the loop processes a command, it appends an ack to `.noodle/control-ack.ndjson`: `{"id": "a1b2c3", "action": "pause", "status": "ok"}` (or `"status": "error", "message": "..."`). The writer polls `control-ack.ndjson` for a matching `id` (short timeout, default 5s). If the ack arrives, it confirms success. If it times out, it warns that the loop may not be running. **Deduplication:** The loop keeps an in-memory set of processed command IDs (reconstructible from `control-ack.ndjson` on restart). Before executing a command, it checks the set — if the ID was already processed, it skips execution and re-appends the original ack. This makes retries safe for all command types, including steer and kill (which have side effects that aren't naturally idempotent).

**Not a stable API.** The control channel is internal plumbing for the TUI and power-user scripts. The format may change between versions. Third-party tools that write to it should expect to adapt.

## Cook Lifecycle

```
queue item → spawn cook → cook completes → review? → spawn taster → taster accepts?
                                                                      ├── yes → merge worktree to main → adapter done script → next item
                                                                      └── no  → respawn cook with feedback
                                           (review: false)  → merge worktree to main → adapter done script → next item
                                          → cook fails    → recovery (up to max retries)
                                          → cook stuck    → kill → recovery
```

## Review

Review runs after every cook by default. The sous chef can override this per queue item by annotating `review: false` for trivial items. The taster is spawned as a cook session with the taster skill, receiving the original cook's diff, test output, and verification results.

The taster outputs a structured verdict: `{"accept": bool, "feedback": "..."}`. On accept, the loop gates completion on a successful merge to `main`, then calls the adapter's `done` script. On reject, the loop spawns a new cook with the taster's feedback prepended to the original prompt.

## Concurrency + Worktree Isolation

Every cook runs in its own git worktree, including single-cook runs (`concurrency.max_cooks = 1`). This keeps execution uniform and prevents root-checkout mutations. The loop creates the worktree automatically before spawning.

**Merge policy (MVP):**
- Direct merge to `main` only. Opening PRs is out of scope for this plan.
- Preferred path: the cook uses the `commit` skill to commit logical changes and merge its own worktree after verification passes.
- Fallback path: if the cook exits without merging but verification/taster verdict is acceptable, the loop merges the cook worktree into `main` and records that merge event before adapter `done`.
- If merge fails, the backlog item is not marked done; the loop emits actionable diagnostics and enters recovery.

`.noodle/mise.json` includes a `Resources` snapshot (max concurrency, active count, available slots) so the sous chef can see how many slots are open when writing the queue. The sous chef should fill available slots with independent work items — it should not queue items that conflict with each other (e.g., modifying the same files) for parallel execution. Tickets provide the safety net: if two cooks accidentally target overlapping resources, the ticket system prevents conflicts at runtime.

## File-Based Queue + Mise

The queue and mise are plain JSON files in `.noodle/`, readable by both humans and agents:

- **`.noodle/mise.json`** — The gathered state brief. Written by the Go loop after each cycle (adapter sync, active cooks, tickets, resources, recent history). The sous chef reads this file to make scheduling decisions.
- **`.noodle/queue.json`** — The prioritized queue. Written by the sous chef. The loop watches this file via fsnotify and processes items as they appear.

This design has three advantages:
1. **Observable** — The human can inspect both files at any time to see what Noodle knows and what it plans to do.
2. **Simple sous chef interface** — The sous chef just reads one JSON file and writes another. No special binary commands needed for scheduling. The sous-chef skill teaches the agent the schemas and how to reason about priorities.
3. **Cross-platform** — JSON files work everywhere. No IPC, no sockets, no platform-specific mechanisms.

**Atomic writes:** All JSON state files (`.noodle/mise.json`, `.noodle/queue.json`, `.noodle/tickets.json`, `.noodle/sessions/*/meta.json`) are written atomically: write to a temp file in the same directory, then `os.Rename` to the target path. This prevents partial reads under fsnotify. The sous-chef skill teaches the agent the same pattern for writing `queue.json`.

The loop regenerates `.noodle/mise.json` after each cook completion, after recovery events, and on a configurable interval. When the sous chef writes `.noodle/queue.json`, fsnotify triggers the loop to re-read and process new items.

**Fallback:** If the sous chef fails to write a queue, the loop uses the last valid `.noodle/queue.json` on disk. If no queue file exists, the loop falls back to FIFO ordering of backlog items from the mise.

## Data Structures

- `Loop` — Holds config, spawner, mise builder, event writer, monitor, adapter runner. Methods: `Run(ctx)`, `Cycle(ctx)` (one scheduling round)
- `Queue` — Ordered list of `QueueItem` from sous chef output
- `QueueItem` — See Phase 8 for full schema. Key fields: backlog item ID, provider, model, skill, review flag, rationale.
- `Constraint` — Hard rules: max concurrent cooks, exclusivity (one cook per backlog item), ticket check, worktree isolation requirement
- `TasterVerdict` — Accept/reject + feedback string
- `RecoveryInfo` — Exit reason, last action, files changed, completed/failed sub-agents, pending tasks
- `ResumeContext` — Human-readable summary built from `RecoveryInfo` that gets injected into the respawned cook's prompt

Recovery chain: failed cook `task-1` respawns as `task-1-recover-1`, which if it fails becomes `task-1-recover-2`, up to the max. `RecoveryChainLength` walks the chain to enforce the limit.

## Runtime Self-Healing

The scheduling loop applies the same self-healing principle as startup. When infrastructure fails at runtime — an adapter script errors, a skill can't be resolved, a worktree can't be created — the loop classifies the failure:

- **Repairable** (adapter script missing, skill not found, config drift): pause the affected queue item, spawn a repair cook with the `noodle` meta-skill and the diagnostic context. After repair, re-validate and resume.
- **Fatal** (spawner broken, provider unreachable, filesystem full): log a detailed, actionable error and halt the loop. The error message includes what failed, why, and how to fix it.

The loop never silently skips broken items or produces cryptic errors. It either fixes the problem or tells the user exactly how to.

## Oops

Distinct from repair (which fixes Noodle's own infrastructure), oops addresses problems in the user's project — broken tests, failing builds, workflow issues. The scheduling loop spawns an oops session when a cook's work causes systemic failures (e.g., tests that were passing before a cook now fail across the codebase, not just in the cook's changes). The skill used is configurable via `phases.oops` in config. The user won't have Noodle's source code in their project, so the oops skill must be self-contained — it teaches the agent how to diagnose and fix the user's project, not Noodle itself.

## Verification

### Static
- `go test ./loop/...` — Unit tests:
  - Constraint enforcement: respects max concurrency, prevents duplicate work
  - Tickets: skips items with active tickets
  - Queue processing: items spawned in order
  - Worktree policy: every spawned cook gets a dedicated worktree (including max concurrency = 1)
  - Fallback: uses last `.noodle/queue.json` on disk when sous chef fails
  - Empty queue: loop waits for next fsnotify event on queue file
  - Queue file write triggers loop to re-read and process
  - Accept path: merge-to-`main` succeeds before adapter `done` runs
  - Fallback merge path: loop merges verified unmerged cook worktree before `done`
  - Merge failure: item remains open and recovery/actionable error path triggers
  - Taster accept → merge gate passes → adapter `done` script called
  - Taster reject → cook respawned with feedback
  - `review: false` annotation → taster skipped, merge gate still enforced, adapter `done` called
  - Reconciliation: orphaned tmux session killed, stale meta.json marked exited
  - Reconciliation: running tmux session adopted, counted against concurrency
  - Pause: no new cooks spawn while paused, active cooks continue
  - Drain: no new cooks spawn, loop exits when active cooks finish
- `go test ./recover/...` — Unit tests:
  - Resume context includes exit reason, last action, files changed
  - Recovery chain length correctly counted
  - Max retries enforced (returns error after limit)

### Runtime
- `noodle start` with a single todo item: writes `.noodle/mise.json`, sous chef writes `.noodle/queue.json`, loop spawns a cook in a worktree, reviews, merges to `main`, marks done
- `noodle start` with max concurrency = 1: never spawns more than one cook at a time
- `noodle start` with max concurrency = 1: spawned cook still runs in a dedicated worktree
- `noodle start` with max concurrency = 2 and two items: spawns two cooks in separate worktrees concurrently
- Cook completes → taster spawned → accepts → adapter `done` script runs
- Cook completes → taster accepts → cook uses `commit` skill to merge to `main` → adapter `done` runs
- Cook completes → taster accepts but cook did not merge → loop merges worktree to `main` → adapter `done` runs
- Cook completes → taster accepts but merge fails → item not marked done, actionable error logged, recovery path triggered
- Cook completes → taster rejects → new cook spawned with feedback
- Failed cook: loop builds resume context, respawns with `-recover-1` suffix
- Cook that fails 3 times: loop stops retrying, marks item as failed
- Missing adapter script at runtime: repair cook spawned, script created, queue item resumes
- Unreachable provider at runtime: loop halts with actionable error (not silent failure)
- `noodle start` after crash: adopts surviving tmux sessions, recovers dead ones, cleans orphaned tickets
- Pause → resume via control channel: loop stops and restarts spawning correctly
- Drain via control channel: loop finishes active cooks then exits cleanly
- Control command ordering: three commands appended rapidly are processed in order
- Control ack: command id in ack matches command id written
- Concurrent writers: two writers appending simultaneously produce valid NDJSON (no interleaved lines)
- Control truncation: file is truncated after processing, offset resets correctly
