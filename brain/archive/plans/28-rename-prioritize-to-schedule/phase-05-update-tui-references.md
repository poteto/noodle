Back to [[archive/plans/28-rename-prioritize-to-schedule/overview]]

# Phase 5: Update TUI references

## Goal

Update all TUI color fields, type maps, string literals, and task type lists from "prioritize" to "schedule". After this phase, the TUI renders "schedule" tasks with the correct color and recognizes "schedule" in all lookup maps.

## Changes

### `tui/styles.go`

- `Theme.Prioritize` field renamed to `Theme.Schedule`
- `theme` var: `Prioritize: lipgloss.Color("#fdba74")` becomes `Schedule: lipgloss.Color("#fdba74")`

### `tui/components/theme.go`

- `Theme.Prioritize` field renamed to `Theme.Schedule`
- `DefaultTheme`: `Prioritize: lipgloss.Color("#fdba74")` becomes `Schedule: lipgloss.Color("#fdba74")`
- `ColorPool` comment: `"// orange (prioritize)"` becomes `"// orange (schedule)"`
- `TaskTypeColor()`: `case "prioritize"` becomes `case "schedule"`, returns `DefaultTheme.Schedule`

### `tui/task_editor.go`

- `taskTypes` slice: `"prioritize"` becomes `"schedule"`

### `tui/queue.go`

- `noPlanTaskTypes` map: key `"prioritize"` becomes `"schedule"`
- Empty queue message: `"A prioritize agent will fill it"` becomes `"A schedule agent will fill it"`

### `tui/model_snapshot.go`

- `inferTaskType()` known slice (line 674): `"prioritize"` becomes `"schedule"`

## Data structures

- `Theme` struct in both `tui/styles.go` and `tui/components/theme.go`: field `Prioritize` renamed to `Schedule`

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical field and string renames across 5 files |

## Verification

### Static
```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

### Runtime
- `TaskTypeColor("schedule")` returns the orange color (#fdba74)
- `TaskTypeColor("prioritize")` falls through to hash-based pool (no longer a named case)
- Task editor dropdown shows "schedule" instead of "prioritize"
- Queue tab shows "schedule" tasks with orange color badge
- Empty queue message reads "A schedule agent will fill it"
