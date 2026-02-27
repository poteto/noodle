Back to [[archive/plans/66-event-trigger-system/overview]]

# Phase 1 — Event Bus Types and NDJSON Writer

## Goal

Define the full lifecycle event vocabulary and build the append-only NDJSON writer. This is the foundation — data structures before logic.

## Changes

- **`event/loop_event.go`** (new) — `LoopEventType` enum, `LoopEvent` struct, `EventWriter` with mutex-guarded append and sequence counter.
- **`event/loop_event_test.go`** (new) — unit tests for writer.

## Data Structures

- `LoopEventType` — string enum: `stage.completed`, `stage.failed`, `order.completed`, `order.failed`, `order.dropped`, `order.requeued`, `worktree.merged`, `merge.conflict`, `quality.written`, `schedule.completed`, `registry.rebuilt`, `sync.degraded`, `bootstrap.completed`, `bootstrap.exhausted`
- `LoopEvent` — `Seq uint64`, `Type LoopEventType`, `At time.Time`, `Payload json.RawMessage`
- `EventWriter` — mutex, file handle, monotonic sequence counter, `Emit(eventType, payload)` method. Append JSON line + newline. On write failure: log warning, continue (best-effort). Truncation method: keep last N lines (reuse the truncation pattern from `order_audit.go`, which will be deleted in phase 2).

The full event vocabulary is declared here even though emission happens in later phases. This follows foundational-thinking: define the data structures before the logic that produces them.

## Routing

Provider: `codex`, Model: `gpt-5.3-codex`

## Verification

```bash
go test ./event/... && go vet ./event/...
```

- `EventWriter.Emit` appends valid JSON lines with monotonic sequences
- Concurrent `Emit` calls produce no interleaving (mutex test)
- Truncation keeps last N lines
- Write failure logs warning but does not panic or return error
