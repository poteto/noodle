# Phase 03: Loop, Runtime, and Control Safety

Back to [[plans/112-codebase-simplification-audit/overview]]

## Goal

Make loop/control/reconcile/runtime behavior deterministic, restart-safe, and bounded under failure.

## Depends on

- [[plans/112-codebase-simplification-audit/phase-02-canonical-contract-cutline]]

## Findings in scope

- `25-47`, `71-75`

## Changes

- Key stage transitions by explicit order/stage identity and remove ambiguous failure targeting.
- Consolidate duplicated failure/merge/control/reconcile orchestration into shared deterministic primitives.
- Harden control dedupe and pending-review persistence for restart safety.
- Replace fragile session metadata string parsing with typed metadata boundaries.
- Enforce bounded shutdown semantics for session and merge lifecycles.

## Data structures

- Explicit stage transition keying model.
- Persisted control identity/dedupe model.
- Typed session metadata store used by runtime/reconcile.

## Within-phase priority

- **P0 first:** `41` (pending-review data loss on restart), `44` (corrupt pending-review treated as empty), `71` (shutdown hangs indefinitely), `73` (recovery skips live sessions).
- **P1 next:** `25`, `27`, `28`, `31`, `32`, `36`, `37`, `38`, `40`, `42`, `43`, `45`, `72`, `74`, `75`.
- **P2 last:** `26`, `29`, `30`, `33`, `34`, `35`, `39`, `46`, `47`.

## Done when

- Stage failure/advance behavior is deterministic across replay/restart.
- Control commands are idempotent under crash/replay scenarios.
- Shutdown has a measurable upper bound under forced hangs.
- Runtime recovery decisions use typed metadata parsing only.

## Verification

### Static
- `go test ./loop ./runtime ./monitor ./worktree ./server`
- `go test -race ./loop ./runtime ./server`

### Runtime
- Crash/restart fixture replay for control/pending-review/reconcile paths.
- Forced hang tests for merge/session termination with asserted shutdown timeout bounds.

## Rollback

- Keep loop transition, dedupe, and runtime metadata changes in separable commits.
- If bounded-shutdown regresses, revert shutdown segment independently while retaining deterministic transition fixes.
