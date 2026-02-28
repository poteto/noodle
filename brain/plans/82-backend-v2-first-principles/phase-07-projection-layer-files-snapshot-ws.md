# Phase 7: Projection Layer for Files, Snapshot, and WS

Back to [[plans/82-backend-v2-first-principles/overview]]

## Goal

Project all external views from canonical state so files, snapshot API, and websocket stream cannot drift.

## Changes

- Introduce projection layer generating `orders.json`, `status.json`, and snapshot payloads from canonical state
- Ensure websocket updates are emitted from projection changes, not independent state reads
- Keep `orders-next.json` ingestion unchanged, but consume into canonical event stream
- Retain append-only event logs as durable session history

## Data Structures

- `ProjectionBundle`
- `OrderProjection`
- `SnapshotView`
- `ProjectionHash`

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Implementation | `codex` | `gpt-5.3-codex` | Projection and stream wiring is mechanical |

## Verification

### Static

- Server/UI paths depend on projection outputs only
- Snapshot and WS envelope types generated from shared schema
- `go test ./... && go vet ./...`

### Runtime

- Golden projection tests for canonical scenarios
- Websocket tests for initial snapshot, incremental updates, subscribe/backfill ordering
- Edge cases: rapid state churn, slow clients, reconnect + backfill replacement semantics
