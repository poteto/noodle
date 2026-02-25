Back to [[plans/45-task-editor-overhaul/overview]]

# Phase 6: Context-sensitive up/down navigation

## Goal

Add up/down arrow keys for field navigation, context-sensitive to the active field: on enum fields, up/down moves between fields; on the multiline prompt, up/down moves the cursor within text and escapes to adjacent fields at boundaries. Update the footer.

## Changes

### `tui/task_editor.go:HandleKey`
- When no dropdown is open and field is an enum field (Skill, Model, Provider):
  - Up → previous field (wrapping)
  - Down → next field (wrapping)
- When no dropdown is open and field is Prompt:
  - Up → call `moveCursorV(-1)`. If it returns false (already on first line), move to previous field (wrapping to last field)
  - Down → call `moveCursorV(1)`. If it returns false (already on last line), move to next field
- Tab/Shift+Tab continue to work on all fields (unchanged)
- When dropdown IS open: up/down already handled by Phase 4 (dropdown navigation)

### `tui/task_editor.go:Render` — footer
- Update to: `tab/↑↓: fields · enter/space: open · ctrl+enter: submit · esc: cancel`

## Data structures

No new types.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Clear spec, mechanical key routing |

## Verification

### Static
- `go test ./tui/... && go vet ./...`
- Test: Down on Skill (field 1) → Model (field 2)
- Test: Up on Model (field 2) → Skill (field 1)
- Test: Down on Provider (last field) → wraps to Prompt (field 0)
- Test: Up on Prompt with single-line text → wraps to Provider (last field)
- Test: Down on Prompt with single-line text → moves to Skill (field 1)
- Test: Up/Down in multiline prompt moves cursor between lines (not between fields)
- Test: Up on first line of multiline prompt → moves to Provider (last field)
- Test: Down on last line of multiline prompt → moves to Skill (field 1)
- Test: Up/Down with dropdown open → does NOT move between fields (handled by dropdown)

### Runtime
- Launch TUI, press `n`
- Up/Down arrows navigate between fields on enum fields
- Type multiline text in prompt — up/down moves cursor within text
- Press up on first line of prompt → focus jumps to Provider
- Press down on last line of prompt → focus jumps to Skill
- Footer shows updated keybinding hints
