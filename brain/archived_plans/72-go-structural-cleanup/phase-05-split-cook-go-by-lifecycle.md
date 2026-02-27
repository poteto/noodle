Back to [[plans/72-go-structural-cleanup/overview]]

# Phase 5: Split cook.go by lifecycle

## Goal

Break the 1067-line `cook.go` into files organized by lifecycle phase. Each file handles one stage of a cook's life. No logic changes — pure file reorganization.

## Changes

**`loop/cook_spawn.go`** — Spawning and dispatch:
- `spawnCook`
- `dispatchSession`
- `ensureWorktree`
- `ensureSkillFresh`
- `spawnOptions` type
- `worktreePath`
- `resetWorktreeState`
- `atMaxConcurrency`
- `persistOrderStageStatus`

**`loop/cook_completion.go`** — Completion handling:
- `drainCompletions` (the Loop method that calls applyStageResult)
- `applyStageResult`
- `handleCompletion`
- `handleBootstrapResult`
- `advanceAndPersist`
- `failAndPersist`
- `readQualityVerdict`
- `collectAdoptedCompletions`
- `removeOrder`

**`loop/cook_merge.go`** — Worktree merging:
- `mergeCookWorktree`
- `resolveMergeMode`
- `persistMergeMetadata`
- `canMergeStage`
- `readSessionSyncResult`
- `handleMergeConflict`
- `jsonQuote`
- Merge metadata constants

**`loop/cook_retry.go`** — Retry and recovery:
- `retryCook`
- `processPendingRetries`
- `retryFailureReason`
- `buildAdoptedCook`
- `readSessionStatus`
- `dropAdoptedTarget`

**`loop/cook_steer.go`** — Steering and session control:
- `steer`
- `killCook`
- `buildSteerResumeContext`

**`loop/cook_watcher.go`** — Session watcher goroutine:
- `startSessionWatcher`
- `stageResultStatus`
- `nextDispatchGeneration`

**Delete `loop/cook.go`** — all content moves to the above files.

Note: if Phase 4 already moved `enqueueCompletion` / `takeCompletionOverflow` to `completion_buffer.go`, they stay there. Adjust split accordingly.

## Verification

- `go test ./...` — all tests pass
- `go vet ./...` — no issues
- `wc -l loop/cook*.go` — no file over 300 lines
- `grep -r 'package loop' loop/cook*.go` — all files in same package
