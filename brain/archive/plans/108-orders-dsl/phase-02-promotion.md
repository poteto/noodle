Back to [[plans/108-orders-dsl/overview]]

# Phase 2 — Wire into Promotion Flow

## Goal

Replace the `ParseOrdersStrict` call in `consumeOrdersNext` with `ParseCompactOrders` + `ExpandCompactOrders`. Update test fixtures from old format to compact format.

## Routing

Provider: `codex` | Model: `gpt-5.4`

## Changes

### Update: `loop/orders.go`

**`consumeOrdersNext`** (~line 305):

Replace:
```go
incoming, err := orderx.ParseOrdersStrict(nextData)
```

With:
```go
compact, err := orderx.ParseCompactOrders(nextData)
if err != nil { /* .bad rename, same as today */ }
incoming, err := orderx.ExpandCompactOrders(compact)
if err != nil { /* .bad rename with expansion error */ }
```

Everything after (dedup, merge, atomic write, delete) stays the same.

**Error handling:** Both parse errors and expansion errors → `.bad` rename + inject into `lastPromotionError`. Same flow as today, just two possible failure points instead of one.

### Update: `loop/testdata/**/orders-next.json`

Convert all `orders-next.json` fixtures to compact format:
- `task_key` → `do`
- `provider` → `with`
- Remove `skill` and `status` fields
- Keep `model`, `runtime`, `group`, `prompt`, `extra_prompt`, `extra` as-is

Known fixtures:
- `orders-next-promotion-with-inflight-cook/state-01/.noodle/orders-next.json`
- `orders-next-promoted-after-schedule-completes/state-01/.noodle/orders-next.json`
- Any others found during implementation

### Verify: `loop/testdata/**/orders.json` and `internal/snapshot/testdata/**/orders.json`

These are internal format — they should NOT change. Confirm tests still pass.

### Update: loop test assertions

Tests that assert on the structure of promoted orders may need updating if they checked for `status` being carried from the input (it's now always set by expansion). `skill` is set by expansion (`skill` = `do` value), so assertions on `skill` should still pass.

### Prerequisite: update `NormalizeAndValidateOrders` for ad-hoc stages

`isValidStageTaskType` in `internal/orderx/orders.go` drops stages that don't resolve via the task registry. Ad-hoc stages (no `TaskKey`, just `Prompt`) would be silently dropped. Update the validation to pass through stages where `TaskKey` is empty but `Prompt` is non-empty.

## Verification

- `go test ./loop/... -run "TestConsumeOrdersNext|TestPromotion|TestCycle"`
- `go test ./internal/snapshot/...`
- `go vet ./loop/...`
