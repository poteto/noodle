Back to [[archived_plans/52-diff-viewer-for-reviews/overview]]

# Phase 2: Diff API endpoint

## Goal

Add `GET /api/reviews/{id}/diff` to the server. This endpoint resolves the review item, computes the diff from its worktree path, and returns it as JSON.

## Changes

**`server/server.go`**
- No new fields on `Server` — the endpoint calls the standalone `worktree.DiffWorktree(path)` function directly. The pending review item already carries the absolute `worktree_path`, so no project dir or `worktree.App` is needed.
- Register new route: `mux.HandleFunc("GET /api/reviews/{id}/diff", s.handleReviewDiff)`
- `handleReviewDiff` handler:
  1. Extract `{id}` from path via `r.PathValue("id")`
  2. Call `loop.ReadPendingReview(s.runtimeDir)` to load pending reviews
  3. Find the item by ID — 404 if not found
  4. Call `worktree.DiffWorktree(item.WorktreePath)` — base branch discovered from the worktree's remote HEAD, falls back to "main"
  5. Return `{ "diff": "...", "stat": "..." }` as JSON
  6. Handle errors: 404 for missing review item, 500 for git failures with descriptive error message

## Data structures

- Response: `struct{ Diff string; Stat string }` with json tags `"diff"`, `"stat"`

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical endpoint following existing server patterns |

## Verification

### Static
- `go vet ./server/...`
- `go test ./server/...` (if server has tests)

### Runtime
- Start noodle, park a review item, `curl localhost:<port>/api/reviews/<id>/diff` — verify JSON response with diff and stat fields
- Test 404: request diff for nonexistent review ID
- Test error: request diff for item whose worktree was already cleaned up
