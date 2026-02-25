Back to [[plans/46-web-ui/overview]]

# Phase 1: Extract Snapshot Package

## Goal

Move snapshot-loading logic out of `tui/` into `internal/snapshot/` so both the TUI and the new HTTP server can consume the same data assembly code.

## Changes

- **New `internal/snapshot/` package** — Move `loadSnapshot()` and its helpers from `tui/model_snapshot.go`. The types it returns (`Snapshot`, `Session`, `QueueItem`, `EventLine`, `FeedEvent`) move here too. Add JSON struct tags to all exported types — these become the API contract.
- **`tui/model_snapshot.go`** — Delete the moved functions. Import from `internal/snapshot/`. The TUI's existing types become aliases or thin wrappers.
- **`tui/model.go`** — Update `snapshotMsg` and `refreshSnapshotCmd` to use the shared types.

Helper functions that are pure TUI rendering (`healthDot`, `statusIcon`, `modelLabel`, etc.) stay in `tui/`. Only the data-loading and data-assembly functions move.

## Data structures

- `snapshot.Snapshot` — same fields as current `tui.Snapshot`, with `json:"..."` tags
- `snapshot.Session`, `snapshot.QueueItem`, `snapshot.EventLine`, `snapshot.FeedEvent` — same, JSON-tagged

## Routing

Provider: `codex` | Model: `gpt-5.3-codex`

## Verification

### Static
- `go test ./...` — existing TUI snapshot tests still pass
- `go vet ./...` clean
- No import cycles between `internal/snapshot/` and `tui/`

### Runtime
- `noodle start` renders same data as before
