Back to [[plans/87-go-codebase-simplification/overview]]

# Phase 4: Add `stringx.Normalize` + Migrate Callers

Split into sub-phases to limit blast radius (65 files is too many for one atomic change).

## 4a: Create `stringx.Normalize` + migrate first batch

### Goal

Add the helper and migrate the first ~20 call sites (packages `config/`, `internal/`, `mise/`).

### Changes

- **`internal/stringx/stringx.go`** — add `Normalize(s string) string` returning `strings.ToLower(strings.TrimSpace(s))`
- **`internal/stringx/stringx_test.go`** — add tests for `Normalize`
- **~20 files in `config/`, `internal/`, `mise/`** — mechanical replacement of `strings.ToLower(strings.TrimSpace(x))` with `stringx.Normalize(x)`. Don't migrate instances where only `TrimSpace` is used (no `ToLower`).

### Data Structures

- `func Normalize(s string) string` — single utility function

### Routing

- Provider: `codex`
- Model: `gpt-5.3-codex`
- Mechanical replacement.

### Verification

- `go test ./internal/stringx/...` — new tests pass
- `go vet ./...` — clean
- `go test ./...` — full suite passes

## 4b: Migrate remaining `stringx.Normalize` callers

### Goal

Complete the migration across remaining packages (`loop/`, `monitor/`, `dispatcher/`, `worktree/`, `parse/`, `skill/`, `server/`).

### Changes

- **~45 remaining non-test files** — same mechanical replacement

### Routing

- Provider: `codex`
- Model: `gpt-5.3-codex`

### Verification

- `go vet ./...` — clean
- `go test ./...` — full suite passes
- Grep for `strings.ToLower(strings.TrimSpace(` in non-test files — **must be zero**. Partial migration is not acceptable.

## 4c: Consolidate PID liveness

### Goal

Replace `worktree/lock.go:isProcessAlive()` with `procx.IsPIDAlive()`. The implementations are functionally identical (both check `kill -0` + EPERM). Delete the local copy.

### Changes

- **`worktree/lock.go`** — delete `isProcessAlive`, import and use `procx.IsPIDAlive`

### Routing

- Provider: `codex`
- Model: `gpt-5.3-codex`

### Verification

- `go test ./worktree/...` — PID liveness still works
- `go test ./...` — full suite passes
