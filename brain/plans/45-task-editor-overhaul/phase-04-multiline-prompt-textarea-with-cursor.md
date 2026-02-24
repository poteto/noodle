Back to [[plans/45-task-editor-overhaul/overview]]

# Phase 4: Multiline prompt textarea with cursor

## Goal

Replace the single-line character-by-character prompt input with a multiline textarea that shows a visible text cursor when focused. Enter inserts newlines in the prompt; Ctrl+Enter submits the form from any field.

## Changes

### `tui/task_editor.go` — textarea state
- Replace `prompt string` with a small textarea model:
  - `promptLines []string` — lines of text
  - `promptRow int` — cursor row
  - `promptCol int` — cursor column (rune index within current line)
- Helper methods:
  - `insertRune(r rune)` — insert at cursor, advance col
  - `insertNewline()` — split current line at cursor, move to next line
  - `deleteBack()` — backspace: if col > 0 delete char, if col == 0 join with previous line
  - `promptText() string` — join lines with `\n` for submission
  - `moveCursor(dr, dc int)` — move cursor with clamping

### `tui/task_editor.go:HandleKey`
- When field is Prompt:
  - Enter → `insertNewline()` (not submit)
  - Backspace → `deleteBack()`
  - Left/Right → move cursor within line
  - Home/End or Ctrl+A/Ctrl+E → beginning/end of line
  - Any text → `insertRune()`
- Ctrl+Enter → submit from ANY field (replaces plain Enter for submission)
- When field is NOT Prompt:
  - Enter → open dropdown (if enum field) or no-op
- Update footer hint: `ctrl+enter: submit`

### `tui/task_editor.go:Render`
- Prompt field renders all lines (up to a max visible height, e.g. 5 lines)
- When focused: render a block cursor (`▎` or inverse-video character) at the cursor position
- When unfocused: no cursor shown, just the text (or "(empty)" placeholder)
- Use `ansi.StringWidth()` for correct cursor positioning with multi-byte characters

### `tui/task_editor.go:Submit`
- Use `promptText()` (joined lines) instead of `e.prompt`

## Data structures

No new exported types — cursor state is internal to `TaskEditor`.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Cursor positioning and multiline text editing require careful implementation |

## Verification

### Static
- `go test ./tui/... && go vet ./...`
- Test: Enter inserts newline, Ctrl+Enter submits
- Test: Backspace at beginning of line joins with previous line
- Test: cursor position tracks correctly after insertions and deletions
- Test: `promptText()` returns correct multiline string

### Runtime
- Launch TUI, press `n` — prompt field shows blinking/visible cursor when focused
- Type text, press Enter — cursor moves to new line, text wraps
- Backspace at line start joins lines
- Ctrl+Enter submits the form with multiline prompt
- Tab away from prompt — cursor disappears
- Tab back — cursor reappears at previous position
