Back to [[plans/90-interactive-sessions/overview]]

# Phase 4 — Snapshot Integration

## Goal

Include interactive sessions in the snapshot so the UI can discover and display them. Sessions carry their `Kind` so the UI can distinguish them. Route data through `LoopState` — the snapshot builder must not reach into loop internals.

## Changes

**`loop/types.go`** (or `loop/state_snapshot.go`) — Extend `LoopState` to include interactive session metadata. Add a slice or map of interactive session summaries (ID, name, status, kind, started-at) that the loop populates when building state.

**`internal/snapshot/types.go`** — Add `Kind string` field to the `Session` snapshot type. Add `CorrelationID string` field — this is the control command ID that spawned the session, so the UI can deterministically match a `chat` ack to the spawned session (not rely on "next interactive session seen in snapshot"). Both order and interactive sessions appear in the existing `Sessions`/`Active`/`Recent` slices — no separate `InteractiveSessions` field.

**`internal/snapshot/build.go`** (or equivalent snapshot builder) — When building snapshots, read interactive session data from `LoopState` (not from `l.chats` directly). Set `Kind: "interactive"` on these sessions. Order sessions get `Kind: "order"`.

**Critical: snapshot builder reads from `LoopState`, not loop internals.** The loop populates `LoopState` with interactive session data on each cycle, and the snapshot builder consumes it — same boundary as order sessions.

## Data Structures

- `Session.Kind string` — `"order"` or `"interactive"`
- `Session.CorrelationID string` — control command ID that spawned this session
- `LoopState` extended with interactive session summary data (including correlation ID)
- Snapshot's `Sessions`, `Active`, `Recent` slices include both kinds

## Routing

- **Provider:** `codex`
- **Model:** `gpt-5.4`

## Verification

### Static
- `go build ./...`
- `go vet ./...`
- Existing snapshot tests pass with `Kind` field added

### Runtime
- Test: spawn interactive session, build LoopState, verify interactive session data present
- Test: fetch snapshot, verify session appears with `kind: "interactive"`
- Test: snapshot includes interactive session events in `events_by_session`
- Test: interactive session completion removes it from `Active`, adds to `Recent`
- Test: order sessions have `kind: "order"` in snapshot
- Test: interactive session has `correlation_id` matching the control command ID that spawned it
- Test: two concurrent chat spawns — each session has a distinct `correlation_id`, UI can match deterministically
