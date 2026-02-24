Back to [[plans/45-task-editor-overhaul/overview]]

# Phase 3: Wire dropdowns to enum fields

## Goal

Replace left/right arrow cycling on Skill, Model, and Provider fields with the dropdown overlay component. Each enum field gets its own `Dropdown` instance. Enter/Space opens the dropdown when the field is focused; the overlay renders on top of the form.

## Changes

### `tui/task_editor.go`
- Add three `components.Dropdown` fields: `skillDropdown`, `modelDropdown`, `providerDropdown`
- Initialize dropdowns in `OpenNew`/`OpenEdit` with the correct items and selected index
- `HandleKey` changes:
  - When no dropdown is open and field is an enum field:
    - Enter/Space → open that field's dropdown (set `Cursor` to `Selected`, `Toggle()`)
    - Left/Right → still cycles (calls `cyclePrev`/`cycleNext` which now update `dropdown.Selected`)
  - When a dropdown is open:
    - Up/Down → `SelectPrev()`/`SelectNext()` on the active dropdown
    - Enter → `Confirm()` the selection, close dropdown
    - Esc → `Close()` the dropdown (don't close the whole form)
    - Any other key → close dropdown first, then handle normally
- Remove old `cyclePrev`/`cycleNext` switch statements — replace with `dropdown.Selected` index manipulation
- Remove hardcoded `models` and `providers` vars — pass as items to dropdowns (still static lists, but owned by dropdown instances)

### `tui/task_editor.go:Render`
- After rendering the form rows, if a dropdown is open, overlay its `Render()` output on top of the form body
- Position the overlay vertically aligned with the field that owns it (calculate row offset)
- Use `lipgloss.Place` or manual string overlay to position the dropdown

### Key routing priority
1. Dropdown open → route to dropdown (up/down/enter/esc)
2. No dropdown → route to form field navigation

## Data structures

No new types — uses `components.Dropdown` from phase 2.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Overlay positioning requires judgment, non-trivial rendering logic |

## Verification

### Static
- `go test ./tui/... && go vet ./...`
- Test: opening dropdown sets cursor to current selected value
- Test: confirming dropdown updates the selected value
- Test: Esc on open dropdown closes it without closing the form
- Test: Enter on a focused enum field opens the dropdown

### Runtime
- Launch TUI, press `n`, tab to Skill field
- Press Enter — dropdown overlay appears with all task types
- Up/Down navigates the list, Enter selects, Esc closes
- Left/Right still cycles without opening the dropdown
- Verify overlay renders on top of (not below) other form fields
