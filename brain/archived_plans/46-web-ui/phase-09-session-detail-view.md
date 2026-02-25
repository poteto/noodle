Back to [[archived_plans/46-web-ui/overview]]

# Phase 9: Chat Detail Panel

## Goal

Build the Slack-style chat panel — a slide-out panel that shows an agent's event stream as a conversation thread with steer input. Design matches `ui_prototype/board.html` chat overlay.

## Changes

- **`ui/src/components/ChatPanel.tsx`** — Slide-out overlay panel (right side, 560px). Header with agent name, task, meta badges, context progress bar. Message area. Chat input at bottom.
- **`ui/src/components/ChatMessages.tsx`** — Scrollable message list rendering events as chat messages:
  - `think` → Agent message with avatar, sender name, timestamp, body text. Consecutive thinks collapse avatar.
  - `tool` → Compact inline card with icon, tool name, path.
  - `system` → Centered divider with label badge.
  - `cost` → Subtle mono-font note.
  - `steer` → Right-aligned user message with yellow bubble and hard shadow.
- **`ui/src/components/ChatInput.tsx`** — Auto-resizing textarea with send button. Enter sends, Shift+Enter for newline. @mention triggers popup.
- **`ui/src/components/MentionPopup.tsx`** — Dropdown listing other active agents. Shows avatar, name, type, task preview. Arrow keys navigate, Enter/Tab selects, Escape dismisses. Inserts `@AgentName ` into textarea.
- **Auto-scroll** — Scrolls to bottom on new events for active sessions. Manual scroll up disables auto-scroll. Scrolling back to bottom re-enables.
- **Open/close** — Clicking an agent card on the board opens the panel. Escape, backdrop click, or close button dismisses. Only one panel open at a time.

## Data structures

- Events from `useSessionEvents(id)` — fetches `GET /api/sessions/{id}/events`
- Steer sends `useSendControl({action: "steer", target: sessionId, prompt: text})`
- Mention candidates from `Snapshot.active` sessions (excluding current)
- `@AgentName` highlights rendered with a `<span class="mention-tag">` wrapper

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — invoke `frontend-design` and `interaction-design` skills. Chat UX, auto-scroll, @mention autocomplete all need judgment.

## Verification

### Static
- `npm run typecheck` and `npm run build` pass

### Runtime
- Clicking agent card opens chat panel with event stream
- Events render as typed messages (think, tool, system, cost, steer)
- Typing `@` shows mention popup with other agents
- Sending a steer message appends to event stream and sends control command
- Auto-scroll follows new events, pauses on manual scroll up
- Escape/backdrop closes panel
