Back to [[plans/70-scaling-the-loop-redesign/overview]]

# Phase 2: Push-based completion channels

## Goal

Replace per-cycle iteration of all active sessions with a single completion channel. When a session finishes, it pushes a result to the channel. The loop drains this channel at the start of each cycle — O(completions) instead of O(sessions).

## Changes

**`loop/types.go`** — Add `StageResult` type. Add `completions chan StageResult` to the Loop struct. The channel is buffered generously (e.g., 1024) and watcher goroutines use non-blocking send with a fallback mutex-guarded overflow slice. The loop goroutine is the sole consumer: it drains the channel, then locks and drains the overflow. Only the loop goroutine reads the overflow slice — no concurrent read/write. This is the standard fan-in-with-overflow pattern.

**`loop/cook.go`** — `spawnCook()` starts a goroutine per dispatched session that blocks on `session.Done()`, then sends a `StageResult` to the completions channel. `collectCompleted()` becomes `drainCompletions()` — drains the channel then the overflow slice. **Dedup by order ID + stage index + attempt**: if a result arrives for a stage already processed (e.g., `Kill()` closes `Done()` and control path also emits a result), skip the duplicate. The attempt number in the dedup key prevents a late-arriving result from attempt N being confused with attempt N+1 after a retry. Track processed stage keys in a set cleared each cycle.

**Bootstrap session**: The existing `bootstrapInFlight` / `collectBootstrapCompletion()` path is migrated to the same watcher goroutine pattern. The bootstrap session gets a watcher goroutine that sends a `StageResult` on completion, same as any cook. Remove `collectBootstrapCompletion()` — bootstrap is no longer a special case for completion detection.

**Schedule session**: `spawnSchedule()` is migrated to use the same dispatch lifecycle as `spawnCook()`. The schedule session gets a watcher goroutine on `Done()`. Schedule completion is communicated via the same `StageResult` channel (with a sentinel `OrderID` like `"__schedule__"` or a flag). This eliminates the separate code path where schedule sessions bypass the status-derived busy set.

**`loop/loop.go`** — `runCycleMaintenance()` calls `drainCompletions()` first. The cycle no longer needs to know how many sessions exist — it only processes events that arrived since last cycle. **Shutdown quiescence**: the drain-exit condition is producer-count-based, not channel-emptiness: `len(activeCooksByOrder) == 0` (all watcher goroutines have exited). When all producers have exited, the completions channel is guaranteed empty. After the producer count reaches zero, do one final drain pass to process any results that arrived between the last check and now. Note: merge-results channel drain (phase 7) will extend this condition later — keep the shutdown logic extensible but don't reference it here.

**`loop/control.go`** — Control commands that kill sessions (skip, reject) do NOT separately emit `StageResult` entries. Instead, they call `session.Kill()` which closes `Done()`, and the existing watcher goroutine produces the `StageResult`. This eliminates the duplicate-event problem at the source. The watcher goroutine reads `session.Status()` after `Done()` closes to determine the result status.

## Data structures

- `StageResult` — `OrderID string`, `StageIndex int`, `Attempt int`, `IsOnFailure bool`, `Status StageResultStatus` (typed constant: completed/failed/cancelled), `SessionID string`, `WorktreeName string`, `WorktreePath string`, `Error error`
- `StageResultStatus` — typed string constant (`StageResultCompleted`, `StageResultFailed`, `StageResultCancelled`)

## Routing

- Provider: `claude` | Model: `claude-opus-4-6` — concurrency design requires careful reasoning about channel semantics, dedup, and shutdown quiescence

## Verification

### Static
- `go test ./...` — all loop tests pass
- `go vet ./...` — no goroutine leaks flagged
- `collectCompleted()` no longer exists
- No iteration over `activeCooksByOrder` for completion detection
- Control commands do not write to the completions channel directly

### Runtime
- Integration test: dispatch 3 sessions, complete them in arbitrary order, verify all 3 produce StageResults and stages advance correctly
- Race detector: `go test -race ./loop/...`
- Test: session that fails produces a StageResult with failed status
- Test: control skip calls Kill(), watcher goroutine produces cancelled StageResult (no duplicate)
- Test: shutdown waits for producer count to reach zero, then does final drain
- Test: burst of 50 completions in one cycle — overflow slice drains correctly, all processed
- Test: bootstrap session completes via watcher goroutine, no special-case collection
- Test: schedule session completes via watcher goroutine, result processed correctly
- Test: late StageResult from attempt 1 ignored when attempt 2 is already active (dedup by attempt)
