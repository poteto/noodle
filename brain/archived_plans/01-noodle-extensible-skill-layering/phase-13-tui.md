Back to [[archived_plans/01-noodle-extensible-skill-layering/overview]]

# Phase 13 — TUI

## Goal

Build the Bubble Tea terminal UI for monitoring and interacting with the scheduling loop. The TUI is the primary human interface — it shows what's cooking, what's queued, costs, and lets the chef intervene.

Use the `bubbletea-tui` skill when implementing this phase.

## Implementation Notes

This phase will be implemented by Codex. Keep it simple:

- **No over-engineering.** Build exactly the views described. No extra abstractions, no premature component generalization, no "just in case" features. Bubble Tea code should be straightforward — flat model, direct rendering.
- **No backwards compatibility.** This is a greenfield build — there's nothing to be backwards-compatible with.
- **No extreme concurrency patterns.** Use straightforward goroutines and mutexes where needed. No complex channel orchestration or worker pools unless the phase explicitly calls for them.
- **Tests should increase confidence, not coverage.** Test the critical flows (model updates, view transitions, health dot derivation). Skip testing trivial rendering details. Ask: "does this test help us iterate faster?"
- **Prefer markdown fixture tests for parser/protocol flows.** Keep input and expected output in the same anonymized `*.fixture.md` file under `testdata/`, and use this pattern wherever practical.
- **Runtime verification in a fake project is required.** After static checks pass, run Noodle in a fresh temporary git repo under `/tmp` and verify the phase behavior there before closing.


- **Overview + phase completeness check (required).** Before closing the phase, review your changes against both the plan overview and this phase document. Verify every detail in Goal, Changes, Data Structures, and Verification is satisfied; track and resolve any mismatch before considering the phase done.
- **Worktree discipline (required).** Execute each phase in its own dedicated git worktree.
- **Commit cadence (required).** Commit as much as possible during the phase: at least one commit per phase, and preferably multiple commits for distinct logical changes.
- **Main-branch finalize step (required).** After all verification and review checks pass, merge the phase worktree to `main` and make sure the final verified state is committed on `main`.

## Changes

- **`tui/`** — New package. Bubble Tea application with the following views:
  - **Dashboard** — Active cooks with health dots, queue preview, stats, recent history. Default view.
  - **Session detail** — Session metadata, worktree, tickets, recent events. Entered via `enter` on a cook.
  - **Trace view** — Full scrollable event log for a session with filtering. Entered via `t` from session detail.
  - **Queue view** — Current prioritized queue from sous chef, with routing annotations.
- **`cmd_tui.go`** — `noodle tui` command (or make TUI the default when `noodle start` runs in a terminal).
- **`command_catalog.go`** — Register TUI command.

## Visual Reference

### Event type naming convention

All events in the TUI use consistent capitalized labels:

| Label | Source | Example |
|-------|--------|---------|
| `Read` | Tool use | `src/auth/token.ts` |
| `Edit` | Tool use | `src/auth/token.ts — add expiry check` |
| `Bash` | Tool use | `pnpm test -- auth` |
| `Glob` | Tool use | `src/**/*.ts` |
| `Grep` | Tool use | `refreshToken` |
| `Think` | Assistant text | Summary of reasoning |
| `Cost` | API round-trip | `$0.09 · 1.8k in / 0.5k out` |
| `Ticket` | Coordination | `claim #42`, `progress #42`, `done #42` |

### Health dots

Each active cook displays a colored health dot:

- **Green ●** — running, making progress, within budget
- **Yellow ●** — running but concerning: >80% context window, or idle for more than half the stuck threshold
- **Red ●** — failed, retrying, or stuck (no activity past threshold)

### Dashboard (default view)

```
╭─ noodle ─────────────────────────────────────────────────────────────────╮
│  ▸ cooking · 2 active · queue depth 4                                    │
│    $18.42 total · 3 slots available                                      │
│                                                                          │
│  ── Active Cooks ──────────────────────────────────────────────────────  │
│  Updated  Cook                    Model              Now              ●   │
│  2s ago   fix-auth-bug            claude-opus-4.6    Edit token.ts    ●   │
│  5s ago   add-user-tests          gpt-5.3-codex      Bash pnpm test  ●   │
│                                                                          │
│  ── Recent ────────────────────────────────────────────────────────────  │
│  ✓  refactor-middleware  claude-sonnet-4.6   8m 02s    $1.45   ab3f     │
│  ✓  update-readme        gpt-5.3-codex       2m 11s    $0.00   c7d2     │
│  ✗  fix-css-layout       claude-sonnet-4.6   6m 44s    $2.10   9b1f     │
│                                                                          │
│  ── Up Next ───────────────────────────────────────────────────────────  │
│  1. add-pagination        claude-sonnet    review                        │
│  2. fix-mobile-nav        codex            review                        │
│  3. update-api-docs       codex            no review                     │
│  4. refactor-db-queries   claude-opus      review                        │
╰─ enter inspect · q queue · s steer · p pause · ? keys · ctrl+c quit ───╯
```

### Session Detail (entered via `enter` on a cook)

```
◀ back                    Session Detail                   fix-auth-bug

─── Session ──────────────────────────────────────────────────────────
  Provider   claude                         Model     opus-4.6
  Status     running ●                      Duration  12m 34s
  Cost       $3.20                          Retries   0 / 3

─── Worktree ─────────────────────────────────────────────────────────
  Branch   cook/fix-auth-bug

  a3f7b21  fix: resolve auth token refresh race condition
  c8d2e14  test: add integration test for token expiry

─── Tickets ──────────────────────────────────────────────────────────
  Claimed    #42 auth token refresh         2m ago
  Files      src/auth/middleware.ts, src/auth/token.ts

─── Recent Events ────────────────────────────────────────────────────
  18:30:05  Edit    │ src/auth/token.ts — add expiry check
  18:30:12  Bash    │ pnpm test -- auth
  18:30:15  Think   │ Tests pass. Now I'll update the integration
                    │ test to cover the race condition scenario.
  18:30:22  Edit    │ src/auth/__tests__/token.test.ts
  18:30:30  Bash    │ pnpm verify
  18:30:33  Cost    │ $0.09 · 1.8k in / 0.5k out
  18:30:40  Ticket  │ progress — #42 auth token refresh

t trace · k kill · esc back · ? keys
```

### Trace View (entered via `t` from session detail)

Full scrollable event log with filtering. Auto-scrolls to bottom by default.

```
◀ back          Trace · fix-auth-bug          filter: all ▼

  18:29:30  Think   │ I'll start by reading the auth middleware to
                    │ understand the current token refresh flow and
                    │ identify where the race condition occurs.

  18:29:35  Read    │ src/auth/middleware.ts
  18:29:38  Read    │ src/auth/token.ts
  18:29:40  Read    │ src/auth/types.ts
  18:29:41  Cost    │ $0.15 · 3.2k in / 0.8k out

  18:29:45  Think   │ The refresh logic has a race condition: two
                    │ concurrent requests can both trigger a refresh
                    │ simultaneously. I need to add a deduplication
                    │ guard using a promise cache.

  18:29:52  Edit    │ src/auth/middleware.ts — extract refreshToken()
  18:29:58  Cost    │ $0.08 · 1.2k in / 0.4k out
  18:30:05  Edit    │ src/auth/token.ts — add expiry check
  18:30:12  Bash    │ pnpm test -- auth

  18:30:15  Think   │ Tests pass. Now I'll update the integration test
                    │ to cover the race condition scenario where two
                    │ requests hit the refresh endpoint concurrently.

  18:30:18  Cost    │ $0.12 · 2.1k in / 0.6k out
  18:30:22  Edit    │ src/auth/__tests__/token.test.ts
  18:30:25  Bash    │ pnpm test -- auth/integration
  18:30:28  Ticket  │ progress — #42 auth token refresh
  18:30:30  Bash    │ pnpm verify
  18:30:33  Cost    │ $0.09 · 1.8k in / 0.5k out
  18:30:45  Bash    │ git commit -m "fix: resolve auth token refresh
                    │ race condition"

                                                       ▼ auto-scroll

f filter · G scroll bottom · esc back · ? help
```

Filter modes (cycled with `f`):
- `all` — everything (default)
- `tools` — tool events only (Read, Edit, Bash, Glob, Grep)
- `think` — assistant reasoning only
- `ticket` — ticket events only (claim, progress, done)

### Steer (via `s` from any view)

Steering lets the chef talk to any actor in the kitchen from anywhere in the TUI. Pressing `s` opens a command bar at the bottom of the current view. The chef types `@target` to address an actor (with autocomplete), followed by their instructions.

```
╭─ noodle ─────────────────────────────────────────────────────────────────╮
│  ▸ cooking · 2 active · queue depth 4                                    │
│  ...                                                                     │
│                                                                          │
│  ┌─ steer ─────────────────────────────────────────────────────────────┐ │
│  │ @fix-auth-bug focus on unit tests first, skip the race condition   │ │
│  │               ╭──────────────────╮                                  │ │
│  │               │ fix-auth-bug   ● │                                  │ │
│  │               │ add-user-tests ● │                                  │ │
│  │               │ sous-chef        │                                  │ │
│  │               ╰──────────────────╯                                  │ │
│  └─────────────────────────────────────────────────── enter send · esc ─┘ │
╰──────────────────────────────────────────────────────────────────────────╯
```

The autocomplete dropdown appears when the user types `@`, showing all addressable actors: active cooks (with health dots) and the sous chef. Tab or enter selects. If the user is already viewing a session detail, `@cook-name` is pre-filled.

Uses [charmbracelet/huh](https://github.com/charmbracelet/huh) for the input form.

#### Steering a cook (`@cook-name`)

```
@fix-auth-bug focus on unit tests first, skip the race condition
```

1. Kill the targeted cook session
2. Build resume context from the cook's event log (files changed, last action, progress so far)
3. Respawn with the same backlog item, prepending the chef's instructions and the resume context to the prompt

#### Steering the sous chef (`@sous-chef`)

```
@sous-chef bring forward all the security tasks
@sous-chef deprioritize frontend work, focus on API hardening
@sous-chef route everything through claude opus for the next cycle
```

1. Trigger a new sous chef cycle immediately
2. The chef's instructions are prepended to the sous chef's prompt as additional context
3. The sous chef reads the current mise, applies the chef's guidance, and writes a new `queue.json`
4. The loop picks up the new queue via fsnotify

This lets the chef reshape priorities without manually editing the queue or skipping items one by one. The sous chef still has full context (mise, routing config, active cooks) — the chef's instructions add strategic direction on top.

Steering is the chef's primary creative control — not just pause/kill, but active redirection of any actor from wherever they are.

### Help Overlay (via `?` from any view)

```
╭─ Keys ──────────────────────────────────────────────╮
│                                                      │
│  Global                                              │
│  s         steer — talk to any actor (@name prompt)   │
│  p         pause / resume                            │
│  d         drain (finish active, then stop)          │
│  ?         this help                                 │
│  ctrl+c    quit                                      │
│                                                      │
│  Dashboard                                           │
│  enter     inspect cook                              │
│  q         queue view                                │
│                                                      │
│  Session Detail                                      │
│  t         trace — full event log                    │
│  k         kill cook                                 │
│  esc       back                                      │
│                                                      │
│  Queue                                               │
│  x         skip — remove from queue                  │
│  esc       back                                      │
│                                                      │
│  Trace                                               │
│  f         cycle filter (all/tools/think/ticket)     │
│  G         scroll to bottom                          │
│  esc       back                                      │
│                                                      │
╰─ esc close ──────────────────────────────────────────╯
```

The visible footer on each view shows only the 4-5 most common keys. `?` reveals the full set. This keeps the interface approachable — new users see the essentials, power users learn the rest.

### Queue View (entered via `q` from dashboard)

```
◀ back                       Queue                        4 items

  #   Item                    Provider       Model            Review
  1.  add-pagination          claude         sonnet-4.6       ✓
  2.  fix-mobile-nav          codex          gpt-5.3-codex    ✓
  3.  update-api-docs         codex          gpt-5.3-codex    ✗
  4.  refactor-db-queries     claude         opus-4.6         ✓

  Sous chef last ran 34s ago · next run after current cook completes

x skip · esc back · ? keys
```

### Halted State (self-healing or fatal error)

```
╭─ noodle ── HALTED ───────────────────────────────────────────────────╮
│  ⛔ Adapter script missing: .noodle/adapters/backlog-sync             │
│  Spawning repair cook to fix...                                      │
│                                                                      │
│  (or, if fatal:)                                                     │
│  ⛔ Cannot spawn: agents.claude_dir not found                         │
│  Run the bootstrap skill or set agents.claude_dir in config.         │
│  [q] quit                                                            │
╰──────────────────────────────────────────────────────────────────────╯
```

### End-of-Run Report

```
╭─ noodle ── COMPLETE ─────────────────────────────────────────────────╮
│  Duration:    2h 14m                                                 │
│  Cooks:       17                                                     │
│  Reviews:     14 accepted · 3 rejected                               │
│  Total cost:  $34.82                                                 │
│  Commits:     12                                                     │
╰──────────────────────────────────────────────────────────────────────╯
```

## Data Structures

- `Model` — Bubble Tea model. Holds: active view, cook states, queue, event subscriptions, terminal dimensions.
- `View` — Enum: `dashboard`, `session_detail`, `trace`, `queue`, `steer`, `help`
- `CookState` — Runtime state for display: name, status, health (green/yellow/red), provider, model, cost, duration, last action summary, last event time.
- `HealthDot` — Derived from `meta.json` health field (monitor is the single source of truth for health state). The TUI reads health, it does not compute it.
- `TraceFilter` — Enum: `all`, `tools`, `think`, `ticket`

The TUI watches `.noodle/sessions/` via fsnotify for state changes. It reads `meta.json` files (written by the monitor, Phase 7) for dashboard and session detail views. For the trace view, it tails the session's `events.ndjson` directly. The steer view uses [charmbracelet/huh](https://github.com/charmbracelet/huh) for the form input.

## Verification

### Static
- `go build ./tui/...` compiles
- `go test ./tui/...` — Bubble Tea model tests (Update + View produce expected output for given messages)
- Health display tests: TUI renders the correct dot/color for monitor-provided health values (green/yellow/red)

### Runtime
- `noodle tui` launches and shows the dashboard
- Active cooks appear in real-time with green health dots
- Health dots transition: green → yellow → red as monitor updates `meta.json` health field
- `enter` on a cook opens session detail with worktree, tickets, recent events
- `t` from session detail opens trace view with auto-scroll
- `f` in trace view cycles through filters (all → tools → think → ticket)
- Cost totals update as sessions progress
- `s` from any view opens steer command bar
- Typing `@` shows autocomplete dropdown with active cooks and sous-chef
- `@fix-auth-bug focus on tests` → cook killed, new cook spawned with resume context + chef instructions
- `@sous-chef bring forward security tasks` → immediate sous chef cycle with chef guidance, new queue written
- Steer cancel (esc): closes command bar, cook continues unchanged
- `s` from session detail pre-fills `@cook-name`
- `?` from any view opens help overlay, `esc` closes it
- `p` toggles pause/resume, dashboard header updates to show state
- Ctrl+C exits cleanly
