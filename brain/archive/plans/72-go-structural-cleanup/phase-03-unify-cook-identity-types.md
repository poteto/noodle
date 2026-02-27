Back to [[plans/72-go-structural-cleanup/overview]]

# Phase 3: Unify cook identity types

## Goal

Eliminate field duplication across `cookHandle`, `pendingReviewCook`, and `pendingRetryCook` by extracting a shared `cookIdentity` base struct. Remove the field-by-field copying that currently happens when converting between these types.

Per [[principles/foundational-thinking]]: "Get the data structures right before writing logic."

## Changes

**`loop/types.go`** — Define a shared identity struct and embed it:

- `cookIdentity` — holds the fields shared across all three types: orderID, stageIndex, stage, plan
- `cookHandle` embeds `cookIdentity`, adds session-specific fields (session, worktreeName, worktreePath, attempt, generation, startedAt, displayName, isOnFailure, orderStatus)
- `pendingReviewCook` embeds `cookIdentity`, adds review-specific fields (worktreeName, worktreePath, sessionID, reason)
- `pendingRetryCook` embeds `cookIdentity`, adds retry-specific fields (isOnFailure, orderStatus, attempt, displayName)

**`loop/*.go`** — Simplify all construction sites throughout the package. Anywhere a `cookHandle`, `pendingReviewCook`, or `pendingRetryCook` is constructed by copying fields from another handle type, the embedded `cookIdentity` can be copied as a single value instead.

Key sites include `spawnCook`, `applyStageResult`, `retryCook`, `controlMerge`, `processPendingRetries`, `drainMergeResults`, `parkPendingReview`, `loadPendingReview`, `loadPendingRetry`, `buildAdoptedCook`, plus reconcile and schedule construction sites.

## Data structures

```
cookIdentity { orderID, stageIndex, stage, plan }
```

Embedded by all three handle types. The specific fields on each type remain as they are — they represent genuinely different lifecycle state.

## Verification

- `go test ./...` — all tests pass
- `go vet ./...` — no issues
- Manual review: no field-by-field struct copying between handle types (all use embedded identity)
