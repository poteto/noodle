# Phase 02: Canonical Contract Cutline

Back to [[plans/112-codebase-simplification-audit/overview]]

## Goal

Establish one canonical state/order/event contract and remove parallel internal representations that cause translation drift.

## Depends on

- [[plans/112-codebase-simplification-audit/phase-00-scaffold-and-gates]]

## Findings in scope

- `48-63`, `64`, `67`

## Within-phase priority

- **P0 first:** `50` (silent reducer decode failure), `64` (silent event loss).
- **P1 next:** `48` (model split — root cause of most other findings), `49`, `53`, `54`.
- **P2 last:** dead code removal, unused breadth, package folding (`51`, `52`, `55-63`, `67`).

## Changes

- Define authoritative model ownership and demote boundary DTOs accordingly.
- Replace duplicated event registries/switches with one typed registry surface.
- Remove or fold orphaned/internal parallel packages that duplicate live routing/contract behavior.
- Align projection output shape with canonical contract.
- Make unknown-provider event routing explicit and non-lossy (`64`).
- Unify provider action/tool normalization into shared registry path (`67`).

## Data structures

- Canonical lifecycle model and status enums (replaces split `internal/state` + `internal/orderx` models).
- Unified event registry entry: type, decode, idempotency key, reducer handler (replaces ingest/reducer contract duplication).
- Projection DTOs generated from canonical state (replaces hand-maintained projection shapes).

## Migration

This phase changes serialized contract shapes. Existing `.noodle/` state directories may contain files in the old format.

- Persisted `orders.json` and `pending-review.json` must be readable by the new model or explicitly migrated.
- If migration is not feasible, document the breaking change and require state directory reset on upgrade.
- No silent format incompatibility — startup must detect and fail loudly on unreadable state.

## Done when

- There is one authoritative internal lifecycle model.
- Contract conversion points are explicit boundary transforms, not peer canonical models.
- Projection output is accepted by live readers/validators without compatibility shims.
- Unknown-provider events are surfaced, not silently dropped.

## Verification

### Static
- `go test ./internal/... && go vet ./internal/...`
- Contract round-trip tests: marshal old-format fixtures through new model, assert lossless output.
- Event registry exhaustiveness: compiler or test must reject unhandled event types.

### Runtime
- Replay event fixture files (`testdata/events/`) through reducer and compare output state against golden snapshots.
- Restart from persisted state snapshots (old and new format) and assert convergence or explicit migration error.
- Verify unknown-provider events are surfaced in diagnostics, not silently dropped.

## Rollback

- Snapshot state files before cutline change.
- Preserve reversible migration commits (registry/model/projection split) so contract cutline can be reverted in stages.
