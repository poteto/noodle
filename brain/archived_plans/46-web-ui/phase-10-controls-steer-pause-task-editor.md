Back to [[archived_plans/46-web-ui/overview]]

# Phase 10: Controls — Pause, Task Editor, Keyboard Shortcuts

## Goal

Build the remaining interactive controls: pause/resume toggle, task editor modal, and keyboard shortcuts. Steer input lives in the chat panel (phase 9), not here.

## Changes

- **`ui/src/components/LoopControls.tsx`** — Pause/Resume button in the board header. Shows current loop state. Sends `{action: "pause"}` or `{action: "resume"}` via `useSendControl()`.
- **`ui/src/components/TaskEditor.tsx`** — Modal form for creating/editing queue items. Fields: prompt (textarea), type (select: execute/plan/review/reflect/schedule), model (select), provider (select), skill (text input). Defaults fetched from `GET /api/config`. New task sends `{action: "enqueue", ...}`. Edit sends `{action: "edit-item", item: id, ...}`.
- **Keyboard shortcuts** — `n` for new task, `p` for pause/resume. Only when no input is focused.

## Data structures

- Pause/Resume: `{action: "pause"}` / `{action: "resume"}`
- Enqueue: `{action: "enqueue", name: string, prompt: string, task_key: string, provider: string, model: string, skill: string}`
- Edit: `{action: "edit-item", item: string, prompt: string, ...}`
- Config defaults from `GET /api/config` for populating select options

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — invoke `frontend-design` and `interaction-design` skills.

## Verification

### Static
- `npm run typecheck` and `npm run build` pass

### Runtime
- Pause button toggles loop state, UI reflects change on next snapshot
- Task editor: create new task — appears in Queued column on next snapshot. Edit existing — changes reflected.
- Keyboard shortcuts work when no input is focused, don't interfere when typing
