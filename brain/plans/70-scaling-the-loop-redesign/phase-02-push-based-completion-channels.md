Back to [[plans/70-scaling-the-loop-redesign/overview]]

# Phase 2: Push-based completion channels

## Goal

Replace per-cycle iteration of all active sessions with a single completion channel. When a session finishes, it pushes a result to the channel. The loop drains this channel at the start of each cycle — O(completions) instead of O(sessions).

## Changes

**`loop/types.go`** — Add `StageResult` type. Add `completions chan StageResult` to the Loop struct. Use an unbounded buffer (channel + overflow slice drained by the loop) to avoid blocking senders during burst completions. The bounded `maxConcurrency × 2` approach is fragile at scale — an unbounded collect-and-drain pattern is safer.

**`loop/cook.go`** — `spawnCook()` starts a goroutine per dispatched session that blocks on `session.Done()`, then sends a `StageResult` to the completions channel. `collectCompleted()` becomes `drainCompletions()` — a non-blocking drain of the channel that processes all available results. **Dedup by order ID + stage index**: if a result arrives for a stage already processed (e.g., `Kill()` closes `Done()` and control path also emits a result), skip the duplicate. Track processed stage keys in a set cleared each cycle.

**`loop/loop.go`** — `runCycleMaintenance()` calls `drainCompletions()` first. The cycle no longer needs to know how many sessions exist — it only processes events that arrived since last cycle. **Shutdown quiescence**: when draining, the loop must drain both the completions channel and the merge results channel (phase 7) to empty before exiting. The drain-exit condition becomes: `len(activeCooksByOrder) == 0 AND completions channel empty AND merge results channel empty`.

**`loop/control.go`** — Control commands that kill sessions (skip, reject) do NOT separately emit `StageResult` entries. Instead, they call `session.Kill()` which closes `Done()`, and the existing watcher goroutine produces the `StageResult`. This eliminates the duplicate-event problem at the source. The watcher goroutine reads `session.Status()` after `Done()` closes to determine the result status.

## Data structures

- `StageResult` — `OrderID string`, `StageIndex int`, `IsOnFailure bool`, `Status string` (completed/failed/cancelled), `SessionID string`, `WorktreeName string`, `WorktreePath string`, `Error error`

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
- Test: shutdown drains all pending completions before exiting
- Test: burst of 50 completions in one cycle — no blocked senders, all processed
