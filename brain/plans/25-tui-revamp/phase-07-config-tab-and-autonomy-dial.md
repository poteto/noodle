Back to [[plans/25-tui-revamp/overview]]

# Phase 6: Config Tab and Autonomy Dial

## Goal

Implement Config tab with interactive autonomy dial — a visual 3-position slider between approve/review/full that changes system behavior in real-time.

## Changes

### `tui/config.go` — Config tab implementation

Renders sections: Autonomy (with dial), Routing, Budget, Adapters, Controls. Values read from config + runtime state.

Key type: `ConfigTab` with sections, selected control index, autonomy mode.

### `tui/components/dial.go` — Autonomy dial component

Visual 3-position slider: `approve ── review ── full` with marker on current position. Left/right moves marker. Active position in pastel yellow.

Key type: `Dial` with `positions []string`, `selected int`, `Render(width) string`.

### `tui/model.go` — Wire autonomy changes

When dial changes, write control command to `.noodle/control.ndjson` with new autonomy mode.

### `config/config.go` — Autonomy field

Ensure config has `Autonomy string` (values: "full", "review", "approve").

### Controls section

Pill button controls at bottom: Pause, Stop All, Re-Queue. Write control commands via existing `sendControlCmd`.

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`

## Verification

### Static
- `go test ./tui/...` passes
- Test: dial renders correct marker for each mode
- Test: left/right changes position
- Test: config renders values from config struct

### Runtime
- Dial renders as visual slider
- Arrow keys move between positions
- Autonomy change writes to control.ndjson
- Budget shows progress bar
- Adapters show found/missing
- Pause/Stop/Re-Q buttons work
