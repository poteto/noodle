Back to [[plans/95-orders-json-ownership/overview]]

# Phase 3 — Startup Integrity via State Marker

## Goal

Detect when orders.json was modified while the loop was stopped (user killed and restarted noodle). Warn on restart so the scheduler can self-correct.

## Changes

- **`internal/statever/statever.go`** — add `OrdersHash` field to `StateMarker`
- **`loop/state_orders.go`** — stamp hash on write, verify at startup
- **`loop/types.go`** — add dedicated startup warning field
- **`loop/schedule.go`** — inject startup warning into first scheduler prompt

## Data structures

- `StateMarker.OrdersHash` — `string` field, hex-encoded SHA-256 of orders.json content at last write

## Details

### Stamp hash on write

`writeOrdersState()` computes SHA-256 of the serialized orders data and stores it in a loop field (e.g. `l.lastOrdersHash`). `flushState()` includes this hash in the `StateMarker` it writes to state.json:

```go
statever.Write(path, statever.StateMarker{
    SchemaVersion: statever.Current,
    GeneratedAt:   now().UTC(),
    OrdersHash:    l.lastOrdersHash,
})
```

This is a single atomic write — no separate sig file, no crash gap between orders.json and its hash.

### Verify at startup

Before `reconcile()` runs, the startup path:
1. Reads `state.json` (already done for version compatibility check)
2. Reads `orders.json` and computes SHA-256
3. Compares against `StateMarker.OrdersHash`
4. On mismatch: log at warn level, set a dedicated field (e.g. `l.startupIntegrityWarning`)
5. On match or missing hash (first run / upgrade): no warning

### Warning injection

Dedicated field `startupIntegrityWarning string` on the Loop struct:
- Set at startup if integrity check fails
- Included in `buildSchedulePrompt()` as a distinct section (not "ADAPTER WARNINGS")
- Cleared after the first scheduler dispatch completes (not per-cycle like `lastMiseWarnings`)
- Also written to `status.json` for UI visibility

### First-run / upgrade handling

Missing `OrdersHash` in state.json (zero value) → no warning. The first `flushState` writes the hash. Users upgrading from before this feature won't see a false positive.

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `codex` | `gpt-5.3-codex` | Mechanical implementation against clear spec |

## Verification

### Static
- `go vet ./internal/statever/... ./loop/...`
- `go build ./...`

### Runtime
- Stop loop, tamper orders.json, restart → warning in logs and scheduler prompt
- Stop loop, don't tamper, restart → no warning
- Fresh project / upgrade with no hash in state.json → no warning, hash written on first flush
