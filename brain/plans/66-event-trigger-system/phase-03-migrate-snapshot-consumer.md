Back to [[plans/66-event-trigger-system/overview]]

# Phase 3 — Migrate Snapshot Consumer

## Goal

Update the sole consumer of `queue-events.ndjson` — `internal/snapshot` — to read `loop-events.ndjson` and the new `LoopEvent` format. After this phase, `queue-events.ndjson` is fully dead.

## Changes

- **`internal/snapshot/snapshot.go`** — `readQueueEvents()` (~line 431): change file path from `queue-events.ndjson` to `loop-events.ndjson`. Update the anonymous unmarshal struct to match `LoopEvent` shape (`type`, `payload`, `seq`, `at`). Update the switch cases to map new event types (`registry.rebuilt`, `bootstrap.completed`, `bootstrap.exhausted`, `order.dropped`) to `FeedEvent` records.
- **`internal/snapshot/snapshot_test.go`** — update any test fixtures referencing `queue-events.ndjson` format.
- **Testdata fixtures** — update or create NDJSON fixture files with new format if snapshot tests use file-based fixtures.

## Data Structures

- The snapshot reader defines its own anonymous struct to unmarshal `LoopEvent` lines — it does NOT import from `event/` package (same pattern as the current code, which uses an anonymous struct). File format is the API contract.

## Routing

Provider: `codex`, Model: `gpt-5.3-codex`

## Verification

```bash
go test ./internal/snapshot/... && go vet ./...
```

- `readQueueEvents` reads `loop-events.ndjson` and produces correct `FeedEvent` records for each event type
- No references to `queue-events.ndjson` remain anywhere in non-archived code
- Snapshot tests pass with new format fixtures
