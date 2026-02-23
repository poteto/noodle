Back to [[plans/25-tui-revamp/overview]]

# Phase 4: Queue Tab

## Goal

Implement Queue tab as a bordered table using `bubbles/table`. Shows prioritized backlog with lifecycle status.

## Changes

### `tui/queue.go` — Queue tab implementation

Renders queue as `bubbles/table` with columns: #, Type, Item, Status. Pastel palette for header and selection.

Key type: `QueueTab` wrapping `table.Model`. Methods: `SetQueue(items []QueueItem, activeIDs []string)`, `Render(width, height int) string`.

Status derivation:
- `cooking` — session ID in active sessions
- `reviewing` — completed, verdict exists, awaiting merge
- `planned` — has linked plan
- `no plan` — execute task without plan (warning color)
- `ready` — task types that don't need plans (Reflect, Prioritize, Quality)

### `tui/model.go` — Wire queue tab

Route snapshot to queue tab. Pass queue items and active session IDs for status derivation.

### `tui/model_snapshot.go` — Extend for queue status

Add `ActionNeeded []string` from `queue.json`'s `action_needed` field.

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`

## Verification

### Static
- `go test ./tui/...` passes
- Test: table renders with correct alignment at 80 and 120 cols
- Test: status column correct for each lifecycle state
- Test: `no plan` items render in warning color

### Runtime
- Queue table has proper borders and column headers
- j/k navigates rows with selection highlight
- Progress bar above shows X/Y cooked
- Type column color-coded, `no plan` visually distinct
