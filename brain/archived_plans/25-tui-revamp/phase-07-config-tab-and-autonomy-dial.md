Back to [[archived_plans/25-tui-revamp/overview]]

# Phase 7: Config Tab and Autonomy Dial

## Goal

Implement Config tab with interactive autonomy dial — a visual 3-position slider between approve/review/full that changes system behavior in real-time.

## Changes

### `config/config.go` — Replace `review.enabled` with `autonomy` (loop prerequisite)

Replace `ReviewConfig.Enabled bool` with `Autonomy string` on `Config` (values: `"full"`, `"review"`, `"approve"`, default `"full"`). Migration: `review.enabled = true` → `autonomy = "review"`, `review.enabled = false` → `autonomy = "full"`. Remove `ReviewConfig` entirely. Update `DefaultConfig()`, `applyDefaultsFromMetadata`, and all call sites that read `review.enabled` (currently `loop/cook.go` uses `cook.reviewEnabled`).

Add config test: parsing old `review.enabled` TOML produces correct autonomy value. Add test: new `autonomy` field parses correctly.

### `loop/cook.go` — Pending-review state machine (loop prerequisite)

Currently `handleCompletion` auto-merges on success (line 112). Change: when `autonomy != "full"`, successful cooks with APPROVE verdict enter a `"pending-review"` state instead of auto-merging. The worktree stays intact. Add `pendingReview []*activeCook` to Loop state. Auto-merge still happens in `"full"` mode.

### `loop/control.go` — Extend control protocol (loop prerequisite)

Add control actions: `merge` (merges a pending-review cook by session ID), `reject` (cleans up worktree, marks failed), `autonomy` (changes autonomy mode at runtime), `enqueue` (adds item to backlog via adapter), `stop-all` (kills all active cooks), `requeue` (re-queues a failed/rejected item).

Current protocol: `pause|resume|drain|skip|kill|steer`. After: adds `merge|reject|autonomy|enqueue|stop-all|requeue`.

### `tui/config.go` — Config tab implementation

Renders sections: Autonomy (with dial), Routing, Budget, Adapters, Controls. Values read from config + runtime state.

Key type: `ConfigTab` with sections, selected control index, autonomy mode.

### `tui/components/dial.go` — Autonomy dial component

Visual 3-position slider: `approve ── review ── full` with marker on current position. Left/right moves marker. Active position in pastel yellow.

Key type: `Dial` with `positions []string`, `selected int`, `Render(width) string`.

### `tui/model.go` — Wire autonomy changes

When dial changes, write `autonomy` control command to `.noodle/control.ndjson`. Controls section: Pause writes `pause`, Stop All writes `stop-all` (new), Re-Queue writes `requeue` (new). All go through existing `sendControlCmd`.

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`

## Verification

### Static
- `go test ./...` passes (loop + TUI tests)
- Test: `review.enabled = true` parses to `autonomy = "review"`
- Test: `autonomy = "approve"` parses correctly
- Test: pending-review cooks don't auto-merge when autonomy is `"review"`
- Test: `merge` control action merges pending-review cook
- Test: `reject` control action cleans up worktree
- Test: `stop-all` kills all active cooks
- Test: `enqueue` adds item via backlog adapter
- Test: dial renders correct marker for each mode
- Test: left/right changes position

### Runtime
- Dial renders as visual slider
- Arrow keys move between positions
- Autonomy change writes to control.ndjson and loop respects it
- Budget shows progress bar
- Pause/Stop All/Re-Q buttons write correct control commands
