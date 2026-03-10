Back to [[archive/plans/28-rename-prioritize-to-schedule/overview]]

# Phase 1: Update constants and types

## Goal

Rename the canonical task key constants and type-level identifiers from "prioritize" to "schedule". After this phase, the authoritative string value and all functions that return or compare it use "schedule".

## Changes

### `loop/task_types.go`

- `PrioritizeTaskKey()` renamed to `ScheduleTaskKey()`, returns `"schedule"`
- `isPrioritizeTarget()` renamed to `isScheduleTarget()`, compares against `ScheduleTaskKey()`

### `internal/queuex/queue.go`

- `prioritizeTaskKey` constant renamed to `scheduleTaskKey`, value changed to `"schedule"`
- `isPrioritizeBootstrapItem()` renamed to `isScheduleBootstrapItem()`, compares against `scheduleTaskKey`

### `internal/taskreg/registry.go`

- Comment on `TaskType.Key` field: change example from `"prioritize"` to `"schedule"`
- Comment on `TaskType.Schedule` field: change "prioritize skill" to "schedule skill"

## Data structures

No struct changes. Only constant values and function names.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.4` | Mechanical find-and-replace across 3 files |

## Verification

### Static
```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

### Runtime
- `ScheduleTaskKey()` returns `"schedule"`
- `isScheduleTarget("schedule")` returns true, `isScheduleTarget("prioritize")` returns false
- `isScheduleBootstrapItem()` matches items with id/task_key/skill = "schedule"
