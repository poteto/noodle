Back to [[plans/45-task-editor-overhaul/overview]]

# Phase 3: Dropdown overlay component

## Goal

Build a reusable dropdown component that renders as a floating overlay list. The dropdown owns the selection state — it is the single source of truth for the selected index. The parent reads from it, never maintains a parallel index.

## Changes

### `tui/components/dropdown.go` (new file)
- `Dropdown` struct:
  - `Items []string` — display labels
  - `Selected int` — committed selection index (source of truth)
  - `Open bool`
  - `Cursor int` — highlighted index while browsing (reset to Selected on open)
  - `MaxVisible int` — max items shown before scrolling (default 8)
  - `ScrollOffset int` — first visible item index for scrolling
- Methods:
  - `Toggle()` — if closed: set Cursor=Selected, ScrollOffset to show Cursor, open. If open: close.
  - `Close()` — close without changing Selected
  - `Next()` / `Prev()` — move Cursor with wrapping, adjust ScrollOffset to keep Cursor visible
  - `Confirm()` — set Selected=Cursor, close
  - `CycleNext()` / `CyclePrev()` — advance Selected directly (for left/right without opening)
  - `Render(width int) string` — bordered overlay with highlighted Cursor item
- Rendering:
  - Lipgloss border, theme-colored (dim)
  - Cursor item: brand foreground, bold
  - Non-cursor: secondary foreground
  - If items > MaxVisible: show scroll indicators (▲/▼) at top/bottom edges
  - Width: clamped to min(longestItem+4, availableWidth)

### Clamping rules
- If dropdown would extend below available height, clamp MaxVisible to fit
- The caller (TaskEditor.Render) passes available height so the dropdown can self-clamp

## Data structures

- `Dropdown` as above — no exported interfaces, just a struct with methods

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Single component, clear spec |

## Verification

### Static
- `go test ./tui/components/... && go vet ./...`
- Test: Toggle sets Cursor to Selected
- Test: Next/Prev wrap around, adjust ScrollOffset
- Test: Confirm sets Selected=Cursor and closes
- Test: CycleNext/CyclePrev advance Selected without opening
- Test: Render with items > MaxVisible shows scroll indicators
- Test: Render with 0 items returns empty string (no panic)
- Test: width clamping — dropdown never exceeds passed width

### Runtime
- Not wired to the form yet — verified via unit tests only
