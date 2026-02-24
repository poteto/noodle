Back to [[archived_plans/26-context-usage-tracking/overview]]

# Phase 7: Surface context metrics in TUI

## Goal

Show context usage alongside cost in the TUI. Users should see at a glance which sessions are context-heavy and whether compression occurred.

**Dependency:** This phase builds on the TUI revamp ([[plans/25-tui-revamp/overview]]). Implement against the revamped layout, not the old single-pane dashboard.

## Changes

Target the Plan 25 layout components:

- **Left rail stats section** — Add context metrics alongside existing cost:
  - `72% ctx` peak context usage (next to `$4.82 / $50` cost)
  - Compression count if > 0 (e.g., `2 compactions`)

- **Agent list (left rail)** — Add compression indicator per active agent (dot or marker when `CompressionCount > 0`)

- **Feed tab verdict cards** — Include peak context % and compression count in completed cook cards alongside cost and file stats

- **Session detail view** — Show `PeakContextTokens`, `CompressionCount`, `Skill`, `TaskKey`

- **Snapshot layer** — Update TUI model structs (`tui/model.go`, `tui/model_snapshot.go`) to carry the new meta fields. The TUI has its own internal structs for sessions — these need the new fields added.

Read [[plans/25-tui-revamp/overview]] before implementing for layout structure and styling conventions. Use the `bubbletea-tui` skill for TUI patterns.

## Data structures

- TUI session model structs gain `PeakContextTokens`, `CompressionCount`, `TurnCount`, `Skill`, `TaskKey` fields to match enriched `Meta`.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | TUI design decisions, layout judgment |

## Verification

- `go test ./tui/...` — existing tests pass
- Visual: launch TUI with sessions that have context metrics → metrics visible
- Visual: session with compression shows indicator
