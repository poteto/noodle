Back to [[plans/89-runtime-merge-detection/overview]]

# Phase 1 — Add `HasUnmergedCommits` to worktree interface

## Goal

Expose the existing `countUnmergedCommits` logic through the `WorktreeManager` interface so the loop can query whether a worktree has changes to merge.

## Changes

**`worktree/app.go`** — Add public method:
- `HasUnmergedCommits(name string) bool` — delegates to existing `countUnmergedCommits(name) > 0`

**`loop/types.go`** — Add to `WorktreeManager` interface:
- `HasUnmergedCommits(name string) bool`

**`loop/loop_test.go`** — Update `fakeWorktree`:
- Add `hasUnmergedCommits map[string]bool` field
- Implement `HasUnmergedCommits` — returns `true` by default (preserves existing test behavior where stages are expected to merge), `false` only when explicitly set

## Data structures

- No new types. One method addition to an existing interface.

## Routing

| Provider | Model |
|----------|-------|
| `codex` | `gpt-5.3-codex` |

## Verification

```
go test ./worktree/... ./loop/...
go vet ./worktree/... ./loop/...
```

- Confirm `fakeWorktree` compiles and satisfies the interface
- Existing tests pass unchanged (default `true` preserves behavior)
