Back to [[plans/42-requires-approval-gate/overview]]

# Phase 6: TUI Approval Flow with Huh Text Input

## Goal

Wire up the TUI to support all three approval actions: approve (merge), reject, and request changes. The first two already work via keybindings. This phase adds the "request changes" flow with a Huh text input overlay for feedback.

Use the `bubbletea-tui` and `go-best-practices` skills during implementation.

## Changes

### `go.mod`

Add `github.com/charmbracelet/huh` as a direct dependency.

### `tui/model.go` — Key bindings

Add a new keybinding for "request changes" on pending-approval items. Suggest `c` (for "changes") since `r` may conflict with other bindings:
- `m` — merge (existing)
- `x` — reject (existing)
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

### `tui/feed.go` or `tui/verdict.go`

Update the help text shown for pending-approval items to include the `c` keybinding.

### Tests

- `tui/feedback_input_test.go` — test form creation, submit/cancel messages
- Verify the control command is sent correctly with the feedback text

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
