Back to [[plans/38-resilient-skill-resolution/overview]]

# Phase 3: Remove runtime repair

**Routing:** `claude` / `claude-opus-4-6` — state machine simplification, race condition fix, affects all error paths

## Goal

Delete the runtime repair system entirely. It has never successfully repaired anything (all repair sessions fail with duration 0), adds ~300 lines of state machine complexity, and creates a double-spawn race condition where queue items fall out of all tracking maps.

The double-spawn race: when `processPendingRetries` clears the `pendingRetry` map and then `spawnCook` fails, `handleRuntimeIssue` swallows the error. The item is no longer in `activeByTarget` (cleared on completion), no longer in `pendingRetry` (cleared before spawn), and not in `failedTargets`. Next cycle, `planCycleSpawns` sees the item in the queue with no busy/failed entry and spawns it fresh — creating a duplicate.

Transient failures recover via the existing retry mechanism (`retryCook` + `pendingRetry` + `maxRetries`). Fatal infrastructure failures should bubble up to the human, not get absorbed into a repair loop.

## Changes

**Delete `loop/runtime_repair.go` entirely.**

**Delete all runtime-repair test fixtures:**
- `loop/testdata/runtime-repair-*` (all directories)
- `loop/testdata/missing-sync-*` fixtures that reference repair behavior
- Update `loop/fixture_test.go` to remove any repair-specific setup/assertions

**Remove from Loop struct (`loop/types.go`):**
- `runtimeRepairInFlight *runtimeRepairState`
- `runtimeRepairAttempts map[string]int`
- `runtimeRepairState` struct definition
- `runtimeIssue` struct definition

**Remove from `loop/loop.go`:**
- `handleRuntimeIssue()` method — delete entirely
- `advanceRuntimeRepair()` call in `runCycleMaintenance()`
- `runtimeRepairInFlight != nil` guard in `runCycleMaintenance()`
- `runtimeRepairAttempts` initialization in `New()`

**Replace all `handleRuntimeIssue` call sites with direct error handling:**

| Call site | Old behavior | New behavior |
|-----------|-------------|--------------|
| `loop.monitor` error | Spawn repair | Return error (fatal — monitoring is broken) |
| `loop.collect` error | Spawn repair | Return error (let caller handle) |
| `loop.spawn` error | Spawn repair | Return error from `spawnPlannedItems` |
| `loop.control` error | Spawn repair | Return error |
| `loop.queue-next` error | Spawn repair | Log warning, continue (transient file issue) |
| `loop.pending-retry` error | Spawn repair, lose item | Mark item failed, continue loop |
| `mise.build` error | Spawn repair | Return error |

**Fix the `pendingRetry` race in `processPendingRetries()`:**
- When `spawnCook` fails for a pending retry item, do NOT lose the item. Instead: increment attempt, check against `maxRetries`. If exhausted, call `markFailed` + `skipQueueItem`. If not exhausted, put back in `pendingRetry` for next cycle.
- This is the core fix — the item must always be tracked somewhere until it's either active or explicitly failed.

## Data structures

Deleted:
- `runtimeRepairState`
- `runtimeIssue`
- `runtimeRepairInFlight`
- `runtimeRepairAttempts`
- `handleRuntimeIssue()`
- `advanceRuntimeRepair()`
- `ensureRuntimeRepair()`
- `spawnRuntimeRepair()`
- `runtimeRepairSkill()`
- `buildRuntimeRepairPrompt()`
- `runtimeIssueFingerprint()`
- `adoptRunningRuntimeRepair()`
- `findRunningRuntimeRepairSessionID()`

## Verification

```bash
go test ./loop/... && go vet ./...
sh scripts/lint-arch.sh
```

Tests:
- Spawn failure for a queue item → item is retried up to `maxRetries`, then marked failed and skipped. No repair session spawned.
- `processPendingRetries` spawn failure → item stays in `pendingRetry` (not lost). Next cycle retries it.
- `processPendingRetries` spawn failure after max retries → item marked failed, skipped from queue, loop continues.
- Monitor error → loop returns error (fatal exit, not swallowed).
- Queue-next consumption error → logged, cycle continues.
- No `runtimeRepairInFlight` guard blocking cycle progression.
