Back to [[archive/plans/95-orders-json-ownership/overview]]

# Phase 5 — Tests

## Goal

Verify tamper resilience for both the normal-operation and restart paths.

## Changes

- **`internal/statever/statever_test.go`** — unit tests for OrdersHash in state marker
- **`loop/orders_test.go`** — test consumeOrdersNext refactor
- **`loop/` test file** — loop-level tamper resilience tests

## Details

### State marker tests (statever)

- Write marker with OrdersHash, read back → hash preserved
- Missing OrdersHash (upgrade from older version) → zero value, no error

### consumeOrdersNext tests

- Basic merge: existing orders + orders-next.json → correct merged result (no disk read of orders.json)
- No orders-next.json → returns not-promoted
- Invalid orders-next.json → renamed to .bad, returns error
- Duplicate order handling preserved (skip, replace failed, amend active)
- orders-next.json NOT deleted by consumeOrdersNext (caller responsibility)

### Loop-level tamper tests

- **During operation:** Write orders.json externally between cycles → loop ignores it (uses in-memory state), flushState overwrites with authoritative state
- **During operation with fsnotify:** External orders.json write does NOT trigger a cycle (not watched)
- **On restart:** Tamper orders.json while stopped → startup logs warning, dedicated field set, warning appears in first scheduler prompt, cleared after dispatch
- **On restart with malformed JSON:** Tamper with invalid JSON → existing strict parse error handling applies (loop degrades gracefully)
- **First startup / upgrade:** No hash in state.json → no warning, hash written on first flush
- **Crash recovery:** Loop crashes mid-cycle → restart loads from disk (normal), no false tamper warning because state.json hash matches orders.json (both written in same flushState)

Use existing fixture patterns (directory-based state fixtures).

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `codex` | `gpt-5.3-codex` | Test implementation against clear spec |

## Verification

### Static
- `go vet ./...`

### Runtime
- `go test ./internal/statever/... -run OrdersHash -v`
- `go test ./loop/... -run Consume -v`
- `go test ./loop/... -run Tamper -v`
- `pnpm check`
