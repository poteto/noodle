Back to [[plans/115-canonical-state-convergence/overview]]

# Phase 6 — Projection-Backed Snapshot Cutover

## Goal

Move `/api/snapshot` and websocket `snapshot` delivery onto projection-backed full snapshots as soon as the execution path is canonical enough, with one structural snapshot owner and without introducing a new delta protocol.

## Changes

- **`internal/projection/projection.go`** — become the sole owner of structural snapshot topology, including projected review state and projected order/state output
- **`internal/snapshot/snapshot_builder.go`** — reduce to runtime enrichment over `projection.SnapshotView` instead of rebuilding topology from `LoopState`
- **`server/server.go` + `server/ws_hub.go`** — keep the existing full-snapshot API/WS contract, but source it from projection-backed snapshots rather than `LoopStateProvider`
- **`loop/state_snapshot.go` + projection file writers** — make the authority cutover explicit: `orders.json` / `state.json` become pure projection output in this phase, and legacy workflow mutation of those files is deleted

## Data structures

- `projection.ProjectionBundle` — authoritative projected state input for snapshot delivery
- `projection.SnapshotView` — sole structural snapshot model consumed by the server and enriched at runtime
- snapshot field-ownership table covering `orders`, `mode`, `active/recent sessions`, `pending reviews`, `feed events`, `action_needed`, `total_cost_usd`, and `max_concurrency`

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | Read-model ownership and server cutover are architecture-heavy |

## Verification

### Static
- `pnpm test:smoke`
- `go test ./internal/projection/... ./internal/snapshot/... ./server/...`
- `go vet ./internal/projection/... ./internal/snapshot/... ./server/...`

### Runtime
- prove `/api/snapshot` equivalent output from legacy and projection-backed builders for representative states before deleting the legacy source
- prove websocket `snapshot` delivery still works end-to-end with projection-backed full snapshots
- prove every snapshot field has exactly one owner during and after cutover
- prove `orders.json` and `state.json` are projection output only after this phase and are no longer mutated as workflow state
- confirm the legacy snapshot source is deleted or unreachable in the same phase once projection-backed full snapshots are proven
- rerun `pnpm test:smoke` after the projection/snapshot authority cutover and treat unexpected failures as blockers
