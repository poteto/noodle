Back to [[plans/87-go-codebase-simplification/overview]]

# Phase 10: Decompose `reconcileMergingStages` + `handleCompletion`

## Goal

Break two long functions in the loop package into focused sub-functions.

## Changes

### 10a: `reconcileMergingStages` (143 lines, `reconcile.go:182-324`)

Mixed concerns: metadata validation, git branch checks, adoption checks, event emission, state mutation. Deep nesting.

**Also fix:** `reconcile.go:47` swallows `currentOrders()` errors and defaults adopted stage index to `0`. This means a corrupted/unreadable orders file during recovery silently mis-associates adopted sessions to stage 0. Stop swallowing that error — either fail reconciliation deterministically or degrade with explicit fallback and log output.

Extract:
- `extractMergeMetadata(stage) (MergeMetadata, error)` — validate and extract
- `handleAlreadyMergedStage(cook, metadata) error` — recovery path for already-merged
- `requeueStaleMerge(cook, metadata) error` — re-enqueue stalled merge
- `failMissingBranchStage(cook, metadata) error` — branch-gone failure path

Keep `reconcileMergingStages` as the orchestrator with flat guard-clause structure.

### 10b: `handleCompletion` (140 lines, `cook_completion.go:113-252`)

Mixed concerns: schedule vs regular completion, stage messages, merge vs non-merge paths.

Extract:
- `handleScheduleCompletion(cook, output) error` — schedule-specific path
- `processStageMessage(cook, output) (bool, string)` — extract and check stage messages
- `completeWithMerge(ctx, cook, msg) error` — merge queue or direct merge
- `completeWithoutMerge(ctx, cook, msg) error` — non-worktree completion

Keep `handleCompletion` as the top-level router.

## Data Structures

No new types. Sub-functions are private methods on `*Loop`.

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`
- Both functions have intricate control flow that requires careful understanding to decompose correctly.

## Verification

### Static
- `go test ./loop/...` — all loop tests pass
- `go vet ./loop/...` — clean
- No function in modified files exceeds 60 lines

### Runtime
- `go test ./...` — full suite passes
