Back to [[archive/plans/66-event-trigger-system/overview]]

# Phase 5 — Emit Lifecycle Events: Control Commands and Recovery

## Goal

Add event emission at control command paths and crash-recovery paths. These cover the remaining state transitions not handled by phase 4.

## Changes

- **`loop/control.go`**:
  - `controlRequeue` — emit `order.requeued` (order ID).
  - `controlMerge` quality-gate rejection path (~line 277-300) — emit `stage.failed` and conditionally `order.failed` when the manual merge path hits a quality rejection. Currently this path calls `failStage` and `markFailed` but emits no events.
  - `controlReject` / `controlRequestChanges` — emit `stage.failed` (order ID, reason).
- **`loop/reconcile.go`**:
  - "already merged, advance" path (~line 150-162) — emit `stage.completed` with session ID as nil (no session in this path). The `cookHandle` built here has no session, so the payload must treat session ID as optional.
  - Other reconcile state transitions that call `advanceAndPersist` or `failAndPersist` — if phase 4 placed emission inside those functions, these paths are already covered. Verify and add only if needed.

## Data Structures

- `order.requeued` payload: order ID
- Reuse existing payload structs from phase 4 for `stage.failed`, `order.failed`, `stage.completed`

## Routing

Provider: `codex`, Model: `gpt-5.4`

## Verification

```bash
go test ./loop/... && go vet ./...
```

- `requeue` control command emits `order.requeued`
- Manual merge with quality rejection emits `stage.failed` (and `order.failed` when terminal)
- `reject` and `request-changes` emit `stage.failed`
- Reconcile's "already merged" path emits `stage.completed` with nil session ID (no panic, no invalid payload)
- Events from control commands have valid sequences
