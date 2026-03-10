Back to [[plans/87-go-codebase-simplification/overview]]

# Phase 12: Split Large Files

## Goal

Split large files where a specific maintenance pain is demonstrated. One package at a time to limit merge-conflict surface. No logic changes — pure file reorganization.

**Scope reduced from original plan.** Only split files where the size actively hinders navigation or creates merge conflicts during concurrent work. Do not blanket-split everything over 350 lines.

## Changes

### 12a: Worktree files (highest pain — Merge function decomposition in phases 9-10 already touches this code)

**`worktree/commands.go` (400 lines)** → split into:
- `commands.go` — Create, Exec
- `commands_merge.go` — Merge, MergeRemoteBranch
- `commands_cleanup.go` — Cleanup, Prune, cleanRemote

### 12b: Parser files (second priority — frequently edited during provider changes)

**`parse/codex.go` (428 lines)** → split into:
- `codex.go` — `CodexAdapter` struct, `Parse()` method, envelope types
- `codex_items.go` — `parseCodexItem`, `parseCodexResponseItem`, item-related helpers
- `codex_events.go` — `parseCodexEventMsg`, `parseCodexTurnContext`, event helpers

**`parse/claude.go` (349 lines)** → split into:
- `claude.go` — `ClaudeAdapter` struct, `Parse()` method, envelope types
- `claude_tools.go` — tool use/result parsing helpers

### 12c: Minor cleanups (low risk, high clarity)

- **`dispatcher/session_base.go`** — add constants for magic buffer sizes (`scannerInitialBuffer`, `scannerMaxBuffer`)

**Deferred:** `worktree/app.go` and `config/types_defaults.go` splits — only do these if a subsequent feature creates concrete merge-conflict pain.

## Data Structures

No new types. Pure file reorganization within same packages.

## Routing

- Provider: `codex`
- Model: `gpt-5.4`
- Mechanical file splits. No logic changes.

## Verification

### Static
- `go build ./parse/... ./worktree/... ./dispatcher/... ./config/...` — compiles
- `go test ./parse/... ./worktree/... ./dispatcher/... ./config/...` — tests pass
- `go vet ./...` — clean
- No source file in modified packages exceeds 300 lines

### Runtime
- `go test ./...` — full suite passes
