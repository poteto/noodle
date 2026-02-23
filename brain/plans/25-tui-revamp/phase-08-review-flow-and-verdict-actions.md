Back to [[plans/25-tui-revamp/overview]]

# Phase 8: Review Flow and Verdict Actions

## Goal

Wire quality verdicts into Feed tab as actionable cards. In non-full autonomy, verdict cards show merge/reject/diff buttons. Add bulk merge. Show pending count in rail.

## Changes

### `tui/feed.go` — Extend feed items with verdict data

Feed items gain optional `Verdict` field. When present, card renders: verdict label (APPROVE/FLAG/REJECT in semantic color), taster comment, file stats, tests, cost. Below body: pill buttons for Merge, Reject, Diff.

Buttons only render when `autonomy != "full"`. Full mode shows informational cards (no buttons).

### `tui/verdict.go` — Verdict data and actions

Reads `.noodle/quality/` verdict files. Each contains: session ID, verdict, comment, file stats, tests.

Key type: `Verdict` struct, `LoadVerdicts(runtimeDir) ([]Verdict, error)`.

Merge: writes `merge` control command (added in Phase 7's control protocol extension) with session ID. Reject: writes `reject` control command. These actions were added to the loop in Phase 7 — this phase wires them into the TUI.

### `tui/model.go` — Wire verdict actions

Handle `m` (merge), `x` (reject), `d` (diff) on selected verdict card. `a` (merge-all-approved) iterates APPROVE verdicts and sends merge commands.

### `tui/rail.go` — Pending review counter

Show `N pending review` in rail when unactioned verdicts exist.

### `tui/model_snapshot.go` — Load verdict state

Add `Verdicts []Verdict` and `PendingReviewCount int` to Snapshot. Load from `.noodle/quality/`.

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`

## Verification

### Static
- `go test ./tui/...` passes
- Test: verdict card has buttons in review mode, none in full
- Test: merge writes correct control command
- Test: pending count matches unactioned verdicts
- Test: merge-all-approved skips FLAG/REJECT

### Runtime
- Verdict card appears in feed after cook completes
- Review mode: buttons visible, `m` merges
- Full mode: card shows, no buttons
- `a` bulk-merges APPROVE verdicts
- Rail counter updates on merge/reject
