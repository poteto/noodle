# Phase 2: Event Ingestion and Idempotency

Back to [[plans/82-backend-v2-first-principles/overview]]

## Goal

Normalize all external inputs into canonical events with deterministic idempotency guarantees before business logic runs.

## Changes

- Introduce ingestion boundary for control commands, scheduler outputs, and runtime completions
- Route all input through a single ingestion arbiter
- Assign monotonic sequence/event IDs only in the ingestion arbiter
- Enforce source-specific dedup semantics; remove direct state mutations from ingress handlers
- Emit operator-auditable dedup metadata in ack/projection outputs

## Data Structures

- `StateEvent`
- `InputEnvelope`
- `EventID`
- `AppliedEventIndex` (per-source)
- `DedupReason`

### Idempotency Identity Matrix

| Source | Identity key |
|--------|---------------|
| Control command | `command_id` |
| Scheduler output | `scheduler_generation_id + order_id` |
| Runtime completion | `attempt_id + terminal_status` |

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

- End-of-phase e2e smoke test: `pnpm test:smoke`
- Replay tests: same input stream twice converges to identical state
- Duplicate command tests: repeated `control.ndjson` lines apply once with explicit dedup reason
- Crash-replay tests around ingest/ack boundaries
- Edge cases: malformed payloads, missing IDs, out-of-order sequence
