Back to [[plans/97-adapter-schema-validator/overview]]

# Phase 4: Inject Warnings into Scheduler Prompt

## Goal

Give the scheduler visibility into adapter problems so it can create a fix task. Inject mise warnings into `buildSchedulePrompt()` following the existing pattern for `lastPromotionError` and `reconciledFailures`.

## Changes

### `loop/schedule.go`

- Add `miseWarnings []string` parameter to `buildSchedulePrompt()`
- When warnings are non-empty, append a section to the prompt:
  ```
  ADAPTER WARNINGS: The backlog sync produced validation warnings. Items with errors were skipped. Consider creating a task to fix the adapter:
  - <warning 1>
  - <warning 2>
  ```
- This follows the existing pattern: `lastPromotionError` appends `"PREVIOUS ORDERS REJECTED: ..."`, failures append `"Orders failed in a previous session..."`

### `loop/schedule.go` (`spawnSchedule`)

- Pass `l.lastMiseWarnings` (from phase 3) to `buildSchedulePrompt()`

### Tests

- `loop/schedule_test.go` (or add to existing) — test `buildSchedulePrompt` with and without warnings
- Verify prompt contains warning block when warnings present
- Verify prompt unchanged when warnings are empty

## Data structures

No new types. `buildSchedulePrompt` gains one parameter.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Follows existing pattern exactly, mechanical addition |

## Verification

### Static
- `go test ./loop/...` passes
- `go vet ./loop/...` clean

### Runtime
- Test: buildSchedulePrompt with warnings → prompt contains "ADAPTER WARNINGS" block
- Test: buildSchedulePrompt without warnings → prompt unchanged from current behavior
- Test: warnings include per-item details (line numbers, field names)
