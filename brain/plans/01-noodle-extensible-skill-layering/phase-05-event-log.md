Back to [[plans/01-noodle-extensible-skill-layering/overview]]

# Phase 5 — Event Log + Tickets

## Goal

Build the append-only event log and the ticket coordination protocol. Events are the raw audit trail for every cook session — tool calls, thinking, costs, ticket claims. The monitor (Phase 7) reads event logs and materializes derived state into files. There is no pub/sub bus — consumers read state files and tail event logs via fsnotify.

Tickets prevent duplicate work when multiple cooks run concurrently — a cook claims a ticket when it starts work on a target, and the scheduling loop skips items with active tickets.

**Reference codebase:** The previous implementation has a working event system and coordination protocol worth consulting. Read `.noodle/reference-path` for the location, then look at `event/`. The wall claim events (`wall.go`) map directly to the ticket protocol — rename and adapt.

## Implementation Notes

This phase will be implemented by Codex. Keep it simple:

- **No over-engineering.** Build exactly what's described. No extra abstractions, no premature generalization, no "just in case" code.
- **No backwards compatibility.** This is a greenfield build — there's nothing to be backwards-compatible with.
- **No extreme concurrency patterns.** Use straightforward goroutines and mutexes where needed. No complex channel orchestration or worker pools unless the phase explicitly calls for them.
- **Tests should increase confidence, not coverage.** Test the critical flows that would break silently if wrong. Skip testing implementation details, trivial getters, or obvious wiring. Ask: "does this test help us iterate faster?"
- **Prefer markdown fixture tests for parser/protocol flows.** Keep input and expected output in the same anonymized `*.fixture.md` file under `testdata/`, and use this pattern wherever practical.


- **End-of-phase Claude review (required).** After implementing this phase, run a non-interactive Claude review of your changes and capture NDJSON output, for example: `claude -p --output-format stream-json --verbose --include-partial-messages "Review the changes for this phase. Report risks, regressions, and missing tests." | tee .noodle/reviews/<phase-id>-review.ndjson`.
- **Observe NDJSON liveness while it runs.** Watch the review log (`tail -f .noodle/reviews/<phase-id>-review.ndjson`). Any appended NDJSON line (`stream_event`, `assistant`, `user`, `system`, `result`) means Claude is still working.
- **Stall criteria + completion gate.** Treat the review as stalled only when no new NDJSON lines appear for more than 180s *and* the Claude process is still alive. Do not mark the phase complete until a terminal `result` event is present in the review log and blocking findings are addressed.
- **Overview + phase completeness check (required).** Before closing the phase, review your changes against both the plan overview and this phase document. Verify every detail in Goal, Changes, Data Structures, and Verification is satisfied; track and resolve any mismatch before considering the phase done.
- **Worktree discipline (required).** Execute each phase in its own dedicated git worktree.
- **Commit cadence (required).** Commit as much as possible during the phase: at least one commit per phase, and preferably multiple commits for distinct logical changes.
- **Main-branch finalize step (required).** After all verification and review checks pass, merge the phase worktree to `main` and make sure the final verified state is committed on `main`.

## Changes

- **`event/`** — New package. Event schema, event writer (append to per-session NDJSON file), event reader (tail/query a session's log file), ticket state materializer.
- **`command_catalog.go`** — No separate `events` CLI command. Event log access is via the TUI trace view (Phase 13).

## `.noodle/` Directory Layout

All runtime state lives in the project's `.noodle/` directory. This directory can be gitignored — it contains only runtime state, not configuration.

```
.noodle/
├── mise.json                    # gathered state brief (Phase 8)
├── queue.json                   # prioritized queue from sous chef (Phase 9)
├── tickets.json                 # materialized active tickets
├── control.ndjson               # CLI → loop command queue (Phase 9)
├── adapters/                    # default adapter scripts (recreatable via bootstrap)
└── sessions/
    ├── fix-auth-bug/
    │   ├── meta.json            # status, provider, model, cost, duration, health
    │   └── events.ndjson        # append-only event log
    └── add-user-tests/
        ├── meta.json
        └── events.ndjson
```

## Data Structures

- `Event` — Type + payload + timestamp + session ID
- `EventType` — Enum:
  - Core: `spawned`, `action`, `cost`, `state_change`, `exited`
  - Cook lifecycle: `cook_started`, `cook_completed`
  - Tickets: `ticket_claim`, `ticket_progress`, `ticket_done`, `ticket_blocked`, `ticket_release`
- `EventWriter` — Appends events to `.noodle/sessions/{cook-id}/events.ndjson`. Append-only, no locking needed (one writer per session).
- `EventReader` — Reads/tails a session's event log. Supports filtering by event type and time range.
- `Ticket` — Target (backlog item ID or file path), claimant cook ID, optional file list. Tickets become stale after a configurable timeout (default 30min) with no `ticket_progress` event.
- `TicketMaterializer` — Reads ticket events from all active session logs, writes materialized state to `.noodle/tickets.json`. Called by the monitor (Phase 7).

### Ticket Protocol

Tickets are coordination events that prevent multiple cooks from doing the same work simultaneously. A cook writes ticket events to its own `events.ndjson`. The monitor reads ticket events from all active session logs and materializes the current state into `.noodle/tickets.json`. The scheduling loop reads `tickets.json` to enforce constraints.

#### Ticket Event Types

| Event | Payload | When |
|-------|---------|------|
| `ticket_claim` | `{target, target_type, files}` | Cook starts work on a target |
| `ticket_progress` | `{target, summary}` | Cook proves liveness on claimed target (periodic, every ~60s) |
| `ticket_done` | `{target, outcome}` | Cook finishes work on target (success or failure) |
| `ticket_blocked` | `{target, blocked_by, reason}` | Cook can't proceed — waiting on another cook's work |
| `ticket_release` | `{target, reason}` | Cook voluntarily releases a claim (e.g., reprioritized, wrong approach) |

#### Target Types

A ticket's `target` identifies what work is being claimed. The `target_type` field distinguishes:

- **`backlog_item`** — A backlog item ID (e.g., `"42"`). The scheduling loop's primary constraint — it won't assign the same backlog item to two cooks.
- **`file`** — A file path (e.g., `"src/auth/token.ts"`). Advisory — the scheduling loop uses file claims to detect potential conflicts when the sous chef queues items that might touch overlapping files.
- **`plan_phase`** — A plan phase (e.g., `"03/phase-2"`). Prevents two cooks from working on the same plan phase.

#### Ticket Lifecycle

```
ticket_claim → ticket_progress (periodic) → ticket_done
                                           → ticket_release (voluntary)
              → ticket_blocked (waiting)   → ticket_progress (unblocked)
                                           → ticket_release (gave up)
```

#### Materialized State (`tickets.json`)

The monitor writes `.noodle/tickets.json` as an atomic JSON file containing all active tickets:

```json
[
  {
    "target": "42",
    "target_type": "backlog_item",
    "cook_id": "fix-auth-bug",
    "files": ["src/auth/token.ts", "src/auth/middleware.ts"],
    "claimed_at": "2026-02-21T18:30:00Z",
    "last_progress": "2026-02-21T18:35:00Z",
    "status": "active"
  }
]
```

Status values: `active`, `blocked`, `stale`. The monitor derives status from the event stream — `active` if recent progress, `blocked` if last event was `ticket_blocked`, `stale` if no progress within the timeout.

#### Scheduling Loop Constraints

The scheduling loop reads `tickets.json` and enforces:

1. **Backlog exclusivity** — Won't spawn a cook for a backlog item that has an active or blocked ticket.
2. **File conflict detection** — Warns (but doesn't block) when a queued item's expected files overlap with an active ticket's file list. The sous chef can use this to avoid parallel conflicts.
3. **Stale ticket cleanup** — Treats stale tickets as abandoned. The scheduling loop can reassign the work.

#### Relationship to Stuck Detection (Phase 7)

The stuck threshold (default 120s) and ticket stale timeout (default 30m) serve different purposes. Stuck detection flags a cook as unresponsive — the scheduling loop can kill it and trigger recovery. When a cook is killed, the monitor removes its tickets from `.noodle/tickets.json`. The 30m stale timeout is a safety net for edge cases where a cook exits without appending `ticket_done`.

## Verification

### Static
- `go test ./event/...` — Event write/read round-trip, event filtering, ticket materialization
- Ticket staleness detection tests (ticket with no progress expires)
- Active tickets query returns only non-stale, non-done tickets

### Runtime
- Append events to a session log, read them back, verify round-trip
- Two cooks claiming the same target: `.noodle/tickets.json` shows the active ticket, second cook sees it
- Kill a cook: monitor removes its tickets from `.noodle/tickets.json`
