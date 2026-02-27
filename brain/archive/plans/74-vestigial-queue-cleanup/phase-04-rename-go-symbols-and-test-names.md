Back to [[archive/plans/74-vestigial-queue-cleanup/overview]]

# Phase 4: Rename Go symbols and test names

## Goal

Rename internal Go symbols and test function names that still use "queue" terminology. These don't affect external behavior but make the codebase internally consistent.

## Changes

### `cmd_status.go` + `cmd_status_test.go`

- Rename `QueueDepth` field → `OrdersDepth` in `statusSummary` struct
- Update CLI output strings: `queue=%d` → `orders=%d`
- Update all references in `cmd_status.go` and `cmd_status_test.go`

### Test function/comment renames

Rename test functions and update comments across these files. Use `replace_all` or find-and-replace — these are mechanical:

**`loop/loop_test.go`:**
- `TestCycleSpawnsCookFromQueue` → `TestCycleSpawnsCookFromOrders`
- Comments: "write queue" → "write orders" (~17 occurrences)

**`loop/sous_chef_test.go`:**
- `TestBuildQueueTaskTypesPrompt*` → `TestBuildOrderTaskTypesPrompt*`
- Test data strings: "When the queue is empty" → "When orders are empty"

**`internal/orderx/orders_test.go`:**
- `TestQueuex*` prefix → `TestOrderx*` prefix on all test functions

**`cmd_status_test.go`:**
- `TestRunStatusReadsSessionsAndQueue` → `TestRunStatusReadsSessionsAndOrders`

**`loop/log_test.go`:**
- `TestLogQueueNextPromoted` → `TestLogOrdersNextPromoted`
- Comments: "Empty queue" → "Empty orders", "No queue file" → "No orders file"

## Routing

Provider: `codex` | Model: `gpt-5.3-codex` — mechanical find-and-replace across test files.

## Verification

```bash
go build ./... && go test ./... && go vet ./...
grep -rn 'Queue\|queue' --include='*.go' loop/ internal/orderx/ cmd_status*.go
```

Review grep output — remaining matches should only be `MergeQueue` (active system), `enqueue`/`requeue` (active control commands), and `enqueueTrigger`/`enqueueCompletion` (active internal helpers). No matches for `QueueDepth`, `FromQueue`, `TestQueuex`, or `TestBuildQueue`.
