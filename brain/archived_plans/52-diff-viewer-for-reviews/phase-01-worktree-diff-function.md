Back to [[archived_plans/52-diff-viewer-for-reviews/overview]]

# Phase 1: Worktree diff function

## Goal

Add a function to compute the unified diff and stat summary for a worktree branch against the integration branch. This is the data layer that the API endpoint will call.

## Changes

**`worktree/diff.go`** (new file)
- `DiffResult` struct: `Diff string`, `Stat string`
- `DiffWorktree(worktreePath string) (DiffResult, error)` — standalone function (not an `App` method). The server doesn't have a `worktree.App` and doesn't need one — the pending review item already carries the absolute worktree path. The function discovers the base branch by running `git -C <path> symbolic-ref refs/remotes/origin/HEAD` (falls back to `"main"` on error), then runs `git -C <path> diff <branch>...HEAD` and `git -C <path> diff --stat <branch>...HEAD`. Returns both as strings.
- Validate `worktreePath` exists before running git — return clear error if not (the worktree may have been cleaned up).
- Handle edge cases: no changes (return empty strings), git command failure (wrap error with context), relative worktree path (resolve to absolute using `filepath.Abs` before passing to git).

**`worktree/diff_test.go`** (new file)
- Test with a real temporary git repo: create a repo, add a worktree, make changes, commit, call `Diff()`, assert the result contains expected file names and +/- lines.
- Test error case: nonexistent path.

## Data structures

- `DiffResult{ Diff string, Stat string }`

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Straightforward Go function wrapping git commands |

## Verification

### Static
- `go vet ./worktree/...`
- `go test ./worktree/...`

### Runtime
- Run the test — it creates a real git worktree and verifies the diff output
- Edge case: empty diff (no changes on the branch)
- Edge case: binary files in the diff
