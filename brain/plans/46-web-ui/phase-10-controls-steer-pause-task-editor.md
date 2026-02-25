Back to [[plans/46-web-ui/overview]]

# Phase 10: Controls — Steer, Pause, Task Editor

## Goal

Build the interactive controls: steer input, pause/resume toggle, and task editor modal. Parity with `tui/model.go` key handlers and `tui/task_editor.go`.

## Changes

- **`ui/src/components/SteerInput.tsx`** — Text input for steering agents. `@` triggers mention autocomplete showing active session IDs. Submits via `useSendControl({action: "steer", target, prompt})`. Accessible via a toolbar button or keyboard shortcut.
- **`ui/src/components/LoopControls.tsx`** — Pause/Resume button in the toolbar/header. Shows current loop state. Sends `{action: "pause"}` or `{action: "resume"}` via `useSendControl()`.
- **`ui/src/components/TaskEditor.tsx`** — Modal form for creating/editing queue items. Fields: prompt (textarea), type (select: execute/plan/review/reflect/prioritize), model (select), provider (select), skill (text input). New task sends `{action: "enqueue", ...}`. Edit sends `{action: "edit-item", item: id, ...}`.
- **`ui/src/components/Layout.tsx`** — Top-level layout with navigation tabs and toolbar (loop state, steer button, new task button). All views render inside this layout.
- **Keyboard shortcuts** — Optional: backtick for steer, `n` for new task, `p` for pause. Only when no input is focused.

## Data structures

- Steer: `{action: "steer", target: string, prompt: string}`
- Pause/Resume: `{action: "pause"}` / `{action: "resume"}`
- Enqueue: `{action: "enqueue", name: string, prompt: string, task_key: string, provider: string, model: string, skill: string}`
- Edit: `{action: "edit-item", item: string, prompt: string, ...}`

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — @-mention autocomplete, modal UX, keyboard shortcut handling require judgment. Invoke `frontend-design` skill.

## Verification

### Static
- `npm run typecheck` and `npm run build` pass

### Runtime
- Steer input: type `@` shows agent list, select agent, type message, submit — control command appears in `control.ndjson`
- Pause button: click toggles loop state, UI reflects change on next snapshot
- Task editor: create new task — appears in queue on next snapshot. Edit existing — changes reflected.
- Keyboard shortcuts work when no input is focused, don't interfere when typing
