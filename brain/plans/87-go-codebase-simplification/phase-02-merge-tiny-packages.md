Back to [[plans/87-go-codebase-simplification/overview]]

# Phase 2: Merge Tiny Packages

## Goal

Eliminate package overhead for packages too small to justify their own directory. Per subtract-before-you-add, simplify the package tree before building new shared infrastructure.

## Changes

- **`internal/failure/`** — keep the package as-is. Adversarial review found that failure taxonomy covers system/session/runtime scopes beyond just orders, and `orderx` already pulls config/registry concerns. Merging would couple unrelated domains. Instead, only inline the tiny `display.go` helper (`OwnerPrefixForDisplay`) at its call sites and delete that file if it has no other exports.

- **`runtime/sprites.go`** — this 10-line file is a thin wrapper around `NewDispatcherRuntime()`. Inline it at call sites and delete the file if it becomes empty. If the entire `runtime/` package becomes trivially small, consider whether it still justifies its own package.

## Data Structures

No new types. Existing types move packages — update import paths.

## Routing

- Provider: `codex`
- Model: `gpt-5.3-codex`
- Mechanical moves with import updates.

## Verification

### Static
- `go build ./...` — all imports resolve
- `go test ./internal/orderx/...` — merged tests pass
- `go vet ./...` — clean

### Runtime
- `go test ./...` — full suite passes
