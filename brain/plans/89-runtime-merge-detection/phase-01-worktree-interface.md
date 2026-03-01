Back to [[plans/89-runtime-merge-detection/overview]]

# Phase 1 — Add `HasUnmergedCommits` to worktree interface

## Goal

Expose the existing `countUnmergedCommits` logic through the `WorktreeManager` interface so the loop can query whether a worktree has changes to merge.

## Changes

**`worktree/app.go`** — Add public method:
- `HasUnmergedCommits(name string) (bool, error)` — delegates to existing `cherryStatus`. Returns error on git failures instead of silently collapsing to `false`.

**`loop/types.go`** — Add to `WorktreeManager` interface:
- `HasUnmergedCommits(name string) (bool, error)`

**`loop/loop_test.go`** — Update `fakeWorktree`:
- Add `hasUnmergedCommits map[string]bool` field
- Implement `HasUnmergedCommits` — returns `true, nil` by default (preserves existing test behavior where stages are expected to merge), `false, nil` only when explicitly set

**`loop/defaults.go`** — Update `noOpWorktree`:
- Add `HasUnmergedCommits(string) (bool, error)` returning `false, nil`

## Data structures

- No new types. One method addition to an existing interface. Signature returns `(bool, error)` — git failures must not silently become "no changes."

## Routing

| Provider | Model |
|----------|-------|
| `codex` | `gpt-5.3-codex` |

## Verification

```
go test ./worktree/... ./loop/...
go vet ./worktree/... ./loop/...
```

- Confirm `fakeWorktree` and `noOpWorktree` both compile and satisfy the interface
- Existing tests pass unchanged (default `true` preserves behavior)
- Verify error propagation: a git failure returns `(false, err)`, not `(false, nil)`
