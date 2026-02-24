---
id: 25
created: 2026-02-23
status: done
---

# TUI Revamp: 4B Command Center

## Context

The current TUI (`tui/`) is functional but basic — single-pane dashboard, string-builder rendering, no split layout, no tabs. The old Scrawl-era TUI (`~/code/scrawl/noodle/cook/tui/`) had 11k lines with rich interactivity but was tightly coupled to the old architecture.

This plan redesigns the TUI as a **command center**: persistent left rail (agents + stats) with a tabbed right pane (Feed, Queue, Brain, Config). The design was brainstormed interactively with the human, who chose this layout from 5 alternatives.

## Scope

**In scope:**
- Split layout: left rail + tabbed right pane
- Pastel color palette (yellow brand, warm palette, no purple)
- 4 tabs: Feed, Queue, Brain, Config
- Feed: NDJSON events rendered as bordered cards, steer messages visible as brand-yellow Chef cards
- Queue: proper `bubbles/table` with status lifecycle column
- Brain: knowledge activity feed with glamour markdown preview
- Config: autonomy dial (approve/review/full), routing, budget, controls
- Review flow: quality verdicts as actionable feed cards (non-full autonomy only)
- Reusable components: cards, pills/buttons, tab bar, autonomy slider
- Steer: backtick (`` ` ``) opens steer overlay from any tab, fix spacebar bug and mid-message @mentions
- Task editor: `n` creates new task, `e` edits selected queue item. Same overlay, all QueueItem fields (title, type, model, provider, skill, priority)
- `glamour` integration for rendering brain notes and plans in-TUI

**Out of scope:**
- Remote/cloud TUI (web interface)
- Session replay / time travel
- Metrics dashboards / charting
- Queue grab-and-reorder (future enhancement)
- Custom themes / user-configurable colors

## Constraints

- **Subtract first.** Phase 1 guts the old rendering code before building new. Per subtract-before-you-add.
- **Redesign, not bolt-on.** The current `tui/` package gets rewritten. Don't preserve the existing surface/overlay model — replace with the rail+tab architecture.
- **File-based state.** The TUI reads `.noodle/` files (queue.json, sessions/*/meta.json, events.ndjson). No new state protocols — everything the new tabs need must come from existing file state or natural extensions to it.
- **Components are dumb.** Per the bubbletea-tui skill: components expose methods + Render(width), main model owns message routing.
- **Width flows down.** Every render method takes `width int`. Parent calculates available width after borders/padding.
- **Cache rendered output.** Cards, table rows, and list items cache their rendered string keyed by width.
- **Use bubbles components.** `table` for Queue, `viewport` for Feed/Brain scrolling, `progress` for budget bar, `help` for keybindings, `spinner` for active agents. No tabs component exists in bubbles — build a simple tab bar.
- **Use glamour for markdown.** Brain tab previews render plans/notes with `charmbracelet/glamour`. Width-constrained to right pane.
- **Loop changes before UI.** The autonomy dial, merge/reject actions, and task creation all require loop-level changes (new control actions, pending-review state machine, `review.enabled` → `autonomy` migration). These loop changes are prerequisites baked into phases 7-9, not separate plan items — but they must land before the TUI features that depend on them.
- **Replace `review.enabled`, don't add alongside.** The current `config.ReviewConfig.Enabled` bool becomes the `autonomy` field. Migration: `enabled: true` → `"review"`, `enabled: false` → `"full"`. No dual source of truth.
- **Incremental snapshot ingestion.** The snapshot refresh (every 2s) must not re-read all session events from scratch. Feed uses cursor-based tailing (track last-read offset per session). Brain scans are bounded by mtime. Cap feed history to a reasonable window (e.g. last 500 events).

## Applicable Skills

- `bubbletea-tui` — component patterns, styling, rendering, state management
- `skill-creator` — if any new skills are needed (unlikely)

## Alternatives Considered

1. **Dashboard monitor** (old TUI style) — rejected: passive, doesn't show what agents are doing
2. **Chat/thread feed** (Slack-style) — rejected: loses at-a-glance density
3. **Kanban board** — rejected: doesn't show agent detail, limited horizontal space
4. **Focus mode** — rejected: loses multi-agent overview (but the Feed tab's "enter expand" achieves this)
5. **Minimal HUD** — rejected: no structure, everything flat

The human chose **4B: Command Center with Tabs** for the balance of overview (rail) + depth (tabs).

## Layout Sketches

### Feed Tab (default)

```
  noodle cooking                                          stable · $1.24

 ┌─ Agents ─────────┐   Feed   Queue   Brain   Config
 │                   │   ────
 │  ● quality-001    │
 │    opus · 2m14s   │   quality-cook-001                         just now
 │                   │  ┌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┐
 │  ● verify-002     │  ╎  ✓ Tests passing · 14/14                    ╎
 │    codex · 1m08s  │  ╎  Committed fix for session expiry nil check ╎
 │                   │  ╎  auth/session.go  +2 −1                     ╎
 │  ✓ reflect-003    │  └╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┘
 │    done · 42s     │
 │                   │   ★ Chef → quality-001                      30s ago
 │                   │  ┌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┐
 ├───────────────────┤  ╎  "focus on edge cases in the auth handler"  ╎
 │  3 active         │  └╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┘
 │  3 pending review │    steer cards render in brand yellow
 │  55 queued        │
 │  $4.82 / $50      │   Quality on execute-006 · arrow bindings    10s ago
 │  $0.97/hr         │  ┌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┐
 └───────────────────┘  ╎  APPROVE  "Clean fix. Types consistent     ╎
                        ╎  across the handler chain. Tests pass."    ╎
                        ╎                                             ╎
                        ╎  3 files  +12 −8   .worktrees/exec-types   ╎
                        ╎  ✓ 42/42 tests     $0.18                   ╎
                        ╎                                             ╎
                        ╎   ┌─ Merge ─┐  ┌─ Reject ─┐  ┌─ Diff ─┐  ╎
                        ╎   │    ✓    │  │    ✗     │  │   ◆    │  ╎
                        ╎   └─────────┘  └──────────┘  └────────┘  ╎
                        └╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┘
                         verdict buttons only in non-full autonomy

  j/k scroll · m merge · x reject · d diff · a merge all · ` steer
```

### Queue Tab

```
  noodle cooking                                          stable · $4.82

 ┌─ Agents ─────────┐   Feed  Queue   Brain   Config
 │                   │         ─────
 │  ● execute-007    │
 │    opus · 1m30s   │   3/62 cooked  ████░░░░░░░░░░░░░░░░ 5%
 │                   │
 │                   │  ┌────┬──────────────┬──────────────────┬───────────┐
 │                   │  │  # │ Type         │ Item             │ Status    │
 │                   │  ├────┼──────────────┼──────────────────┼───────────┤
 │                   │  │  1 │ Execute      │ Arrow bindings   │ reviewing │
 ├───────────────────┤  │  2 │ Execute      │ AI canvas        │ reviewing │
 │  1 active         │  │  3 │ Execute      │ AI-suggested r…  │ cooking   │
 │  3 pending review │  │  4 │ Execute      │ Canvas-native …  │ planned   │
 │  55 queued        │  │  5 │ Execute      │ One-click expo…  │ planned   │
 │  $4.82 / $50      │  │  6 │ Execute      │ First-time exp…  │ planned   │
 │  $0.97/hr         │  │  7 │ Plan         │ Error recovery   │ no plan   │
 └───────────────────┘  │  8 │ Execute      │ MCP Server (sc…  │ planned   │
                        │  9 │ Reflect      │ Session pattern…  │ ready     │
                        │ 10 │ Prioritize   │ Re-prioritize …  │ ready     │
                        └────┴──────────────┴──────────────────┴───────────┘
                                                              48 more ↓

  j/k navigate · enter detail · e edit · n new task · ` steer
```

### Brain Tab

```
  noodle cooking                                          stable · $1.24

 ┌─ Agents ─────────┐   Feed   Queue  Brain   Config
 │                   │                 ─────
 │  ● quality-001    │
 │    opus · 2m14s   │   12 notes  6 principles  3 plans
 │                   │
 │  ● verify-002     │
 │    codex · 1m08s  │   Today
 │                   │
 │  ✓ reflect-003    │   reflect-003 · 30s ago
 │    done · 42s     │
 │                   │     new    codebase/auth-session-types
 │                   │            Session types use string IDs throughout;
 ├───────────────────┤            handlers must not cast
 │  3 active         │
 │  59 queued        │     new    principles/validate-at-boundary
 │  $1.24 / $50      │            Only validate at system boundaries,
 │  $0.82/hr         │            trust internal code
 └───────────────────┘
                           edit   codebase/handler-type-safety
                                  Added int-vs-string gotcha


                         quality-cook-001 · 3m ago

                           edit   todos
                                  Marked #21 complete

  j/k scroll · enter preview (glamour markdown) · / search · ` steer
```

### Config Tab

```
  noodle cooking                                          stable · $1.24

 ┌─ Agents ─────────┐   Feed   Queue   Brain  Config
 │                   │                         ──────
 │  ● quality-001    │
 │    opus · 2m14s   │   Autonomy
 │                   │   ╌╌╌╌╌╌╌╌
 │  ● verify-002     │   Mode              review
 │    codex · 1m08s  │                     ◀── ──────●──────── ──▶
 │                   │                     approve  review  full
 │  ✓ reflect-003    │
 │    done · 42s     │
 │                   │   Routing
 │                   │   ╌╌╌╌╌╌╌
 ├───────────────────┤   Default model      claude-opus-4-6
 │  3 active         │   Provider            anthropic
 │  59 queued        │   Max agents          5
 │  $1.24 / $50      │
 │  $0.82/hr         │   Budget
 └───────────────────┘   ╌╌╌╌╌╌
                         Per-cook max        $5.00
                         Run budget          $50.00
                         Spent               $1.24
                         ████░░░░░░░░░░░░░░░░░░░░░░░░░░ 2.5%

                         ─────────────────────────────────────────────
                          ┌─ Pause ─┐   ┌─ Stop All ─┐   ┌─ Re-Q ─┐
                          │   ⏸     │   │     ■      │   │   ↻    │
                          └─────────┘   └────────────┘   └────────┘

  j/k navigate · enter edit · ←→ adjust dial · ` steer
```

### Steer Overlay (bottom drawer, from any tab)

```
  noodle cooking                                          stable · $1.24

 ┌─ Agents ─────────┐   Feed   Queue   Brain   Config
 │                   │   ────
 │  ● quality-001    │
 │    opus · 2m14s   │   quality-cook-001                         just now
 │                   │  ┌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┐
 │  ● verify-002     │  ╎  ✓ Tests passing · 14/14                    ╎
 │    codex · 1m08s  │  ╎  Committed fix for session expiry nil check ╎
 │                   │  └╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌┘
 │  ✓ reflect-003    │
 │    done · 42s     │
 │                   │    @quality-cook-001
 │                   │    @execute-007
 ├───────────────────┤    @everyone
 │  3 active         │
 │  59 queued        │  ─────────────────────────────────────────────
 │  $1.24 / $50      │  > focus on edge cases @quality-cook-001|
 │  $0.82/hr         │
 └───────────────────┘  enter send · esc close · @ mention anywhere

```

### Task Editor Overlay (bottom drawer — create or edit)

```
  noodle cooking                                          stable · $1.24

 ┌─ Agents ─────────┐   Feed  Queue   Brain   Config
 │                   │        ─────
 │  ● quality-001    │
 │    opus · 2m14s   │  ┌────┬──────────────┬──────────────────┬───────────┐
 │                   │  │  # │ Type         │ Item             │ Status    │
 │  ● verify-002     │  ├────┼──────────────┼──────────────────┼───────────┤
 │    codex · 1m08s  │  │▸ 1 │ Execute      │ Arrow bindings   │ reviewing │
 │                   │  │  2 │ Execute      │ AI canvas        │ reviewing │
 │  ✓ reflect-003    │  └────┴──────────────┴──────────────────┴───────────┘
 │    done · 42s     │
 │                   │  ─────────────────────────────────────────────
 │                   │   Edit Task #1                         (e to edit)
 ├───────────────────┤   ╌╌╌╌╌╌╌╌╌╌╌╌╌
 │  3 active         │   Title     Arrow key bindings for canvas|
 │  59 queued        │   Type      ◀ Execute  ▶
 │  $1.24 / $50      │   Model     ◀ claude-opus-4-6  ▶
 │  $0.82/hr         │   Provider  ◀ claude  ▶
 └───────────────────┘   Skill     execute
                         Priority  ◀ high  ▶

  tab cycle · ←→ adjust · enter save · esc cancel
```

## Color Palette

```
Brand:    #fde68a (pastel yellow)     Borders:  #fcd34d (deeper gold)
Surface:  #1c1c2e (navy-charcoal)     Card BG:  #24243a (lifted surface)
Success:  #86efac (pastel green)       Warning:  #fdba74 (pastel orange)
Error:    #fca5a5 (pastel coral)       Info:     #93c5fd (pastel blue)
Text:     #f5f5f5 (near-white)        Dim:      #6b6b80 (muted)
Secondary:#a8a8b8 (lavender-gray)
```

Task type colors: Execute=#86efac, Plan=#93c5fd, Quality=#fde68a, Reflect=#f9a8d4, Prioritize=#fdba74

## Autonomy Dial

Three modes control how the TUI behaves:

| Mode | Queue spawns? | Verdicts | Feed shows |
|------|--------------|----------|------------|
| `full` | Auto | Auto-merge on APPROVE | Verdicts (no buttons) |
| `review` | Auto | Blocks until human merges/rejects | Verdicts with merge/reject/diff buttons |
| `approve` | Blocks until human approves | Same as review | Verdicts + spawn request cards with approve/skip |

The queue never stops. In `review` mode, completed work sits in worktrees awaiting merge. In `approve`, existing cooks finish but new spawns wait.

## Keyboard Shortcuts

### Global (any tab)

| Key | Action |
|-----|--------|
| `1-4` | Switch tabs (Feed, Queue, Brain, Config) |
| `` ` `` | Open steer overlay (send instructions to agents) |
| `n` | Open task creator (add item to queue) |
| `p` | Pause / resume loop |
| `ctrl+d` | Detach (TUI exits, loop continues headless) |
| `ctrl+c` | First press: show "press again to quit". Second: quit |
| `?` | Toggle help overlay |

### Feed tab

| Key | Action |
|-----|--------|
| `j/k` | Scroll cards |
| `enter` | Expand card (full detail) |
| `f` | Filter by agent / event type |
| `m` | Merge (on verdict card, non-full only) |
| `x` | Reject (on verdict card, non-full only) |
| `d` | View diff (on verdict card) |
| `a` | Merge all approved verdicts |

### Queue tab

| Key | Action |
|-----|--------|
| `j/k` | Navigate rows |
| `enter` | Show item detail (plan phases, rationale) |
| `e` | Edit selected item (opens task editor prefilled) |
| `g` | Grab / reorder (future) |

### Brain tab

| Key | Action |
|-----|--------|
| `j/k` | Scroll entries |
| `enter` | Open glamour markdown preview |
| `esc` | Back to list (from preview) |
| `/` | Search brain notes |

### Config tab

| Key | Action |
|-----|--------|
| `j/k` | Navigate settings |
| `enter` | Edit selected value |
| `←/→` | Adjust autonomy dial |
| `esc` | Cancel edit |

### Steer overlay

| Key | Action |
|-----|--------|
| `@` | Open mention autocomplete (works anywhere in message) |
| `↑/↓` | Navigate mention list |
| `enter` | Select mention / submit instruction |
| `esc` | Close mentions → close steer |

### Task editor overlay (create & edit)

| Key | Action |
|-----|--------|
| `tab` | Cycle between fields (title, type, model, provider, skill, priority) |
| `←/→` | Cycle options on selector fields (type, model, provider, priority) |
| `enter` | Submit (create new / save edit) |
| `esc` | Cancel and close |

## Phases

1. [[archived_plans/25-tui-revamp/phase-01-subtract-gut-old-tui]]
2. [[archived_plans/25-tui-revamp/phase-02-scaffold-layout-framework-and-pastel-palette]]
3. [[archived_plans/25-tui-revamp/phase-03-reusable-components]]
4. [[archived_plans/25-tui-revamp/phase-04-feed-tab]]
5. [[archived_plans/25-tui-revamp/phase-05-queue-tab]]
6. [[archived_plans/25-tui-revamp/phase-06-brain-tab-with-glamour-markdown]]
7. [[archived_plans/25-tui-revamp/phase-07-config-tab-and-autonomy-dial]]
8. [[archived_plans/25-tui-revamp/phase-08-review-flow-and-verdict-actions]]
9. [[archived_plans/25-tui-revamp/phase-09-navigation-steer-and-polish]]

## Verification

- `go test ./tui/...` — all TUI tests pass
- `go vet ./...` — no issues
- `go build ./...` — compiles
- Visual verification: launch `noodle start` in a terminal, confirm layout renders correctly at 80-col and 120-col widths
- Each tab renders with test data (snapshot fixtures)
- Keyboard navigation works across all tabs
- Autonomy dial changes feed behavior in real-time
- Steer: @mention works mid-message, spacebar works after autocomplete, steered session includes resume context from event log
- Task editor: create submits new task to queue, edit prefills and saves changes
- Double ctrl+c quit confirmation works
