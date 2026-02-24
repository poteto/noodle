Back to [[plans/42-requires-approval-gate/overview]]

# Phase 6: TUI Approval Flow with Huh Text Input

## Goal

Rewrite the TUI approval flow to work with `pendingReview` state instead of verdict files. The current `m`/`x` keybindings are verdict-driven (`snapshot.Verdicts` + `ActionNeeded`) — Phase 3 deletes verdict loading, so these must be rebuilt from the persisted pending-review data. This phase also adds the "request changes" flow with a Huh text input overlay.

Use the `bubbletea-tui` and `go-best-practices` skills during implementation.

## Changes

### `go.mod`

Add `github.com/charmbracelet/huh` as a direct dependency.

### `tui/model_snapshot.go` — Replace verdict loading with pending-review

Remove `loadVerdicts` call and `Verdicts` field. Replace with pending-review data loaded from the persistence file (`.noodle/pending-review.json`, created in Phase 2). Add a `PendingReview []PendingReviewItem` field to `Snapshot` with item ID, worktree name, and session ID.

### `tui/feed.go` — Replace verdict cards with pending-review cards

Remove `verdicts` field from `FeedTab`. Replace verdict card rendering with pending-review cards that show the item ID, worktree, and action hints (m/x/c). The `cardCount`, `SetSnapshot`, `SelectedSessionID`, and `Render` methods all reference `f.verdicts` — rewrite to use `f.pendingReview`.

### `tui/model.go` — Rewrite `m`/`x` to use pending-review

Current `m`/`x` handlers call `mergeSelectedVerdict`/`rejectSelectedVerdict` which iterate `snapshot.Verdicts`. Rewrite to iterate the selected pending-review item and send merge/reject control commands. Delete `mergeSelectedVerdict`, `rejectSelectedVerdict`, `mergeAllApproved`, `isActionable`.

### `tui/model.go` — Key bindings

Add `c` (request changes) alongside the rewritten `m`/`x`:
- `m` — merge (rewritten to use pending-review)
- `x` — reject (rewritten to use pending-review)
- `c` — request changes (new)

### `tui/feedback_input.go` (new file)

A Bubble Tea component wrapping Huh's text input:

- `FeedbackInput` struct with a `huh.Form` containing a text area
- `Init()`, `Update()`, `View()` methods
- Returns a `feedbackSubmitMsg` with the text when submitted
- Returns a `feedbackCancelMsg` on Escape

### `tui/model.go` — State management

- Add `feedbackInput *FeedbackInput` field to `Model`
- Add `feedbackTargetID string` — which pending item the feedback is for
- When `c` is pressed on a pending item -> initialize `FeedbackInput`, set target ID
- When feedback overlay is active -> route key events to `FeedbackInput`
- On `feedbackSubmitMsg` -> send `request-changes` control command with prompt text
- On `feedbackCancelMsg` -> close overlay, return to normal mode

### `tui/model.go` — View

When `feedbackInput` is active, render it as an overlay on top of the current view (similar to how task editor works).

### `tui/model_render.go`

Update the overlay rendering branch (around line 48) to render the feedback input when active. Also update the keybar/action hints (around line 129) to show the `c` keybinding for pending-approval items.

### `tui/feed.go`

Update the help text shown for pending-approval items to include the `c` keybinding.

### Tests

- `tui/feedback_input_test.go` — test form creation, submit/cancel messages
- Verify the control command is sent correctly with the feedback text
- Test `m` sends merge control command for selected pending-review item
- Test `x` sends reject control command for selected pending-review item
- Test that `m`/`x`/`c` are no-ops when no pending-review items exist

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — TUI design decisions, Huh integration.

## Verification

```sh
go test ./tui/...
# Manual: launch TUI, set autonomy to approve, complete a cook, verify:
#   - Pending item appears
#   - m merges, x rejects
#   - c opens text input, submitting sends request-changes command
#   - New agent spawns with feedback in prompt
```
