Back to [[archive/plans/28-rename-prioritize-to-schedule/overview]]

# Phase 6: Update tests and test data

## Goal

Update all test files and fixture directories so "prioritize" references become "schedule". After this phase, all tests pass with the new naming and no test code or fixture mentions "prioritize".

## Changes

### Test files (string literals and assertions)

15 test files need "prioritize" string literals updated to "schedule":

- **`loop/task_types_test.go`** — assertions on `ScheduleTaskKey()` return value and `isScheduleTarget()`
- **`loop/loop_test.go`** — test case names and fixture references containing "prioritize"
- **`loop/queue_test.go`** — queue item IDs, skill names, task_key values
- **`loop/sous_chef_test.go`** — steer target assertions
- **`config/config_test.go`** — TOML snippets with `[prioritize]` section, field assertions on `Config.Schedule`
- **`internal/queuex/queue_test.go`** — queue items with id/skill/task_key "prioritize", `isScheduleBootstrapItem()` assertions
- **`internal/taskreg/registry_test.go`** — task type key literals
- **`tui/queue_test.go`** — `noPlanTaskTypes` map assertions, queue item fixtures
- **`tui/model_test.go`** — steer target assertions
- **`tui/model_snapshot_test.go`** — snapshot fixtures with "prioritize" queue items
- **`skill/resolver_test.go`** — skill name references
- **`dispatcher/tmux_session_test.go`** — session name fixtures
- **`generate/skill_noodle_test.go`** — generated doc assertions with "prioritize" field names
- **`parse/parse_test.go`** — parse fixture references
- **`tui/components/components_test.go`** — `TestBadgeTaskType` iteration: `"Prioritize"` becomes `"Schedule"`

### Testdata directory renames

4 directories under `loop/testdata/` need renaming:

- `empty-state-should-schedule-prioritize` to `empty-state-should-schedule`
- `prioritize-exited-without-complete-should-fail` to `schedule-exited-without-complete-should-fail`
- `prioritize-failed-retries-should-surface-session-reason` to `schedule-failed-retries-should-surface-session-reason`
- `prioritize-failed-retries-should-surface-stderr-reason` to `schedule-failed-retries-should-surface-stderr-reason`

Within these directories, any fixture JSON files containing "prioritize" as queue item IDs, skill names, or session names must be updated to "schedule".

### Testdata fixture content

Inside the renamed directories, update:
- Queue JSON files: `"id": "prioritize"` becomes `"id": "schedule"`, `"skill": "prioritize"` becomes `"skill": "schedule"`
- Session directory names: `prioritize-id` becomes `schedule-id` (under `state-*/. noodle/sessions/`)
- Session meta.json: any name or session ID references

## Data structures

No new types. Test assertions update to match renamed constants, types, and functions from phases 1-5.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical find-and-replace across test files and fixtures |

## Verification

### Static
```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

### Runtime
- All 15 test files pass
- `grep -r 'prioritize' loop/testdata/` returns zero hits
- `grep -r 'prioritize' --include='*_test.go' .` scoped to source returns zero hits
