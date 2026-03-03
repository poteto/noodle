# Phase 02: Loop Orchestration Consolidation

Back to [[plans/112-codebase-simplification-audit/overview]]

## Goal

One failure/merge/completion orchestration path. The duplication across completion, control, and reconcile is the root cause of the P0 data-loss bugs — fixing the duplication and the bugs is the same work.

## Depends on

- [[plans/112-codebase-simplification-audit/phase-01-canonical-model]]

## Findings in scope

- `26`, `35-47`, `71`

## Priority

- **P0 first:** `41` (pending-review data loss on restart), `44` (corrupt pending-review treated as empty), `71` (shutdown hangs indefinitely).
- **P1 next:** `26`, `36`, `38`, `40`, `42`, `43`, `45`.
- **P2 last:** `35`, `37`, `39`, `46`, `47`.

## Changes

- Consolidate failure/merge/completion orchestration into one deterministic path used by completion, control, and reconcile.
- Key stage transitions by explicit `(order_id, stage_index)` identity.
- Make control command dedupe restart-safe with persisted identity model.
- Replace string-based session/adoption status parsing with typed metadata.
- Enforce bounded shutdown for merge/session termination.
- Surface corrupt `pending-review.json` instead of treating as empty.

## Data structures

- Unified orchestration primitive used by all three paths (replaces duplicated failure/merge logic).
- Persisted control identity/dedupe model (replaces non-restart-safe in-memory dedupe).
- Typed session metadata (replaces string-based status parsing).

## Migration

This phase changes persisted control and pending-review structures.

- Existing `pending-review.json` must be readable by the new model or explicitly migrated.
- Control dedupe state transition must handle pre-phase format gracefully (first run after upgrade reconciles).
- Startup must fail loudly on unreadable control/metadata state.

## Done when

- Failure/merge/completion logic exists in one path, called from completion/control/reconcile.
- Stage failure/advance is deterministic across restart/replay.
- Control commands are idempotent under crash/replay.
- Shutdown has a measurable upper bound.
- Corrupt `pending-review.json` is surfaced, not silently ignored.

## Verification

### Static
- `go test ./loop/... && go vet ./loop/...`
- `go test -race ./loop/...`

### Runtime
- Crash/restart replay tests for control, pending-review, and reconcile.
- Forced-termination tests with asserted shutdown timeout bounds.

## Rollback

- Snapshot `.noodle/` state before rollout.
- Keep orchestration consolidation, dedupe, and shutdown changes in separable commits.
