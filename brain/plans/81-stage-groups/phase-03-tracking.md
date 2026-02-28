Back to [[plans/81-stage-groups/overview]]

# Phase 3 — Multi-cook tracking per order

**Routing:** claude / claude-opus-4-6

## Goal

Change cook tracking from one-cook-per-order to multiple-cooks-per-order using a composite key. This unblocks spawning multiple stages from the same order.

## Changes

### `loop/types.go` — cook tracking map

Change `activeCooksByOrder map[string]*cookHandle` to use a composite key `"orderID:stageIndex"`. Add helper functions:

- `cookKey(orderID string, stageIndex int) string` — builds the composite key
- `cooksByOrder(orderID string) []*cookHandle` — returns all cooks for an order
- `removeCook(key string)` — removes a cook from the map

These helpers keep the key format encapsulated.

### `loop/cook_spawn.go` — `spawnCook`

Update the tracking write: `l.cooks.activeCooksByOrder[cookKey(cand.OrderID, cand.StageIndex)] = cook`

Update `atMaxConcurrency()` — currently counts `len(activeCooksByOrder)`. This still works since each cook has its own key.

### `loop/cook_completion.go`

Update cook lookup on completion to use the composite key. The `cookHandle` already stores `orderID` and `stageIndex`, so the key can be reconstructed.

### All other callers of `activeCooksByOrder`

Grep for `activeCooksByOrder` and update each site. Common patterns:
- Lookup by orderID → use `cooksByOrder(orderID)`
- Delete by orderID → delete by composite key
- Iteration → unchanged (flat map iteration still works)

## Verification

- `go test ./loop/...` — all existing tests pass (no behavioral change yet, just tracking refactor)
- `go vet ./...`
- Manual verification: start noodle, dispatch an order, verify cook tracking works as before
