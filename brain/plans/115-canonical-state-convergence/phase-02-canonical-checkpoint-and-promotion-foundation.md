Back to [[plans/115-canonical-state-convergence/overview]]

# Phase 2 — Canonical Checkpoint And Promotion Foundation

## Goal

Make canonical state observable and durable on the production path before any caller cutover starts, define the one-time bootstrap from legacy state into the first canonical checkpoint, restore canonical identity correctly after restart, and move scheduler promotion onto canonical order creation instead of leaving it as a legacy side channel.

## Changes

- **`loop/loop.go`** — replace the current in-memory-only reducer shadow path with production durable canonical snapshot/effect-ledger persistence after accepted events
- **legacy startup import path** — when no canonical checkpoint exists yet, import `orders.json` plus `pending-review.json` once, persist the canonical checkpoint immediately, and mark the bootstrap complete before normal loop execution continues
- **`loop/loop.go` + canonical startup path** — restore the next event identity from the durable checkpoint so event IDs, effect IDs, and projection versions remain monotonic across restart
- **`internal/reducer/snapshot.go`** — become the production checkpoint format, not just a test helper
- **`loop/loop_cycle_pipeline.go`** — route `orders-next.json` promotion through canonical order creation, merge against in-memory order state rather than re-reading disk `orders.json`, and explicitly preserve cooldown/bootstrap/amendment behavior
- **`loop/loop.go` + `loop/state_orders.go`** — remove per-cycle `orders.json` reload/watch behavior so normal operation stops treating disk `orders.json` as authoritative once canonical startup state is loaded
- **`loop/state_orders.go` + `loop/pending_review.go`** — restrict legacy reads/writes to the temporary bootstrap boundary only, with explicit deletion of bootstrap-only authority once later phases cut over

## Data structures

- `reducer.DurableSnapshot` — persisted canonical baseline for restart and parity
- bootstrap import record or startup invariant proving the first checkpoint was synthesized from legacy state exactly once
- restored canonical event identity derived from the durable checkpoint's last applied event
- canonical promotion event payloads carrying enough scheduler/order metadata to create orders without legacy order mutation

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | Foundational migration scaffold and crash-ordering judgment |

## Verification

### Static
- `pnpm test:smoke`
- `go test ./internal/reducer/... ./internal/integration/... ./loop/...`
- `go vet ./internal/reducer/... ./loop/...`

### Runtime
- prove the live loop writes durable canonical snapshot/effect-ledger state on the production path
- prove the first restart without a checkpoint imports legacy order/review state once, writes the initial canonical checkpoint, and does not require a drained repo
- prove restart can load that checkpoint without relying on a fresh in-memory canonical rebuild
- prove post-restart event IDs, effect IDs, and projection versions always advance past the durable checkpoint
- prove schedule promotion now creates canonical orders while preserving empty-promotion cooldown, bootstrap schedule insertion, and active-order amendment behavior
- prove external `orders.json` writes during normal operation are no longer ingested through per-cycle reloads or fsnotify triggers
- rerun `pnpm test:smoke` after the foundation cutover and treat unexpected failures as blockers
