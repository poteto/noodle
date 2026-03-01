# Phase 7: Projection Layer for Files, Snapshot, and WS

Back to [[plans/82-backend-v2-first-principles/overview]]

## Goal

Project all external views from canonical state so files, snapshot API, and websocket stream cannot drift.

## Changes

- Introduce projection layer generating `orders.json`, `status.json`, `.noodle/state.json`, and snapshot payloads from canonical state
- Ensure websocket updates are emitted from projection changes, not independent state reads
- Keep `orders-next.json` ingestion unchanged, but consume into canonical event stream
- Retain append-only event logs as durable session history
- Version projection artifacts and websocket deltas with canonical event sequence
- Require atomic file write pattern (`write temp` + `rename`) for projection files

### Sub-phase split

- **7a** file projections and atomic write semantics
- **7b** snapshot projection + hash/version derivation
- **7c** websocket replay/backfill with version cursor contract (depends on 7b version cursor + hash contract being complete)

## Data Structures

- `ProjectionBundle`
- `OrderProjection`
- `SnapshotView`
- `ProjectionHash`
- `ProjectionVersion`

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Implementation | `codex` | `gpt-5.3-codex` | Projection and stream wiring is mechanical |

## Verification

### Static

- Server/UI paths depend on projection outputs only
- Snapshot and WS envelope types generated from shared schema
- Projection files use atomic writer helper only
- `go test ./... && go vet ./...`

### Runtime

- End-of-phase e2e smoke test: `pnpm test:smoke`
- Golden projection tests for canonical scenarios
- Websocket tests for initial snapshot, incremental updates, subscribe/backfill ordering
- Sequence-cursor replay tests proving no missing/duplicate deltas on reconnect
- Edge cases: rapid state churn, slow clients, reconnect + backfill replacement semantics
