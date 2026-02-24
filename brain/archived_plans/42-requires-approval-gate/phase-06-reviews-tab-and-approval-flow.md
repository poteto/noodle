Back to [[archived_plans/42-requires-approval-gate/overview]]

# Phase 6: Reviews Tab and Approval Flow

## Goal

Delete the Config tab. Replace it with a Reviews tab that is the exclusive home for all approval workflow. The tab title shows the pending review count when non-zero: `Reviews (2)` or just `Reviews`. Approval keybindings (`m`/`x`/`c`) only work in the Reviews tab — not in Feed or Queue. `enter` shells out to `git diff` in the review's worktree, using the user's configured pager/diff tooling — no built-in diff viewer.

## Context

The `tui-feed-redesign` worktree has already:
- Removed the Brain tab (tabs are now Feed, Queue, Config)
- Redesigned Feed as a read-only agent dashboard (no approval UI)
- Removed h/l actor navigation on Feed

Phase 3 removes verdicts and approval workflow from Feed. This phase replaces Config with Reviews and rebuilds approval there.

## Changes

### Delete `tui/config_tab.go`

Delete entirely. The autonomy dial moves out — Phase 4 simplifies to 2 modes (`auto`/`approve`), controlled via `.noodle.toml` not a TUI widget.

### `tui/tab_bar.go` — Dynamic Reviews tab title

- Rename `TabConfig` to `TabReviews`
- Change `tabNames` from static `[3]string` to a function `tabName(tab Tab, pendingCount int) string`
- Feed and Queue return fixed names. Reviews returns `"Reviews (N)"` when `pendingCount > 0`, otherwise `"Reviews"`
- `renderTabBar` takes an additional `pendingReviewCount int` parameter
- `Tab.String()` returns `"Reviews"` for `TabReviews`

### `tui/reviews_tab.go` (new file)

A new `ReviewsTab` component that renders pending review items and handles approval actions.

```go
type ReviewsTab struct {
    items     []PendingReviewItem
    selection int
    scroll    int
}
```

- `PendingReviewItem` struct: item ID, worktree path, session ID, skill name, summary (from pending-review persistence, Phase 2)
- `SetPendingReviews(items []PendingReviewItem)` — called from snapshot refresh
- `SelectUp()` / `SelectDown()` — navigate items
- `SelectedItem() (PendingReviewItem, bool)` — returns currently selected item
- `Render(width, height int) string` — compact row list with selection indicator. Empty state: "No pending reviews."

Layout (Option B — compact rows):

```
PENDING REVIEW

▸ fix-login-redirect    review   gpt-4.1
  Redirect loop when session expires on /dashboard
  .worktrees/fix-login-redirect

  add-retry-backoff     cook     gpt-4.1
  Add exponential backoff to HTTP client retries
  .worktrees/add-retry-backoff


enter view diff  m merge  x reject  c request changes
```

Selected row gets `▸` indicator. Action legend sits at the bottom of the content area. Each item renders 3 lines: name/skill/model, summary, worktree path.

### `tui/model_snapshot.go` — Load pending reviews into snapshot

Add `PendingReviews []PendingReviewItem` to `Snapshot`. Load from `.noodle/pending-review.json` (created in Phase 2). Remove `Verdicts` field and `loadVerdicts` if not already done in Phase 3.

### `tui/model.go` — Replace ConfigTab with ReviewsTab

- Replace `configTab ConfigTab` with `reviewsTab ReviewsTab`
- In `snapshotMsg` handler: replace `m.configTab.SetAutonomy(...)` with `m.reviewsTab.SetPendingReviews(m.snapshot.PendingReviews)`
- Tab `"3"` switches to `TabReviews` (rename from `TabConfig`)

### `tui/model.go` — `enter` shells out to `git diff`

Use `tea.ExecProcess` to suspend the TUI and run `git diff` in the selected review's worktree. The user's configured `$GIT_PAGER`, `core.pager`, and `diff.tool` settings are respected automatically — no custom config needed.

```go
case "enter":
    if m.activeTab == TabReviews {
        if item, ok := m.reviewsTab.SelectedItem(); ok {
            c := exec.Command("git", "-C", item.WorktreePath, "diff", "main", "--stat", "-p")
            return m, tea.ExecProcess(c, func(err error) tea.Msg {
                return diffExitMsg{err: err}
            })
        }
    }
```

The diff target branch (`main`) should come from the same integration branch detection used by `noodle worktree merge`. Add a `BaseBranch` field to `PendingReviewItem` populated during snapshot loading, falling back to `main`.

On `diffExitMsg`, the TUI resumes — no state change needed.

### `tui/model.go` — Approval keybindings in Reviews tab only

Move `m`/`x`/`c` handlers to `TabReviews` only:

```go
case "m":
    if m.activeTab == TabReviews {
        if item, ok := m.reviewsTab.SelectedItem(); ok {
            return m, sendControlCmd(m.runtimeDir, m.now, loop.ControlCommand{
                Action: "merge", Item: item.ID,
            })
        }
    }
case "x":
    if m.activeTab == TabReviews {
        if item, ok := m.reviewsTab.SelectedItem(); ok {
            return m, sendControlCmd(m.runtimeDir, m.now, loop.ControlCommand{
                Action: "reject", Item: item.ID,
            })
        }
    }
case "c":
    if m.activeTab == TabReviews {
        // Open feedback input overlay (see below)
    }
```

Delete `mergeSelectedVerdict`, `rejectSelectedVerdict`, `mergeAllApproved`, `isActionable` methods.

### `tui/feedback_input.go` (new file)

Bubble Tea component wrapping Huh's text input for "request changes" feedback:

- `FeedbackInput` struct with a `huh.Form` containing a text area
- `Init()`, `Update()`, `View()` methods
- Returns `feedbackSubmitMsg{text, targetID}` on submit
- Returns `feedbackCancelMsg{}` on Escape

### `tui/model.go` — Feedback overlay state

- Add `feedbackInput *FeedbackInput` and `feedbackTargetID string` fields
- `c` key on Reviews tab: initialize `FeedbackInput`, set target ID
- When overlay active: route keys to `FeedbackInput`
- On submit: send `request-changes` control command with feedback text
- On cancel: close overlay

### `tui/model_render.go` — Wire up Reviews tab

- `renderLayout`: replace `case TabConfig` with `case TabReviews`
- Pass `m.snapshot.PendingReviewCount` (or `len(m.snapshot.PendingReviews)`) to `renderTabBar`
- When `feedbackInput` is active, render as overlay (same pattern as `taskEditor`)
- Keybar for `TabReviews`: `j/k select`, `enter view diff`, `m merge`, `x reject`, `c request changes`
- Remove `TabConfig` keybar case and `←/→ autonomy` hint
- Remove `h/l actors` from Reviews tab (approval items, not actor sessions)

### `go.mod`

Add `github.com/charmbracelet/huh` as a direct dependency.

### Tests

- `tui/reviews_tab_test.go` — selection, empty state, pending count
- `tui/feedback_input_test.go` — form creation, submit/cancel messages
- Update `tui/model_test.go` — tab `"3"` maps to `TabReviews`, `m`/`x`/`c` only work on Reviews tab

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — TUI design decisions, Huh integration, component architecture.

## Verification

```sh
go test ./tui/...
# Confirm no Config tab references remain:
rg 'TabConfig|ConfigTab|config_tab|configTab' tui/
# Confirm approval keybindings only in Reviews:
rg 'mergeSelectedVerdict|rejectSelectedVerdict|mergeAllApproved' tui/
# Manual: launch TUI, verify:
#   - Tab 3 shows "Reviews" (or "Reviews (N)")
#   - enter suspends TUI, opens git diff in user's pager, resumes on exit
#   - m/x/c only work on Reviews tab
#   - c opens text input, submitting sends request-changes
```
