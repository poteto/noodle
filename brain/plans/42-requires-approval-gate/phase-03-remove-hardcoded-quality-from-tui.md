Back to [[plans/42-requires-approval-gate/overview]]

# Phase 3: Remove Hardcoded Quality from TUI

## Goal

Replace all hardcoded `"quality"` string literals in the TUI with `"review"` and remove the named `Quality` theme color. The `TaskTypeColor` function already has a hash-based fallback for unknown types, so removing named colors is safe.

## Changes

### `tui/components/theme.go`

- Rename `Theme.Quality` field to `Theme.Review`
- Update `DefaultTheme.Review` (keep same color `#fde68a`)
- Update `ColorPool` comment: `// yellow (quality)` -> `// yellow (review)`
- Update `TaskTypeColor` switch: `case "quality"` -> `case "review"`

### `tui/queue.go`

- `noPlanTaskTypes`: replace `"quality": {}` with `"review": {}`

### `tui/task_editor.go`

- `taskTypes` slice: replace `"quality"` with `"review"`

### `tui/model_snapshot.go`

- `inferTaskType` known list: replace `"quality"` with `"review"`

### `tui/queue_test.go`

- Update test data that references `"quality"` task key to `"review"`

### `tui/verdict.go`

- Update comments that reference "quality review" to "review"

## Routing

Provider: `codex` | Model: `gpt-5.3-codex` — mechanical string replacements.

## Verification

```sh
go test ./tui/...
# Confirm no remaining quality references:
rg '"quality"' tui/
```
