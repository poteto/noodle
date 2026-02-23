Back to [[plans/25-tui-revamp/overview]]

# Phase 6: Brain Tab with Glamour Markdown

## Goal

Implement Brain tab — recent knowledge activity grouped by agent, with glamour-rendered markdown preview on enter.

## Changes

### `tui/brain.go` — Brain tab implementation

Renders brain activity as a clean list grouped by agent. Each entry: tag (new/edit/delete), file path, one-line description. Two modes: list (default) and preview (after enter).

Key type: `BrainTab` with `items []BrainActivity`, selection index, preview mode flag.

`BrainActivity`: `Agent`, `At`, `Tag` (new/edit/delete), `FilePath`, `Description`.

### `tui/brain_preview.go` — Glamour markdown renderer

Wraps `charmbracelet/glamour` to render brain notes as styled terminal markdown. Constrains width to right pane. Dark style matching pastel palette.

Key function: `RenderMarkdown(content string, width int) (string, error)`.

### `tui/model_snapshot.go` — Brain activity tracking

Add `BrainActivity []BrainActivity` to Snapshot. Scan `brain/` for files modified since loop started, sorted by mtime descending. **Bounded scan**: track last-seen mtime, only stat files with mtime > last scan. Cap to 100 recent entries. Infer agent from session events mentioning brain file writes.

### `go.mod` — Add glamour dependency

`go get github.com/charmbracelet/glamour`

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`

## Verification

### Static
- `go test ./tui/...` passes
- Test: brain tab renders list with correct tags and grouping
- Test: glamour renders sample markdown without error
- Test: preview mode switches on enter, esc returns to list

### Runtime
- Brain tab shows recently modified notes grouped by source
- Tags colored: new=green, edit=blue, delete=coral
- Enter shows glamour-rendered preview
- Esc returns to list
- Empty state has placeholder
