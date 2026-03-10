Back to [[archive/plans/74-vestigial-queue-cleanup/overview]]

# Phase 2: Rename vestigial files

## Goal

Three Go source files have names from the queue era. Their contents are already correct (order types, status stamping, order audit tests) — only the filenames are misleading.

## Changes

### 1. `loop/queue.go` → `loop/stamp_status.go`

Contains `stampStatus()` and nothing else (after phase 1 deletes `slicesEqual`). The name `queue.go` is misleading.

### 2. `loop/queue_audit_test.go` → `loop/order_audit_test.go`

Tests for `order_audit.go` — the source file was already renamed but the test file wasn't.

### 3. `internal/orderx/queue.go` → `internal/orderx/types.go`

Contains `Order`, `Stage`, `OrderStatus`, `StageStatus` type definitions. Pure types file, nothing queue-related.

## Execution

Use `git mv` for each rename so git tracks the history. No content changes needed — these are pure renames.

```
git mv loop/queue.go loop/stamp_status.go
git mv loop/queue_audit_test.go loop/order_audit_test.go
git mv internal/orderx/queue.go internal/orderx/types.go
```

## Routing

Provider: `codex` | Model: `gpt-5.4` — three `git mv` commands, no code changes.

## Verification

```bash
go build ./... && go test ./...
```

All imports remain unchanged (Go imports reference packages, not files). Tests should pass without modification.
