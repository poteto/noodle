Back to [[plans/115-canonical-state-convergence/overview]]

# Phase 5 — Merge Recovery And Review-State Hardening

## Goal

Move merge success/failure and restart-time recovery onto canonical authority after review ownership is already canonical, so restart correctness does not depend on legacy order or review files lingering for later phases.

## Changes

- **`loop/cook_merge.go`** — stop imperatively advancing order state after merge; emit canonical merge outcomes only
- **`loop/reconcile.go`** — converge `merging` stages and canonical review state from canonical recovery data instead of legacy `orders.json` pruning rules
- **`internal/reducer/reducer.go` + canonical checkpoint writer** — persist merge-recovery metadata and the already-canonical review state in the same crash-safe boundary as the canonical checkpoint
- **legacy review/recovery helpers** — delete superseded direct order/review mutation paths as part of this phase

## Data structures

- merge outcome payload carrying `order_id`, `stage_index`, worktree identity, and recoverable merge metadata
- persisted merge-recovery record stored inside the canonical durable snapshot with explicit write ordering
- canonical review-state checkpoint content sufficient for restart pruning, merge approval, request-changes, and reject without any separate legacy sidecar authority

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | Merge, review, and restart semantics are tightly coupled architectural work |

## Verification

### Static
- `pnpm test:smoke`
- `go test ./internal/reducer/... ./internal/integration/... ./loop/...`
- `go vet ./internal/reducer/... ./loop/...`

### Runtime
- prove successful merges advance only through canonical transitions
- prove merge failures, conflicts, approval, request-changes, and reject preserve existing operator behavior once review state is already canonical
- prove a crash during `merging` or while parked for review leaves enough canonical metadata for restart-time convergence without legacy order or review surgery
- confirm the superseded merge/review recovery paths are deleted in the same phase
- rerun `pnpm test:smoke` after merge/recovery hardening and treat unexpected failures as blockers
