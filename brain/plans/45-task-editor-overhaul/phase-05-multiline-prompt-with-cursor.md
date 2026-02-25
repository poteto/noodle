Back to [[plans/45-task-editor-overhaul/overview]]

# Phase 5: Multiline prompt textarea with cursor

## Goal

Replace the single-line character-by-character prompt input with a multiline textarea that shows a visible text cursor when focused. Enter inserts newlines in the prompt; Ctrl+Enter submits the form from any field. Includes viewport scrolling and vertical target-column tracking.

## Changes

### `tui/task_editor.go` — textarea state
- Replace `prompt string` with:
  - `promptLines []string` — lines of text (always at least one empty string)
  - `promptRow int` — cursor row (line index)
  - `promptCol int` — cursor column (rune index within line)
  - `promptTargetCol int` — desired column for vertical movement (set on horizontal moves, preserved on vertical)
  - `promptScrollOffset int` — first visible line when lines exceed max visible height
  - `maxPromptLines int` — max visible lines (e.g. 5), set based on available form height
- Helper methods:
  - `insertRune(r rune)` — insert at cursor, advance col, update targetCol
  - `insertNewline()` — split current line at cursor, move to next line
  - `deleteBack()` — backspace: if col > 0 delete rune; if col == 0 and row > 0, join with previous line
  - `promptText() string` — join lines with `\n` for submission
  - `moveCursorH(delta int)` — move cursor left/right with line wrapping, update targetCol
  - `moveCursorV(delta int) bool` — move cursor up/down. Returns false if at boundary (first/last line) — caller uses this for field escape in Phase 6. Clamp col to line length but preserve targetCol.
  - `ensureCursorVisible()` — adjust promptScrollOffset so promptRow is within [scrollOffset, scrollOffset+maxPromptLines)

### `tui/task_editor.go:HandleKey`
- When field is Prompt:
  - Enter → `insertNewline()`, return `ActionHandled`
  - Backspace → `deleteBack()`, return `ActionHandled`
  - Left/Right → `moveCursorH()`, return `ActionHandled`
  - Any text → `insertRune()`, return `ActionHandled`
- Ctrl+Enter → call `Submit()` and return `ActionSubmit` from ANY field (replaces plain Enter for form submission)
- When field is NOT Prompt and no dropdown open:
  - Enter → open dropdown (enum fields) — already handled in Phase 4
- Update footer hint: replace `enter: submit` with `ctrl+enter: submit`

### Ctrl+Enter portability
- Bubble Tea v2 with keyboard enhancements detects Ctrl+Enter as `KeyEnter` with `ModCtrl`. On terminals without keyboard enhancements, Ctrl+Enter may send the same sequence as Enter. Fallback: if keyboard enhancements are not available, Enter on a non-Prompt field still submits (existing behavior preserved).

### `tui/task_editor.go:Render`
- Prompt field renders visible lines (from scrollOffset to scrollOffset+maxPromptLines)
- When focused: render a block cursor character (`▎`) at cursor position via inverse-video style or inserted character
- When unfocused: no cursor, just text (or "(empty)" placeholder)
- Use `ansi.StringWidth()` for correct cursor positioning with multi-byte characters
- If lines exceed maxPromptLines, show a subtle `+N more` indicator

### `tui/task_editor.go:Submit`
- Use `promptText()` (joined lines) instead of raw string

## Data structures

No new exported types — cursor/viewport state is internal to TaskEditor.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Cursor positioning, viewport scrolling, and Ctrl+Enter portability require careful implementation |

## Verification

### Static
- `go test ./tui/... && go vet ./...`
- Test: Enter inserts newline, splits line at cursor position
- Test: Backspace at col 0 joins with previous line
- Test: Backspace at row 0, col 0 is a no-op
- Test: `moveCursorV` returns false at boundaries (row 0 up, last row down)
- Test: targetCol preserved across vertical movement through shorter lines
- Test: `ensureCursorVisible` adjusts scrollOffset correctly
- Test: `promptText()` returns correct multiline string with `\n` separators
- Test: Ctrl+Enter from Prompt field returns ActionSubmit
- Test: Ctrl+Enter from enum field returns ActionSubmit
- Test: Enter on Prompt returns ActionHandled (not ActionSubmit)
- Test: empty prompt lines → Submit returns nil

### Runtime
- Launch TUI, press `n` — prompt shows visible cursor when focused
- Type text, press Enter — new line, cursor moves down
- Backspace at line start joins lines
- Type more than 5 lines — viewport scrolls, cursor stays visible
- Ctrl+Enter submits from any field
- Tab away → cursor disappears. Tab back → cursor reappears at previous position.
