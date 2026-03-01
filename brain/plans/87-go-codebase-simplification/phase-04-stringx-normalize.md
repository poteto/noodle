Back to [[plans/87-go-codebase-simplification/overview]]

# Phase 4: Add `stringx.Normalize` + Consolidate PID Liveness

## Goal

Two small utility consolidations in one phase.

### 4a: `stringx.Normalize`

Add `Normalize(s string) string` to `internal/stringx/stringx.go` — returns `strings.ToLower(strings.TrimSpace(s))`. Migrate call sites using the double-wrapped pattern.

### 4b: PID liveness deduplication

Replace `worktree/lock.go:isProcessAlive()` with `procx.IsPIDAlive()`. The implementations are functionally identical (both check `kill -0` + EPERM). Delete the local copy.

## Changes

- **`internal/stringx/stringx.go`** — add `Normalize` function
- **`internal/stringx/stringx_test.go`** — add tests for `Normalize`
- **All 65 non-test files** — mechanical replacement of `strings.ToLower(strings.TrimSpace(x))` with `stringx.Normalize(x)`. The actual count is 65 non-test sites (not ~20 as originally estimated). Migrate all of them — partial migration leaves dual idioms that defeat the simplification goal. Don't migrate instances where only `TrimSpace` is used (no `ToLower`).
- **`worktree/lock.go`** — delete `isProcessAlive`, import and use `procx.IsPIDAlive`

## Data Structures

- `func Normalize(s string) string` — single utility function

## Routing

- Provider: `codex`
- Model: `gpt-5.3-codex`
- Fully mechanical replacement. Codex excels at this.

## Verification

### Static
- `go test ./internal/stringx/...` — new tests pass
- `go test ./worktree/...` — PID liveness still works
- `go vet ./...` — clean
- Grep for `strings.ToLower(strings.TrimSpace(` in non-test files — **must be zero**. Partial migration is not acceptable.

### Runtime
- `go test ./...` — full suite passes
