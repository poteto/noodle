Back to [[archive/plans/49-work-orders-redesign/overview]]

# Phase 8: Snapshot and API

## Goal

Update the snapshot layer and SSE API to expose orders instead of flat queue items. The web UI (phase 9) consumes this.

## Changes

**`tui/`** — Delete the entire `tui/` package before replacing snapshot types. The TUI references `snapshot.QueueItem` extensively — if `QueueItem` is deleted first, the build breaks until `tui/` is also deleted. Deleting `tui/` here (not phase 10) ensures this phase lands with a clean build.

**`internal/snapshot/types.go`** — Replace queue types:
- Delete `QueueItem` struct (safe now that `tui/` is deleted above)
- Add `Order` and `Stage` structs mirroring the loop types (JSON-serializable)
- Update `Snapshot` struct: replace `Queue []QueueItem` with `Orders []Order`
- Replace `ActiveQueueIDs []string` with `ActiveOrderIDs []string`
- Keep `ActionNeeded []string` (order-level, unchanged)

**`internal/snapshot/snapshot.go`** — `LoadSnapshot()`:
- Read `orders.json` instead of `queue.json`. Phase 5 added a minimal `LoadSnapshot` patch that reads `orders.json` and converts to `QueueItem` shape as a bridge. This phase replaces that bridge with proper order types — delete the temporary conversion code from phase 5.
- Convert `queuex.OrdersFile` to `snapshot.Order` slice (package is still `queuex` at this point — optional rename to `orderx` happens in phase 10)
- Populate `ActiveOrderIDs` from active sessions metadata
- Update `queue-events.ndjson` parser: handle `order_drop` event type (phase 5 audit emits this instead of `queue_drop`). Keep `queue_drop` handling temporarily for events written before the migration.

**`server/sse.go`** — No changes needed if snapshot struct change is backward-compatible at the JSON level. The SSE hub serializes the full snapshot — it'll automatically include orders.

**`server/server.go`** — Verify `/api/snapshot` endpoint returns the updated snapshot shape. No code changes needed if snapshot type drives serialization.

## Data structures

- `snapshot.Order{ID, Title, Plan, Rationale, Stages, Status, OnFailure}` — mirrors loop Order including OnFailure stages
- `snapshot.Stage{TaskKey, Prompt, Skill, Provider, Model, Runtime, Status, Extra}` — mirrors loop Stage including Extra metadata
- Snapshot JSON shape changes: `queue: [...]` → `orders: [...]`

## Routing

| Provider | Model |
|----------|-------|
| `codex` | `gpt-5.3-codex` |

Mechanical type migration. Clear spec from phase 2 types.

## Verification

### Static
- `go build ./...` and `go vet ./...` pass
- No remaining references to `snapshot.QueueItem`

### Runtime
- Unit test: LoadSnapshot reads orders.json and populates snapshot.Orders
- Unit test: Snapshot serialization includes orders with stages, OnFailure, and Extra
- Manual: start noodle, hit `/api/snapshot`, verify orders appear in JSON response
- Unit test: LoadSnapshot handles both `order_drop` and legacy `queue_drop` event types in queue-events.ndjson
- Manual: verify SSE stream sends snapshots with orders field
- Unit test: snapshot serialization includes PendingReviewItem.Reason field (for phase 9 UI consumption)
- Integration test note: snapshot/API shape (`orders`, `active_order_ids`, `OnFailure`, `Extra`) consumed correctly by Board column derivation — cover in phase 10 integration tests
