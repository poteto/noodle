Back to [[plans/45-task-editor-overhaul/overview]]

# Phase 5: Context-sensitive up/down navigation

## Goal

Add up/down arrow keys for moving between form fields, but make them context-sensitive: on enum fields (Skill, Model, Provider), up/down navigates between fields; on the multiline prompt, up/down moves the text cursor within the textarea. Update the footer to reflect all available keybindings.

## Changes

### `tui/task_editor.go:HandleKey`
- When no dropdown is open:
  - On enum fields (Skill, Model, Provider):
    - Up → previous field (`field--` with wrapping)
    - Down → next field (`field++` with wrapping)
  - On Prompt field:
    - Up → if cursor is on first line, move to previous field; otherwise move cursor up one line
    - Down → if cursor is on last line, move to next field; otherwise move cursor down one line
- Tab/Shift+Tab continue to work on all fields (unchanged)

### `tui/task_editor.go:Render` — footer
- Update footer hint to reflect all keybindings:
  - `tab/↑↓: fields · enter/space: open · ctrl+enter: submit · esc: cancel`
- Keep it concise — one line

### Edge case: cursor column preservation
- When moving up/down between prompt lines, preserve the desired column (clamp to line length if shorter)
- Standard text editor behavior: remember the "target column" when moving vertically

## Data structures

No new types.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Clear spec, mechanical key routing changes |

## Verification

### Static
- `go test ./tui/... && go vet ./...`
- Test: Down on Skill field → moves to Model field
- Test: Up on Model field → moves to Skill field
- Test: Down on last field wraps to Prompt (first field)
- Test: Up/Down in multiline prompt moves cursor between lines
- Test: Up on first line of prompt → moves to last field (Provider)
- Test: Down on last line of prompt → moves to Skill field

### Runtime
- Launch TUI, press `n`
- Use up/down arrows to navigate between fields on enum fields
- Type multiline text in prompt, use up/down to move cursor within text
- Press up on first line of prompt — focus moves to Provider (last field, wrapping)
- Verify footer shows updated keybinding hints
