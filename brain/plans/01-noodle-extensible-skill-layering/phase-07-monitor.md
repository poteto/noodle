Back to [[plans/01-noodle-extensible-skill-layering/overview]]

# Phase 7 — Monitor

## Goal

Build the monitoring pipeline that derives actual cook state from multiple sources: tmux process liveness, event log metadata, and recent events. The monitor reads ground truth — not what a cook claims its status is, but what's actually happening. This is critical for stuck detection, the scheduling loop's lifecycle management, and the TUI's live display.

The monitor writes materialized state to files in `.noodle/`. Consumers (TUI, CLI, scheduling loop) read these files directly or watch them via fsnotify. There is no pub/sub — the monitor is the single writer of derived state, and the filesystem is the transport.

**Reference codebase:** The previous implementation has a working observe/derive pipeline worth consulting. Read `.noodle/reference-path` for the location, then look at `observe/` (observations, claims, derived state) and `monitor/` (continuous loop with fsnotify). The stuck detection threshold and liveness reconciliation patterns are worth adapting.

## Implementation Notes

This phase will be implemented by Codex. Keep it simple:

- **No over-engineering.** Build exactly what's described. No extra abstractions, no premature generalization, no "just in case" code.
- **No backwards compatibility.** This is a greenfield build — there's nothing to be backwards-compatible with.
- **No extreme concurrency patterns.** Use straightforward goroutines and mutexes where needed. No complex channel orchestration or worker pools unless the phase explicitly calls for them.
- **Tests should increase confidence, not coverage.** Test the critical flows that would break silently if wrong. Skip testing implementation details, trivial getters, or obvious wiring. Ask: "does this test help us iterate faster?"

- **End-of-phase Claude review (required).** After implementing this phase, write the review prompt to a temp file and run a non-interactive Claude review with tools disabled and bypassed permissions, for example: `prompt_file="$(mktemp)"; printf '%s\n' "Review the changes for this phase. Report risks, regressions, and missing tests." > "$prompt_file"; claude -p --dangerously-skip-permissions --tools "" -- "$(cat "$prompt_file")" | tee .noodle/reviews/<phase-id>-review.log; rm -f "$prompt_file"`.
- **Observe Claude liveness in global logs while it runs.** Check the global `~/.claude` directory (project session `.jsonl` logs under `~/.claude/projects/`) and watch the active session log; as long as new log entries are being written, Claude is still working.
- **Stall criteria + completion gate.** Treat the review as stalled only when the active global `~/.claude/projects/` session log stops changing for more than 180s *and* the Claude process is still alive. Do not mark the phase complete until the Claude command exits and the global log contains the final assistant output, with blocking findings addressed.
- **Overview + phase completeness check (required).** Before closing the phase, review your changes against both the plan overview and this phase document. Verify every detail in Goal, Changes, Data Structures, and Verification is satisfied; track and resolve any mismatch before considering the phase done.
- **Worktree discipline (required).** Execute each phase in its own dedicated git worktree.
- **Commit cadence (required).** Commit as much as possible during the phase: at least one commit per phase, and preferably multiple commits for distinct logical changes.
- **Main-branch finalize step (required).** After all verification and review checks pass, merge the phase worktree to `main` and make sure the final verified state is committed on `main`.

## Changes

- **`monitor/`** — New package. Observation collection (tmux PID checks, event log file stats), claims parsing (latest events per cook), state derivation (combine observations + claims), stuck detection, state file writing, ticket materialization, continuous monitoring loop.
- Integration with spawner (Phase 6) for tmux pane liveness checks.
- Integration with event log (Phase 5) for reading session events.

## How It Works

The monitor runs as a continuous loop inside the scheduling loop process. On each tick:

1. **Observe** — Check tmux pane liveness for each active cook. Check event log file mtime/size.
2. **Read claims** — Parse recent events from each cook's `events.ndjson` to find latest action, cost, status.
3. **Derive state** — Combine observations + claims into per-cook state: actual liveness (from tmux), last activity timestamp, stuck flag, total cost, current action, health status.
4. **Write state files** — Update `.noodle/sessions/{cook-id}/meta.json` for each cook. Update `.noodle/tickets.json` by reading ticket events from all active session logs.
5. **Detect stuck** — Flag cooks with no activity past the threshold. The scheduling loop reads this from the meta file and can kill + recover.

The loop triggers on fsnotify events on the `sessions/` directory (new events written) with a fallback ticker for liveness checks that fsnotify can't detect (tmux pane death).

## Data Structures

- `Observer` — Collects raw observations: tmux pane PID liveness, event log file mtime/size
- `ClaimsReader` — Parses recent events from a session's `events.ndjson` to find latest action, cost, status
- `SessionMeta` — Per-cook state written to `.noodle/sessions/{cook-id}/meta.json`: status (running/stuck/exited/failed), provider, model, total cost, duration, last activity timestamp, current action summary, health (green/yellow/red), context window usage percentage, retry count
- `Monitor` — Continuous loop. Uses fsnotify on sessions directory + fallback ticker. Debounces writes. Writes `SessionMeta` files and `tickets.json`.
- `StuckThreshold` — Configurable duration (default 120s) after which a cook with no activity is flagged as stuck

### Stuck Detection

A cook is "stuck" when:
1. Its tmux pane is alive (process running), AND
2. No new events or log output for longer than the stuck threshold

This distinguishes stuck (alive but unresponsive) from crashed (tmux pane dead). The scheduling loop can kill stuck cooks and trigger recovery.

### Health Derivation

The monitor derives a health status for each cook, written to `meta.json`:
- **green** — running, making progress, within budget
- **yellow** — running but concerning: >80% context window usage, or idle for more than half the stuck threshold (default: >60s with 120s threshold)
- **red** — failed, retrying, or stuck (no activity past threshold)

The TUI reads health from `meta.json` and displays it as a colored dot.

## Verification

### Static
- `go test ./monitor/...` — Unit tests:
  - Stuck detection: cook with no activity past threshold flagged
  - Liveness: dead tmux pane detected correctly
  - Claims parsing: latest action/cost extracted from event log
  - State derivation: observations + claims combined correctly
  - Health derivation: green/yellow/red thresholds applied correctly
  - SessionMeta written to correct path with expected fields
  - Ticket materialization: ticket events from multiple sessions merged into `tickets.json`

### Runtime
- Monitor running with active cook: `meta.json` shows correct status, cost, last action
- Kill a cook's tmux pane: monitor detects death within one poll cycle, updates `meta.json`
- Cook that stops producing output: flagged as stuck after threshold, health turns red
- Two cooks claiming tickets: `tickets.json` reflects both active tickets
