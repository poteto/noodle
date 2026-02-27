Back to [[archive/plans/28-rename-prioritize-to-schedule/overview]]

# Phase 3: Rename loop functions and logic

## Goal

Rename all scheduling functions in the loop package and update dispatch/completion logic that references them. After this phase, the loop spawns and manages "schedule" sessions instead of "prioritize" sessions.

## Changes

### `loop/prioritize.go` (rename file to `loop/schedule.go`)

- `prioritizeQueueID` constant renamed to `scheduleQueueID`, value changed to `"schedule"`
- `isPrioritizeItem()` renamed to `isScheduleItem()`, compares against `scheduleQueueID`
- `bootstrapPrioritizeQueue()` renamed to `bootstrapScheduleQueue()`
- `prioritizeQueueItem()` renamed to `scheduleQueueItem()`, sets `item.ID` and `item.Skill` to `scheduleQueueID` / `"schedule"`, updates `item.Title` from "prioritizing tasks based on your backlog" to "scheduling tasks based on your backlog"
- `spawnPrioritize()` renamed to `spawnSchedule()`, updates `name` base to `scheduleQueueID`, default skill fallback from `"prioritize"` to `"schedule"`, calls `buildSchedulePrompt()`
- `buildPrioritizePrompt()` renamed to `buildSchedulePrompt()`
- `reprioritizeForChefPrompt()` renamed to `rescheduleForChefPrompt()`, calls `bootstrapScheduleQueue()`

### `loop/cook.go`

- `spawnCook()`: `isPrioritizeItem(item)` becomes `isScheduleItem(item)`, calls `l.spawnSchedule()`
- Completion handling: error message "prioritize failed after retries" becomes "schedule failed after retries"
- Chef steer handler: calls `l.rescheduleForChefPrompt()`

### `loop/reconcile.go`

- `prioritizePromptRegexp` (line 100) renamed to `schedulePromptRegexp`
- `prioritizeQueueID` reference (line 110) updated to `scheduleQueueID` (the constant is defined in `loop/prioritize.go` and renamed there — this file just uses it)

### `loop/loop.go`

- `bootstrapPrioritizeQueue()` call (line 280) becomes `bootstrapScheduleQueue()`
- Comments (lines 248-249): "prioritize session" / "prioritize writes" become "schedule session" / "schedule writes"

### `loop/queue.go`

- Comment (line 15): "Prioritize sessions write to queue-next.json" becomes "Schedule sessions write to queue-next.json"

### `tui/model.go`

- `steerTargets()`: `loop.PrioritizeTaskKey()` becomes `loop.ScheduleTaskKey()`

## Data structures

No struct changes. Only function renames, constant value changes, and one file rename.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical renames across 6 files, no judgment needed |

## Verification

### Static
```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

### Runtime
- Loop bootstrap creates a queue item with `ID: "schedule"` and `Skill: "schedule"`
- `isScheduleItem()` returns true for items with ID "schedule", false for "prioritize"
- Chef steer dispatches a schedule session via `rescheduleForChefPrompt()`
- TUI steer overlay lists "schedule" as first target
