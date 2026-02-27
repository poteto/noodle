Back to [[archive/plans/70-scaling-the-loop-redesign/overview]]

# Phase 2: Push-based completion channels

## Goal

Replace per-cycle iteration of all active sessions with a single completion channel. When a session finishes, it pushes a result to the channel. The loop drains this channel at the start of each cycle — O(completions) instead of O(sessions).

## Changes

**`loop/types.go`** — Add `StageResult` type. Add `completions chan StageResult` to the Loop struct. The channel is buffered generously (e.g., 1024) and watcher goroutines use non-blocking send with a fallback mutex-guarded overflow slice. The overflow slice is **bounded** (`maxCompletionOverflow` in config) and instrumented with a saturation counter. Completion results are critical and cannot be dropped: if overflow is full, watcher goroutine falls back to context-aware blocking send on `completions` (`select { case completions <- r: ...; case <-ctx.Done(): ... }`). The loop goroutine is the sole consumer: it drains the channel, then locks and drains the overflow. Only the loop goroutine reads the overflow slice — no concurrent read/write.

**`loop/cook.go`** — `spawnCook()` starts a goroutine per dispatched session that blocks on `session.Done()`, then sends a `StageResult` to the completions channel. `collectCompleted()` becomes `drainCompletions()` — drains the channel then the overflow slice. **Dedup by generation token**: each dispatch assigns a monotonically increasing generation token (uint64 counter on the Loop struct). The `StageResult` carries this token. On receipt, the loop validates the token matches the currently registered handle for that order — if it doesn't (stale result from a killed session whose handle was already replaced by steer/respawn), the result is discarded. This is more robust than session-ID dedup because control paths (`steer`, `controlStop`) remove map entries before watcher goroutines finish, so a late result from a removed session must be safely ignorable regardless of ID uniqueness.

**Bootstrap session**: The existing `bootstrapInFlight` / `collectBootstrapCompletion()` path is migrated to the same watcher goroutine pattern. The bootstrap session gets a watcher goroutine that sends a `StageResult` on completion, same as any cook. Remove `collectBootstrapCompletion()` — bootstrap is no longer a special case for completion detection.

**Schedule session**: `spawnSchedule()` is migrated to use the same dispatch lifecycle as `spawnCook()`. The schedule session gets a watcher goroutine on `Done()`. Schedule completion uses the same `StageResult` channel with the schedule's real order ID (schedule is already a real order in `loop/schedule.go:14`). Add a `IsSchedule bool` flag to `StageResult` if the completion handler needs to distinguish schedule from cook results. No sentinel order IDs — that creates avoidable flow divergence.

**`loop/loop.go`** — `runCycleMaintenance()` calls `drainCompletions()` first. The cycle no longer needs to know how many sessions exist — it only processes events that arrived since last cycle. **Shutdown quiescence**: track watcher goroutines with a dedicated `sync.WaitGroup` (`watcherWG`) on the Loop struct. Each watcher goroutine calls `watcherWG.Add(1)` on spawn and `watcherWG.Done()` on exit. The drain-exit condition waits on `watcherWG` (via a done channel: goroutine that calls `watcherWG.Wait()` then closes a signal channel). This is safer than `len(activeCooksByOrder) == 0` because control paths (`controlStop`, `steer`) remove map entries before watcher goroutines finish — the WaitGroup tracks the actual goroutine lifecycle. After the WaitGroup signals all watchers exited, do one final drain pass. Note: merge-results channel drain (phase 7) will extend this condition later — keep the shutdown logic extensible but don't reference it here.

**Recovered sessions** (deferred to phase 3): When `Runtime.Recover()` lands in phase 3, it returns `RecoveredSession` with a `SessionHandle`. The loop must register a watcher goroutine for each recovered session, identical to the dispatch path. Without this, recovered sessions that complete after restart never emit a `StageResult`, leaving orders stuck `"active"` forever and blocking shutdown quiescence. In phase 2, the existing `collectAdoptedCompletions()` path continues to work for adopted sessions — it will be replaced when phase 3 introduces the Runtime interface.

**`loop/control.go`** — Control commands that kill active sessions do NOT separately emit `StageResult` entries. Instead, they call `session.Kill()` which closes `Done()`, and the existing watcher goroutine produces the `StageResult`. This eliminates the duplicate-event problem at the source. The watcher goroutine reads `session.Status()` after `Done()` closes to determine the result status. Note: `controlSkip` (`control.go:482`) currently only edits orders (no session kill) and `controlReject` (`control.go:300`) operates on pending-review state (no active session). Only `controlStop` (`control.go:589`) and `steer` (`cook.go:743`) actively kill sessions — those are the paths that rely on watcher-driven StageResult emission.

## Data structures

- `StageResult` — `OrderID string`, `StageIndex int`, `Attempt int`, `IsOnFailure bool`, `Status StageResultStatus` (typed constant: completed/failed/cancelled), `SessionID string`, `Generation uint64`, `IsSchedule bool`, `WorktreeName string`, `WorktreePath string`, `Error error`
- `StageResultStatus` — typed string constant (`StageResultCompleted`, `StageResultFailed`, `StageResultCancelled`)

## Routing

- Provider: `claude` | Model: `claude-opus-4-6` — concurrency design requires careful reasoning about channel semantics, dedup, and shutdown quiescence

## Verification

### Static
- `go test ./...` — all loop tests pass
- `go vet ./...` — no vet issues
- `collectCompleted()` no longer exists
- No iteration over `activeCooksByOrder` for completion detection
- Control commands do not write to the completions channel directly
- `watcherWG` is the sole quiescence mechanism — no `len(activeCooksByOrder) == 0` checks for shutdown

### Runtime
- Integration test: dispatch 3 sessions, complete them in arbitrary order, verify all 3 produce StageResults and stages advance correctly
- Race detector: `go test -race ./loop/...`
- Test: session that fails produces a StageResult with failed status
- Test: `controlStop` calls Kill(), watcher goroutine produces cancelled StageResult (no duplicate direct control emission)
- Test: shutdown waits for `watcherWG` to complete, then does final drain (deterministic — not timing-based)
- Test: burst of 50 completions in one cycle — overflow slice drains correctly, all processed
- Test: bootstrap session completes via watcher goroutine, no special-case collection
- Test: schedule session completes via watcher goroutine, result processed correctly
- Test: steer kills session A (generation N), respawns session B (generation N+1) for same stage — A's late result is ignored (stale generation), B's result is processed
- Test: `controlStop` removes map entry, watcher goroutine exits later — late StageResult with stale generation is discarded
- Test: overflow stress with tiny `completions` buffer + tiny `maxCompletionOverflow`, 200 concurrent completions — exact-once delivery, zero dropped results, no blocked goroutines after drain
- Goroutine leak test: start 10 sessions, complete all, verify `watcherWG` signals done and `runtime.NumGoroutine()` returns to baseline
