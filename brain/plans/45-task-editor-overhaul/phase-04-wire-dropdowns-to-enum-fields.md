Back to [[plans/45-task-editor-overhaul/overview]]

# Phase 4: Wire dropdowns to enum fields

## Goal

Replace left/right arrow cycling on Skill, Model, and Provider with the dropdown overlay. Each enum field uses a `Dropdown` instance as its single source of truth — the editor reads `dropdown.Selected` instead of maintaining a parallel index.

## Changes

### `tui/task_editor.go` — state
- Replace `skill int`, `model int`, `provider int` with three `components.Dropdown` fields: `skillDD`, `modelDD`, `providerDD`
- `OpenNew`: initialize each dropdown with items and Selected=0
- `OpenEdit`: initialize with items and Selected set to matching index (or synthetic entry for unknown skill)
- `Submit`: read `skillDD.Selected`, `modelDD.Selected`, `providerDD.Selected` to get values
- Helper: `activeDropdown() *components.Dropdown` — returns the dropdown for the current field, or nil if on Prompt

### `tui/task_editor.go:HandleKey` — key routing
- Add `dropdownOpen() bool` helper — true if any dropdown is open
- When a dropdown is open (highest priority):
  - Up/Down → `Prev()`/`Next()` on the active dropdown
  - Enter → `Confirm()`, return `ActionHandled`
  - Esc → `Close()`, return `ActionHandled` (not `ActionClose` — don't close the form)
  - Any other key → close dropdown first, then handle normally
- When on an enum field, dropdown closed:
  - Enter/Space → `Toggle()` on the active dropdown, return `ActionHandled`
  - Left/Right → `CyclePrev()`/`CycleNext()` (no overlay)
- Remove old `cyclePrev()`/`cycleNext()` methods
- Remove hardcoded `models` and `providers` vars — items passed to dropdown constructors

### `tui/task_editor.go:Render` — overlay
- Render the form rows as before
- If a dropdown is open, overlay its `Render()` output on top of the form body:
  - Calculate the vertical offset of the owning field (row index in the form)
  - Render the dropdown below the field row, overlaying subsequent rows
  - Pass available height (form height minus field row offset) so dropdown can self-clamp
- For small terminals (height < 10): limit MaxVisible to 3

### Key priority summary (after this phase)
1. Dropdown open → route to dropdown
2. Enum field focused → Enter/Space opens, Left/Right cycles
3. Prompt field focused → existing text input behavior
4. Ctrl+Enter → submit (not yet — added in Phase 5)
5. Esc with no dropdown open → close form

## Data structures

No new types — uses `components.Dropdown` from Phase 3.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Overlay positioning and key routing priority require judgment |

## Verification

### Static
- `go test ./tui/... && go vet ./...`
- Test: Enter on enum field opens dropdown (returns ActionHandled, not ActionSubmit)
- Test: Esc with dropdown open closes dropdown (returns ActionHandled, not ActionClose)
- Test: Esc with no dropdown open returns ActionClose
- Test: Confirm in dropdown updates Selected value
- Test: Submit reads correct values from all three dropdowns
- Test: OpenEdit sets dropdown Selected to correct index for known TaskKey
- Test: model-level key precedence — Esc with dropdown open does NOT close the editor

### Runtime
- Launch TUI, press `n`, tab to Skill field
- Enter opens overlay dropdown with all task types
- Up/Down navigates, Enter selects, Esc closes dropdown
- Left/Right cycles without opening dropdown
- Overlay renders on top of form fields below
- Narrow terminal: dropdown clamps to fewer visible items
