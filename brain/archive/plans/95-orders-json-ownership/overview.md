---
id: 95
created: 2026-03-10
status: archived
---

# Orders.json Exclusive Backend Ownership

Back to [[archive/plans/index]]

Subsumed by [[plans/115-canonical-state-convergence/overview]].

## Context

The loop exclusively owns `orders.json` in practice — all writes go through `writeOrdersState()` / `flushState()` / `consumeOrdersNext()`. But the protection is **instruction-based only**: the schedule skill says "never write orders.json directly." Per serialize-shared-state-mutations and encode-lessons-in-structure, instructions are insufficient — enforcement must be structural.

The deeper issue: the loop re-reads `orders.json` from disk on every cycle (`Cycle()` calls `loadOrdersState()` at `loop.go:401`), and fsnotify watches orders.json changes (`loop.go:382`), triggering an immediate cycle + reload. This means an agent write to orders.json is ingested into in-memory state on the very next tick. Additionally, `consumeOrdersNext()` reads orders.json from disk to merge with incoming orders-next.json, even though the loop has the authoritative state in memory.

If we'd designed this from scratch knowing the loop exclusively owns orders.json, it would never read orders.json from disk during normal operation. In-memory state is authoritative; disk is persistence. The only disk read happens once at startup for crash recovery.

## Scope

**In scope:**

- Refactor `consumeOrdersNext` to accept in-memory state instead of reading from disk
- Remove the per-cycle `loadOrdersState()` call in `Cycle()`
- Stop watching orders.json via fsnotify (only watch orders-next.json and control.ndjson)
- Collapse orders.json write paths behind a single boundary
- Startup integrity check embedded in existing state.json marker
- Remove stale "never write orders.json" instruction from schedule skill
- Tests for tamper resilience

**Out of scope:**

- OS-level file permissions (agents run as same user)
- Preventing agents from reading orders.json (read access preserved)
- Changes to orders-next.json format or promotion validation (already correct)

## Design

Two complementary fixes for two different concerns:

**During operation (structural, preventive):** The loop never reads orders.json from disk after startup. Remove the per-cycle `loadOrdersState()` in `Cycle()`, remove orders.json from the fsnotify watch, and refactor `consumeOrdersNext` to merge against in-memory state. Agent writes to orders.json are structurally harmless — the next `flushState()` overwrites them, and they're never read.

**On restart (detection, advisory):** `flushState()` already writes `state.json` (state marker) alongside orders.json. Add an `OrdersHash` field to `StateMarker` containing the SHA-256 of the last-written orders.json. At startup, verify the hash before `reconcile()` runs. On mismatch: warn (log + dedicated field for scheduler prompt injection), but still load. System self-corrects next scheduler cycle.

## Alternatives considered

1. **Full watermark system** (per-cycle verify + recovery) — Bolt-on detection, ~200 lines. Treats the symptom (tampered reads) instead of eliminating the cause (unnecessary disk reads). Rejected.
2. **Separate .sig sidecar file** — Two atomic writes with a crash gap between them. False tamper warnings after interrupted writes. Every write path must remember to stamp. Rejected in favor of embedding in the existing state marker.
3. **Fail-closed on startup mismatch** — Enter degraded/manual mode if orders.json was tampered while stopped. Heavyweight for a rare scenario. Rejected in favor of warn-and-load (scheduler self-corrects).
4. **In-memory authority + state marker integrity** (chosen) — Remove disk reads during operation. Embed hash in existing state.json for startup verification. Minimal new code, maximum structural enforcement.

## Applicable skills

- `go-best-practices` — Go patterns
- `testing` — Fixture-based tests

## Phases

1. [[archive/plans/95-orders-json-ownership/phase-01-in-memory-merge]]
2. [[archive/plans/95-orders-json-ownership/phase-02-remove-cycle-reload]]
3. [[archive/plans/95-orders-json-ownership/phase-03-startup-integrity]]
4. [[archive/plans/95-orders-json-ownership/phase-04-cleanup]]
5. [[archive/plans/95-orders-json-ownership/phase-05-tests]]

## Verification

- `pnpm check` (full suite)
- `go test ./internal/statever/... ./loop/...`
- Manual: tamper orders.json while loop is running → ignored, overwritten by flushState
- Manual: tamper orders.json while loop is stopped → restart shows warning, system self-corrects
