Back to [[plans/97-adapter-schema-validator/overview]]

# Phase 3: Inject Warnings into Scheduler Prompt

## Goal

Give the scheduler visibility into adapter problems so it can create a fix task. Inject mise warnings into `buildSchedulePrompt()` following the existing pattern for `lastPromotionError` and `reconciledFailures`.

## Changes

### `loop/schedule.go`

- Add `miseWarnings []string` parameter to `buildSchedulePrompt()`
- When warnings are non-empty, append a clearly delimited section to the prompt:
  ```
  ADAPTER WARNINGS: The backlog sync produced validation warnings. Items with errors were skipped. Consider creating a task to fix the adapter:
  - "line 3: invalid JSON: unexpected end of JSON input"
  - "line 5: missing required field title"
  ```
- **Prompt injection protection:**
  - Each warning line is quoted with `fmt.Sprintf("%q", warning)` (Go's %q escapes special characters)
  - Cap at **20 warnings** in the prompt — if more exist, append `"... and N more warnings"`. This bounds the prompt size since warnings originate from adapter-controlled parse failures.
  - The section uses the same append-to-`parts` pattern as `lastPromotionError` and `failures`
- This follows the existing pattern: `lastPromotionError` appends `"PREVIOUS ORDERS REJECTED: ..."`, failures append `"Orders failed in a previous session..."`

### `loop/schedule.go` (`spawnSchedule`)

- Pass `l.lastMiseWarnings` (from Phase 2) to `buildSchedulePrompt()`

### Tests

- `loop/schedule_test.go` — test `buildSchedulePrompt` with and without warnings
- Verify prompt contains `"ADAPTER WARNINGS"` block when warnings present
- Verify prompt unchanged when warnings are empty
- Verify warnings are `%q`-quoted in the output
- Verify cap at 20 warnings with overflow message

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
- Test: `buildSchedulePrompt` with warnings → prompt contains `"ADAPTER WARNINGS"` block
- Test: `buildSchedulePrompt` without warnings → prompt unchanged from current behavior
- Test: warnings include per-item details (line numbers, field names)
- Test: warnings with special characters are properly `%q`-escaped
- Test: 25 warnings → only first 20 in prompt, plus `"... and 5 more warnings"`
