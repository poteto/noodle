# Phase 2: Event Ingestion and Idempotency

Back to [[plans/82-backend-v2-first-principles/overview]]

## Goal

Normalize all external inputs into canonical events with deterministic idempotency guarantees before business logic runs.

## Changes

- Introduce ingestion boundary for control commands, scheduler outputs, and runtime completions
- Assign monotonic sequence/event IDs at ingestion boundaries
- Enforce replay-safe dedup semantics (command ID + sequence watermark)
- Remove direct state mutations from ingress handlers

## Data Structures

- `StateEvent`
- `InputEnvelope`
- `EventID`
- `AppliedEventIndex`

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Implementation | `codex` | `gpt-5.3-codex` | Mechanical boundary consolidation and dedup plumbing |

## Verification

### Static

- Ingress handlers compile against canonical event contract only
- No direct writes to canonical state from boundary handlers
- `go test ./... && go vet ./...`

### Runtime

- Replay tests: same input stream twice converges to identical state
- Duplicate command tests: repeated `control.ndjson` lines apply once
- Crash-replay tests around ack/write boundaries
- Edge cases: malformed event payloads, missing IDs, out-of-order sequence
