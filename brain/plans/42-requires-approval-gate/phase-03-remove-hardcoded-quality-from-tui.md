Back to [[plans/42-requires-approval-gate/overview]]

# Phase 3: Remove Hardcoded Quality from TUI

## Goal

Replace all hardcoded `"quality"` string literals in the TUI with `"review"`, remove the named `Quality` theme color, and delete verdict rendering entirely. Verdicts are a userland concept — noodle's TUI doesn't render them. The `TaskTypeColor` function already has a hash-based fallback for unknown types, so removing named colors is safe.

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

### Delete `tui/verdict.go`

Delete entirely: `Verdict` struct, `loadVerdicts`, `renderVerdictCard`. Verdicts are a userland concept — noodle's TUI no longer renders them. The approval flow (Phase 6) renders from `pendingReview` loop state, not verdict files.

### `tui/model_snapshot.go` — Remove verdict loading

Remove `Verdicts` field from the snapshot and the `loadVerdicts` call. Remove `ActionNeeded` if it's only used for verdict-based approval.

### `tui/feed.go` — Remove verdict card rendering

Remove verdict card rendering from the feed view. The feed shows session activity, not verdicts.

## Routing

Provider: `codex` | Model: `gpt-5.3-codex` — mechanical string replacements.

## Verification

```sh
go test ./tui/...
# Confirm no remaining quality references:
rg '"quality"' tui/
```
