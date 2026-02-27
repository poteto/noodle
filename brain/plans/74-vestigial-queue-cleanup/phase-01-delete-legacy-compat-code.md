Back to [[plans/74-vestigial-queue-cleanup/overview]]

# Phase 1: Delete legacy compat code

## Goal

Remove three pieces of dead/compat code that exist only to support formats no longer written.

## Changes

### 1. Delete `OrderID2` compat shim — `server/server.go`

The `controlRequest` struct has both `OrderID` (`order_id`) and `OrderID2` (`orderId`). The fallback at ~line 257 populates `OrderID` from `OrderID2` when empty. Delete:
- The `OrderID2` field from `controlRequest`
- The fallback `if cmd.OrderID == ""` block

**Check the UI first:** grep `ui/` for `orderId` (camelCase). If the UI sends `orderId`, update it to send `order_id` in the same commit. If the UI already sends `order_id`, this is a pure deletion.

### 2. Delete colon-separated format parser — `internal/snapshot/snapshot.go`

The `InferTaskType` function (~line 574) has a legacy branch that parses `orderID:stageIndex:taskKey` format. The current format is dasherized (`order-id-stageIndex-task-key`). Delete the colon-separated branch and its comment.

### 3. Replace `slicesEqual` with `slices.Equal` — `loop/queue.go`

The hand-rolled `slicesEqual` function at ~line 43 duplicates `slices.Equal` from stdlib. Replace the call site in `stampStatus()` with `slices.Equal` and delete the function. Add `"slices"` to the import block.

## Routing

Provider: `codex` | Model: `gpt-5.3-codex` — mechanical deletions with clear specs.

## Verification

```bash
go build ./... && go test ./... && go vet ./...
```

Confirm no test references the colon-separated format or `OrderID2`. If snapshot tests exist for `InferTaskType`, verify they still pass without the legacy branch (they should — fixtures use dasherized format).
