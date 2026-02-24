Back to [[plans/45-task-editor-overhaul/overview]]

# Phase 2: Dropdown overlay component

## Goal

Build a reusable dropdown component that renders as a floating overlay list on top of the form. This component doesn't handle its own key events — it exposes methods and the parent routes keys (per bubbletea-tui "components are dumb" pattern).

## Changes

### `tui/components/dropdown.go` (new file)
- `Dropdown` struct with: `Items []string`, `Selected int`, `Open bool`, `Cursor int`, `MaxVisible int`
- Methods:
  - `Toggle()` — open/close
  - `Close()`
  - `SelectNext()` / `SelectPrev()` — move highlight cursor, wrap around
  - `Confirm() int` — set selected to cursor position, close, return selected index
  - `Render(width int) string` — renders the overlay list with highlighted item
- Rendering:
  - Bordered box (lipgloss border, theme-colored)
  - Cursor item highlighted with brand color
  - Non-highlighted: secondary foreground
  - Scroll indicator if items exceed `MaxVisible`
  - Width sized to fit longest item + padding

### Styling
- Use theme colors from `components.DefaultTheme`
- Highlighted item: brand foreground, bold
- Border: dim/muted color

## Data structures

- `Dropdown{Items []string, Selected int, Open bool, Cursor int, MaxVisible int}` — `Selected` is the committed choice, `Cursor` is the highlighted item while browsing

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Single component, clear spec |

## Verification

### Static
- `go test ./tui/components/... && go vet ./...`
- Unit tests: Render output with various item counts, cursor positions, open/closed states
- Test wrapping: SelectNext on last item wraps to first

### Runtime
- Not wired to the form yet — verified via unit tests only in this phase
