Back to [[plans/70-scaling-the-loop-redesign/overview]]

# Phase 1: Subtract session maps and decouple order state

## Goal

Remove `activeByTarget` and `activeByID` maps from the Loop struct. Replace with a single `activeCooksByOrder` map keyed by order ID. Derive busy/active state from order stage statuses plus pending-retry state. Keep adopted-session recovery intact — it transitions to the new structure but is not removed.

## Changes

**`loop/types.go`** — Remove `activeByTarget map[string]*activeCook` and `activeByID map[string]*activeCook` from the Loop struct. Replace with `activeCooksByOrder map[string]*cookHandle`. Keep `adoptedTargets` and `adoptedSessions` intact for now (they transition to the Runtime interface in phase 3). Keep `pendingRetry` — it participates in busy-state derivation.

**`loop/cook.go`** — Rewrite `collectCompleted()` to iterate `activeCooksByOrder` (same O(n) for now — push-based replacement in phase 2). Rewrite `spawnCook()` to register in the single map instead of two. Keep `collectAdoptedCompletions()` working against the new structure — adopted sessions are tracked by order ID until phase 3 introduces `Runtime.Recover()`.

**`loop/loop.go`** — Update `planCycleSpawns()` to derive the busy set from: (a) stages with status `"active"` in orders, AND (b) `pendingRetry` map keys, AND (c) `activeCooksByOrder` keys (covers schedule sessions that don't mark stages `"active"`), AND (d) `adoptedTargets` keys (adopted sessions still block re-dispatch until phase 3 removes adoption). All four block re-dispatch for the same order. Capacity check (`loop.go:419`) must also include `adoptedTargets` (it already does — preserve this).

**`loop/queue.go`** — Update `stampStatus()` to derive counts from the new `activeCooksByOrder` map instead of `activeByTarget`.

**`loop/schedule.go`** — `spawnSchedule()` must mark the schedule order's stage `"active"` in orders before dispatching, same as `spawnCook()`. Without this, the busy-set derivation from stage statuses misses running schedule sessions, causing the same schedule order to dispatch every cycle (redispatch storm). This also means schedule sessions participate in `activeCooksByOrder` tracking.

**`loop/control.go`** — Migrate all `activeByTarget`/`activeByID` references: `controlEnqueue` busy check (`:427`), `controlSkip` iteration (`:477`), and `controlStop` kill + removal (`:589-601`). Each switches to `activeCooksByOrder` lookup/iteration.

**`loop/orders.go`** — Add `ActiveOrderIDs(orders OrdersFile) []string` and `BusyTargets(orders OrdersFile) map[string]bool` helper functions that derive state from the orders file directly. Semantics must match existing `dispatchableStages()` filtering (`:208`) — include failed/adopted/ticketed exclusion and on-failure stage handling, not just "status == active".

**Full old-map call-site inventory** (verify none remain after migration):
- `loop/types.go:218-219` — field definitions
- `loop/loop.go:80-81` — initialization
- `loop/loop.go:142` — shutdown kill iteration
- `loop/loop.go:209` — drain exit condition
- `loop/loop.go:364,389-390,419` — idle check, busy set, capacity
- `loop/cook.go:45,69,177-178,187,195-196,735,751,764-765` — capacity, iteration, register, collect, steer
- `loop/control.go:427,477,589-601` — enqueue check, skip, stop
- `loop/queue.go:13-14` — stampStatus
- `loop/schedule.go:100-101` — schedule register
- `loop/fixture_test.go:254-255` — test helper
- `loop/loop_test.go`, `loop/queue_audit_test.go`, `loop/bootstrap_test.go` — test assertions

**Internal sequencing**: (a) Define `cookHandle` type and `activeCooksByOrder` map; (b) inventory and migrate every call site above; (c) migrate `collectAdoptedCompletions()` and verify adoption; (d) migrate test fixtures and assertions; (e) clean up dead code from old maps.

## Data structures

- `cookHandle` — `orderID`, `stageIndex`, `stage Stage`, `isOnFailure`, `orderStatus`, `plan []string`, `attempt int`, `displayName string`, `done <-chan struct{}`, `worktreeName`, `worktreePath`, `session dispatcher.Session` — the `stage` field is required by retry logic (`cook.go:651`), pending review persistence (`pending_review.go:78,101`), and display name derivation
- `activeCooksByOrder map[string]*cookHandle` — single map replacing `activeByTarget` + `activeByID`

The `cookHandle` carries all fields from the current `activeCook` that retry logic (`cook.go:628,648`), pending review persistence (`pending_review.go:78`), and display name derivation need. Nothing is dropped.

## Routing

- Provider: `claude` | Model: `claude-opus-4-6` — judgment-heavy refactoring, need to understand the interplay between maps, adoption, and retry state

## Verification

### Static
- `go test ./...` — all existing loop tests pass
- `go vet ./...` — no vet issues
- No references to `activeByTarget` or `activeByID` remain in the codebase
- `adoptedTargets` still exists and adoption tests pass
- `pendingRetry` entries correctly block re-dispatch (regression test at `loop_test.go:1629`)

### Runtime
- Run `noodle start` with 2-3 orders, verify dispatch/completion/advancement works
- Integration test: multi-stage order dispatches first stage, completes, advances to next
- Test: schedule order dispatches once, stays busy until schedule session completes (no redispatch storm)
- Kill and restart loop — verify adopted-session recovery still works
