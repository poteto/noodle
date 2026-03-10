Back to [[archive/plans/66-event-trigger-system/overview]]

# Phase 2 — Delete Audit Infrastructure

## Goal

Remove the `QueueAuditEvent` system entirely. Redirect the four existing write sites to use the new `EventWriter` from phase 1. No dual paths — subtract the old system and wire in the new one in the same phase.

## Changes

- **`loop/order_audit.go`** — delete `QueueAuditEvent`, `appendQueueEvent`, `truncateQueueEvents`. Keep `RegistryDiff`, `diffRegistryKeys`, `auditOrders` (they stay, but `auditOrders` switches to `EventWriter`).
- **`loop/loop.go`** (~line 150) — registry rebuild: replace `appendQueueEvent(eventsPath, ...)` with `EventWriter.Emit("registry.rebuilt", ...)`.
- **`loop/schedule.go`** (~line 127) — bootstrap exhausted: replace `appendQueueEvent` with `EventWriter.Emit("bootstrap.exhausted", ...)`.
- **`loop/cook_completion.go`** (~line 108) — bootstrap complete: replace `appendQueueEvent` with `EventWriter.Emit("bootstrap.completed", ...)`.

The `EventWriter` must be wired into the `Loop` struct (field or dependency). The runtime dir changes from `queue-events.ndjson` to `loop-events.ndjson`.

## Data Structures

- Payload structs for `registry.rebuilt` (added/removed string slices), `bootstrap.completed` (empty), `bootstrap.exhausted` (reason string), `order.dropped` (order ID, reason)

## Routing

Provider: `codex`, Model: `gpt-5.4`

## Verification

```bash
go build ./... && go test ./loop/... && go vet ./...
```

- No references to `QueueAuditEvent` or `appendQueueEvent` remain in non-test, non-archived code
- No references to `queue-events.ndjson` remain in `loop/` package
- The four redirected write sites emit to `loop-events.ndjson` via `EventWriter`
- Existing loop tests still pass
