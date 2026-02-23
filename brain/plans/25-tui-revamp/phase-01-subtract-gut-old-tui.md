Back to [[plans/25-tui-revamp/overview]]

# Phase 1: Subtract — Gut Old TUI

## Goal

Delete the current TUI rendering code that will be replaced. Keep the foundation: Model struct, snapshot loading, control commands, steer parsing. Remove all surface-specific rendering so Phase 2 starts from a clean slate.

## Changes

### `tui/model_render.go` — Delete render functions

Delete `renderDashboard`, `renderSession`, `renderTrace`, `renderQueue`, `renderSteer`, `renderHelp`. Keep `renderSurface` as a stub that returns a placeholder string. The file will be rewritten in Phase 2.

### `tui/model.go` — Simplify surface model

Remove `surfaceSession`, `surfaceTrace`, `surfaceQueue` constants. Keep `surfaceDashboard` as the sole surface (will become tab container in Phase 2). Remove all surface-specific key handlers (`handleDashboardKey`, `handleSessionKey`, `handleTraceKey`, `handleQueueKey`). Keep `handleSteerKey` — steer logic is preserved and improved in Phase 8.

Remove stale state fields: `selectedActive`, `selectedQueue`, `traceFilter`, `traceFollow`, `traceOffset`, `sessionEventsFollow`, `sessionEventsOffset`. These will be replaced by per-tab state in later phases.

### `tui/styles.go` — Keep but mark for rewrite

Don't change yet — Phase 2 rewrites the palette. Just verify nothing references deleted functions.

### `tui/model_test.go` — Update tests

Remove tests for deleted surfaces. Keep snapshot loading tests and control command tests.

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`

## Verification

### Static
- `go build ./...` compiles
- `go vet ./...` passes
- `go test ./tui/...` passes (fewer tests, all green)

### Runtime
- `noodle start` launches TUI showing placeholder text (no crash)
- Ctrl+c exits cleanly
- Steer (`s`) still opens and submits commands
