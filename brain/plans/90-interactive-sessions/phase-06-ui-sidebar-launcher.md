Back to [[plans/90-interactive-sessions/overview]]

# Phase 6 — UI Types + Sidebar + Launcher

## Goal

Update TypeScript types, show interactive sessions in the AGENTS sidebar, and add a launcher to start new interactive sessions.

## Changes

**`ui/src/client/types.ts`** — Add `chat` variant to `ControlCommand` discriminated union: `{ action: "chat"; prompt: string; name?: string }`. Add `kind?: "order" | "interactive"` and `correlation_id?: string` to `Session` type. No `provider`/`model` fields on the chat command for MVP — backend uses default steerable provider.

**`ui/src/client/generated-types.ts`** (if auto-generated via tygo) — Regenerate to pick up `Kind` on `Session`. If hand-maintained, edit directly.

**`ui/src/components/Sidebar.tsx`** — In the Agents section, after the Scheduler entry, render interactive sessions from snapshot. Filter `snapshot.sessions` (or `snapshot.active` + `snapshot.recent`) where `kind === "interactive"`. Each renders as an `agent-item` button with:
- Avatar showing first letter of session name (not the yellow "S" of scheduler)
- Display name and status
- Click navigates to `/actor/{sessionId}`
- "New Chat" button at the bottom of the agents list to launch

The "New Chat" button opens an inline launcher with:
- Initial prompt textarea
- Optional name field
- "Start" button that sends `{ action: "chat", prompt, name }`

No provider/model dropdown for MVP — only one steerable provider exists (Claude). Backend defaults handle provider/model selection. Add provider/model picker when a second steerable provider exists.

**Session discovery after spawn:** The `chat` ack returns a correlation ID (the control command ID). After sending, the UI watches subsequent snapshots for a session whose `correlation_id` matches the ack ID, then navigates to it. This is deterministic even when multiple chats launch concurrently.

**`ui/src/noodle.css`** — Style interactive agent items distinctly from scheduler:
- No yellow accent — use a different color or neutral treatment
- Smaller/subtler avatar
- "CHAT" or "INTERACTIVE" tag in the meta line

Invoke `frontend-design`, `interaction-design`, and `ts-best-practices` skills.

## Data Structures

- `ControlCommand | { action: "chat"; prompt: string; name?: string }` — discriminated union variant
- `Session & { kind?: "order" | "interactive"; correlation_id?: string }`

## Routing

- **Provider:** `claude`
- **Model:** `claude-opus-4-6`

## Verification

### Static
- `cd ui && pnpm tsc --noEmit` — type check passes
- No runtime errors on existing UI

### Runtime
- Verify `useSendControl()` accepts `{ action: "chat", prompt: "hello" }` without type error
- Launch Noodle, verify sidebar shows "New Chat" button
- Click "New Chat", enter prompt, start session — verify session appears in sidebar after snapshot update (matched via `correlation_id`)
- Click interactive session in sidebar — verify navigation to `/actor/{id}`
- Verify scheduler retains yellow accent styling, interactive sessions look different
- Verify two concurrent chat spawns navigate to the correct session (correlation ID matching)
