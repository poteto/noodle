Back to [[plans/69-cursor-dispatcher/overview]]

# Phase 1: Extract shared remote dispatcher utilities

**Routing:** `codex` / `gpt-5.3-codex` — mechanical refactor, move functions between files

## Goal

Extract git and sync-back utilities from `sprites_dispatcher.go` into shared files so both sprites and the upcoming cursor dispatcher can reuse them. Pure refactor — no behavior changes.

## Data structures

No new types. Existing functions are moved to new files.

## Changes

**`dispatcher/git.go` (new)**

Move these functions from `sprites_dispatcher.go`:
- `gitRemoteURL(repoPath string) string` — reads origin URL
- `gitCurrentBranch(repoPath string) string` — reads current branch
- `pushWorktreeBranch(ctx, worktreePath, branch string) error` — force-pushes worktree branch to origin

Keep `authenticatedRemoteURL()` in `sprites_dispatcher.go` — it's sprites-specific (Cursor has its own GitHub integration and doesn't need token injection).

**`dispatcher/sync.go` (new)**

Move from `sprites_dispatcher.go`:
- `writeSyncResult(runtimeDir, sessionID string, result SyncResult) error` — updates spawn.json with sync result

**`dispatcher/sprites_dispatcher.go`**

Remove the moved functions. Add import if needed (they're in the same package so no import changes).

## Verification

### Static
- `go build ./dispatcher/...` — compiles
- `go vet ./dispatcher/...` — passes
- All existing tests pass unchanged: `go test ./dispatcher/... -race`

### Runtime
- No new tests needed — this is a pure move refactor
- Verify sprites dispatcher still works by running existing sprites tests
