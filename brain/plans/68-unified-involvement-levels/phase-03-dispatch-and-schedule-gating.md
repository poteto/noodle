Back to [[plans/68-unified-involvement-levels/overview]]

# Phase 3: Dispatch and schedule gating

## Goal

Wire the new mode-derived gates for dispatch and schedule injection. Add a `dispatch` control command for manual-mode users.

## Changes

- **loop/loop.go**:
  - In `planCycleSpawns()`: early-return with no candidates when `!l.config.AutoDispatch()`. Log "dispatch suppressed (manual mode)".
  - Guard the two schedule injection points (~lines 455, 470) with `l.config.AutoSchedule()`. Log when suppressed.
- **loop/cook_retry.go**: Gate `processPendingRetries()` and `retryCook()` with `l.config.AutoDispatch()`. In manual mode, retries queue but don't auto-spawn.
- **loop/control.go**: Add `case "dispatch":` — runs one pass of dispatch logic ignoring the mode gate, spawns the first candidate (or targets `cmd.OrderID` if set). This is how manual-mode users trigger cooking.
- **server/server.go**: Add `"dispatch"` to `validActions`.

## Data Structures

No new types — `dispatch` uses existing `ControlCommand.OrderID`.

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — dispatch semantics need judgment

## Verification

### Static
- `go build ./...`
- `go vet ./...`

### Runtime
- Test: `Mode = ModeManual`, `planCycleSpawns()` returns no candidates
- Test: `Mode = ModeAuto`, `planCycleSpawns()` returns candidates normally
- Test: `Mode = ModeManual`, retries don't auto-spawn
- Test: `Mode = ModeManual`, no schedule orders injected when queue is empty
- Test: `dispatch` control command spawns one cook
- Test: `dispatch` with `order_id` targets specific order
- `go test ./loop/... ./server/...`
