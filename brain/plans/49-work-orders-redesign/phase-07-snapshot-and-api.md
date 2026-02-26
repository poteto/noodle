Back to [[plans/49-work-orders-redesign/overview]]

# Phase 7: Snapshot and API

## Goal

Update the snapshot layer and SSE API to expose orders instead of flat queue items. The web UI (phase 8) consumes this.

## Changes

**`internal/snapshot/types.go`** — Replace queue types:
- Delete `QueueItem` struct
- Add `Order` and `Stage` structs mirroring the loop types (JSON-serializable)
- Update `Snapshot` struct: replace `Queue []QueueItem` with `Orders []Order`
- Replace `ActiveQueueIDs []string` with `ActiveOrderIDs []string`
- Keep `ActionNeeded []string` (order-level, unchanged)

**`internal/snapshot/snapshot.go`** — `LoadSnapshot()`:
- Read `orders.json` instead of `queue.json`
- Convert `orderx.OrdersFile` to `snapshot.Order` slice
- Populate `ActiveOrderIDs` from active sessions metadata
- Update `queue-events.ndjson` parser: handle `order_drop` event type (phase 4 audit emits this instead of `queue_drop`). Keep `queue_drop` handling temporarily for events written before the migration.

**`server/sse.go`** — No changes needed if snapshot struct change is backward-compatible at the JSON level. The SSE hub serializes the full snapshot — it'll automatically include orders.

**`server/server.go`** — Verify `/api/snapshot` endpoint returns the updated snapshot shape. No code changes needed if snapshot type drives serialization.

## Data structures

- `snapshot.Order{ID, Title, Plan, Rationale, Stages, Status}`
- `snapshot.Stage{TaskKey, Prompt, Skill, Provider, Model, Runtime, Status}`
- Snapshot JSON shape changes: `queue: [...]` → `orders: [...]`

## Routing

| Provider | Model |
|----------|-------|
| `codex` | `gpt-5.3-codex` |

Mechanical type migration. Clear spec from phase 1 types.

## Verification

### Static
- `go build ./...` and `go vet ./...` pass
- No remaining references to `snapshot.QueueItem`

### Runtime
- Unit test: LoadSnapshot reads orders.json and populates snapshot.Orders
- Unit test: Snapshot serialization includes orders with stages
- Manual: start noodle, hit `/api/snapshot`, verify orders appear in JSON response
- Manual: verify SSE stream sends snapshots with orders field
