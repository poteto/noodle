Back to [[plans/25-tui-revamp/overview]]

# Phase 9: Navigation, Steer, Task Creator, and Polish

## Goal

Wire all keyboard navigation, fix and improve steer, add task creation UI, implement double ctrl+c quit, and polish visuals (shimmer, empty states, responsive sizing).

## Changes

### `tui/model.go` — Comprehensive key handling

Tab-aware routing:
- Global: `1-4` tabs, `` ` `` steer, `n` task creator, `p` pause/resume, `ctrl+d` detach, `?` help
- Double `ctrl+c`: first press sets `quitPending = true` + shows "press ctrl+c again to quit" message (dimmed, in the keybar area). Second press within 2 seconds calls `tea.Quit`. Timer resets `quitPending` after 2s.
- Feed: `j/k` scroll, `enter` expand, `f` filter, `m` merge, `x` reject, `d` diff, `a` merge-all
- Queue: `j/k` navigate, `enter` detail
- Brain: `j/k` scroll, `enter` preview, `esc` back, `/` search
- Config: `j/k` navigate, `enter` edit, `←/→` dial, `esc` cancel

### `tui/steer.go` — Rewrite steer overlay

Fix two bugs and improve UX:

**Bug 1: Spacebar stops working after autocomplete.** Root cause: Bubble Tea sends space as `tea.KeySpace` (not `tea.KeyRunes`). The current `handleSteerKey` only handles `tea.KeyRunes` for character input. Fix: add `tea.KeySpace` case that appends a space to `steerInput`.

**Bug 2: @mention only works at message start.** This is actually not a parse bug — `mentionQuery` walks backward from cursor and finds any `@` preceded by a space. The issue is that after autocompleting the first mention, `refreshSteerMentions` doesn't re-trigger when typing a new `@` later. Fix: ensure `refreshSteerMentions` scans from the current cursor position, not just the end of input.

**Shortcut change:** Steer opens with `` ` `` (backtick) instead of `s`. Available from every tab and every view. Steer is the primary way to interact with agents — it should be instant.

**Bug 3: Steered sessions lose context.** The kill+respawn flow in `loop/cook.go:steer()` works mechanically (kills session, respawns with steering prompt, re-injects skills via dispatcher). But it doesn't extract resume context from the killed session's event log. The design doc (Plan 1, phase 13) specifies: extract files changed, last action, and progress from `.noodle/sessions/{id}/events.ndjson` before respawning. Without this, the steered session starts from scratch — it looks like steering did nothing because the new session redoes all the work.

Fix: add `buildResumeContext(runtimeDir, sessionID)` in `loop/cook.go` that reads the session's events.ndjson, extracts a progress summary (files touched, last tool calls, ticket progress), and prepends it to the steering prompt. The respawned session gets: resume context + "Chef steering: {prompt}" + original skill/system prompts.

**Steer visibility in Feed:** When a steer command is submitted, it appears in the Feed tab as a card in brand yellow (#fde68a) border. Shows `★ Chef → {agent}` with the steering message in quotes. This makes human interventions visible in the activity stream — the feed tells the complete story.

**UI polish:** Show the steer overlay as a bottom drawer (not full modal). The active tab content stays visible above, steer input renders below with mention dropdown above the input line.

### `tui/task_creator.go` — New file: inline task creator

Opens with `n` from any tab. A bottom drawer overlay (like steer) with fields:
- **Title** — text input for the task description
- **Type** — cycle through task types (Execute, Plan, Quality, Reflect, etc.)
- **Priority** — optional hint for the prioritize agent (high/normal/low)

On submit: writes a new item to the backlog via the backlog adapter (same mechanism the prioritize agent uses). The sous chef picks it up on the next cycle.

Key type: `TaskCreator` struct with `bubbles/textinput` for title, selected type, selected priority. Methods: `Open()`, `Close()`, `Submit() tea.Cmd`, `Render(width) string`.

Tab cycles between fields. Enter submits. Esc cancels.

### `loop/cook.go` — Steer resume context (non-TUI change)

Add `buildResumeContext(runtimeDir string, sessionID string) string` that reads `.noodle/sessions/{sessionID}/events.ndjson` and extracts:
- Files read/edited (from tool_use events with label Read/Edit/Write)
- Last 3-5 actions (from recent events)
- Ticket progress (from ticket_progress/ticket_done events)
- Approximate progress ("was 60% through, had committed 2 files")

Update `steer()` to call this before killing the session, then prepend to steering prompt. The full prompt becomes: `"Resume context: {summary}\n\nChef steering: {prompt}"`.

Also add test in `loop/loop_test.go`: `TestSteerCookSessionIncludesResumeContext` — write fixture events, steer the cook, verify the respawned prompt contains resume context.

### `tui/help.go` — Context-sensitive help

Replace static help with overlay showing keybindings for current tab. Use `bubbles/help` with the tab's key map.

### `tui/rail.go` — Shimmer animation

"noodle cooking" title shimmer from old TUI. Character-by-character yellow shimmer when agents active. `tea.Tick` at 200ms.

### `tui/styles.go` — Polish

- Empty states: "(no activity)" in dim, centered
- Responsive: terminal < 80 cols → rail collapses to icons-only (dots + cost)
- Clean truncation with ellipsis
- Quit confirmation message style: dim text replacing the keybar momentarily

### `tui/model_test.go` — Updated tests

- Double ctrl+c: first sets pending, second quits
- Tab switching preserves scroll position
- Steer: spacebar works after autocomplete
- Steer: @mention works mid-message (e.g., "focus on tests @cook-001 and keep it clean")
- Steer: opens from any tab with backtick
- Task creator: opens, tab cycles fields, enter submits, esc cancels
- Help shows correct keys per tab
- Rail renders at narrow widths

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`

## Verification

### Static
- `go test ./tui/...` all pass
- `go vet ./...` clean
- `sh scripts/lint-arch.sh` passes

### Runtime
- Full keyboard walkthrough: tabs, scroll, expand, steer, task creator, help, quit
- Steer: type `` ` ``, type "@cook" → autocomplete appears, select, press space → space works, type more text with another @mention → autocomplete triggers again
- Task creator: press `n`, type title, tab to type, tab to priority, enter submits, verify item appears in queue on next refresh
- Double ctrl+c: first press shows message, second quits. Wait 2s, need to press twice again
- Shimmer visible when agents active
- Narrow (< 80): rail collapses
- Wide (120+): right pane uses space
- No artifacts on resize
