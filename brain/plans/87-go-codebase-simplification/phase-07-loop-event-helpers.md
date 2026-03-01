Back to [[plans/87-go-codebase-simplification/overview]]

# Phase 7: Extract Loop Event Helpers

## Goal

Extract two repeated patterns in the `loop/` package into named helpers.

## Changes

### 7a: Stage failure state transition + source-specific orchestrators

The 6 call sites share a common state-transition core (build failure metadata, emit StageFailed + OrderFailed events, classify order) but diverge materially in side effects:
- `cook_completion` emits canonical V2 events + scheduler forwarding + worktree cleanup
- `reconcile` skips canonical events (no session available)
- `control_review` carries `AgentMistake` semantics

**Do NOT create a single polymorphic option-bag helper.** Instead, split into:

1. **`(l *Loop).recordStageFailure(cook *cookHandle, reason string, orderClass OrderFailureClass, mistake *AgentMistakeEnvelope)`** — state transition only. Builds failure metadata and emits the two loop events (`StageFailed`, `OrderFailed`). **Does NOT classify the order** — callers handle their own classification after calling this helper. `control_review` paths use `classifyCookMistake` (preserves `CookReason` semantics for scheduler feedback), while other paths use `classifyOrderHard`.

2. **Source-specific callers** remain responsible for their own side effects (canonical V2 events, scheduler forwarding, worktree cleanup) by calling `recordStageFailure` then handling their specific post-conditions.

Migrate all 6 call sites:
- `cook_completion.go:214-237`
- `cook_merge.go:191-206`
- `cook_spawn.go:239-255`
- `reconcile.go:351-366`
- `control_review.go:108-122`
- `control_review.go:172-186`

### 7b: Cook worktree cleanup helper

Create `(l *Loop).cleanupCookWorktree(cook *cookHandle)` to replace the 3 instances of `if strings.TrimSpace(cook.worktreeName) != "" { l.deps.Worktree.Cleanup(...) }`.

Migrate:
- `cook_completion.go:248-249`
- `cook_merge.go:221-222`
- `control.go:330-331` (note: this one skips TrimSpace — standardize on the safer check)

## Data Structures

- `func (l *Loop) recordStageFailure(...)` — pure state transition method on Loop
- `func (l *Loop) cleanupCookWorktree(cook *cookHandle)` — method on Loop

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`
- The stage failure helper requires understanding the 6 variants to design a clean parameterized API.

## Verification

### Static
- `go test ./loop/...` — all loop tests pass
- `go vet ./loop/...` — clean

### Runtime
- `go test ./...` — full suite passes
- Visual inspection: each migrated call site is a single function call, not a multi-line sequence
