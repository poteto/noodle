Back to [[plans/45-task-editor-overhaul/overview]]

# Phase 1: Restructure key routing

## Goal

Move Esc and Enter handling from `model.go` into `TaskEditor.HandleKey` so the editor controls key priority. This is a prerequisite for dropdowns (Esc closes dropdown, not modal) and multiline prompt (Enter inserts newline, not submit).

## Changes

### `tui/task_editor.go:HandleKey`
- Return a typed action instead of `(tea.Cmd, bool)`. New return type: `EditorAction` enum with values `ActionNone`, `ActionHandled`, `ActionSubmit`, `ActionClose`.
- Move Esc handling into HandleKey: `case tea.KeyEsc → return ActionClose`
- Move Enter handling into HandleKey: `case tea.KeyEnter → return ActionSubmit` (for now — phases 3-4 will add dropdown/newline branching)
- All other keys return `ActionHandled` or `ActionNone` as before

### `tui/model.go:handleKey` (Esc block, lines 247-251)
- Remove the `m.taskEditor.open` branch from the Esc handler. The editor now handles its own Esc.

### `tui/model.go:handleTaskEditorKey` (lines 468-481)
- Remove the Enter intercept. Instead, call `HandleKey` and switch on the returned action:
  - `ActionSubmit` → call `Submit()`, return cmd
  - `ActionClose` → call `Close()`, return nil
  - `ActionHandled` → return cmd
  - `ActionNone` → return nil

## Data structures

- `EditorAction int` — small enum constant type on TaskEditor

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical refactor, clear before/after |

## Verification

### Static
- `go test ./tui/... && go vet ./...`
- All existing task editor tests pass with updated return type
- Test: Esc returns `ActionClose`
- Test: Enter returns `ActionSubmit`
- Test: other keys return `ActionHandled` or `ActionNone`

### Runtime
- Launch TUI, press `n` — form opens. Esc closes. Enter submits. Identical behavior to before.
