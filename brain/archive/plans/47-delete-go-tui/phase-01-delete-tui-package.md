Back to [[archive/plans/47-delete-go-tui/overview]]

# Phase 1 — Delete tui/ package

## Goal

Remove all TUI source code from the repository.

## Changes

- Delete `tui/` directory entirely (all `.go` files including `tui/components/`)
- This is 26 files: 18 in `tui/` + 8 in `tui/components/`

The build will break after this phase — `cmd_start.go` imports `tui` and `bubbletea`. That's expected; phase 2 fixes it.

## Routing

| Provider | Model |
|----------|-------|
| `codex` | `gpt-5.3-codex` |

## Verification

### Static
- `rm -rf tui/` succeeds
- No other package outside `cmd_start.go` imports `tui` (already confirmed)

### Runtime
- Build will fail — expected, fixed in phase 2
