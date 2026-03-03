# Phase 01: Canonical Model Unification

Back to [[plans/112-codebase-simplification-audit/overview]]

## Goal

One state model, one event registry, one projection path. Remove the `internal/state` vs `internal/orderx` split and everything downstream that exists only to translate between them.

## Findings in scope

- `48-64`, `67`

## Priority

- **P0 first:** `50` (silent reducer decode failure), `64` (silent event loss).
- **P1 next:** `48` (model split — the root cause), `49`, `53`, `54`, `56`, `60`, `67`.
- **P2 last:** dead code, unused packages, fixture cleanup (`51`, `52`, `55`, `57-63`).

## Changes

- Choose one canonical lifecycle model and status enum. Delete the other.
- Replace duplicated event registries with one typed registry (type → decode → reducer handler).
- Align projection output with canonical state. Remove hand-maintained projection translation.
- Make unknown-provider event routing explicit and non-lossy (`64`).
- Unify provider action normalization into the registry path (`67`).
- Remove orphaned packages: `internal/dispatch`, `internal/rtcap` (if confirmed dead), `internal/recover` (fold).

## Data structures

- Canonical lifecycle model and status enums (replaces `internal/state` + `internal/orderx` split).
- Unified event registry entry (replaces ingest/reducer contract duplication).
- Projection DTOs generated from canonical state (replaces hand-maintained shapes).

## Migration

This phase changes serialized contract shapes. Existing `.noodle/` state may use the old format.

- Persisted `orders.json` and `pending-review.json` must be readable by the new model or explicitly migrated.
- If migration is not feasible, document the breaking change and require state directory reset. State reset requires explicit lead approval.
- No silent format incompatibility — startup must detect and fail loudly on unreadable state.

## Done when

- One authoritative lifecycle model exists. The other is deleted, not demoted.
- Event types are exhaustively handled (compiler or test rejects unhandled types).
- Projection output accepted by live readers without translation shims.
- Unknown-provider events surfaced, not silently dropped.

## Verification

### Static
- `go test ./internal/... && go vet ./internal/...`
- Contract round-trip tests: old-format fixtures through new model, assert lossless output.
- Event registry exhaustiveness enforced by compiler or test.

### Runtime
- Replay event fixtures through reducer, compare output against golden snapshots.
- Restart from persisted state (old and new format), assert convergence or explicit migration error.

## Rollback

- Snapshot `.noodle/` state before rollout.
- Keep model/registry/projection changes in separable commits so the cutline can be reverted in stages.
