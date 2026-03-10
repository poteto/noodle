# Phase 5: Observability and Projections

Back to [[plans/83-error-recoverability-taxonomy/overview]]

## Goal

Expose failure class and recoverability to operators and downstream consumers
without requiring string parsing.

## Changes

- Add classification metadata to loop/control acknowledgements and event payloads
- Project classification into backend files/snapshots used by server/UI
- Keep existing message text while adding typed fields as source of truth

## Data structures

- `FailureProjection`
- `ControlAckFailure`
- `EventFailureMetadata`

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Implementation | `codex` | `gpt-5.4` | File/event projection changes are mechanical |

## Verification

### Static

- Snapshot/control/event schema tests cover new failure metadata fields
- Server tests assert typed failure metadata serialization
- `go test ./... && go vet ./...`

### Runtime

- Execute failing control command and confirm typed failure metadata appears in ack
- Trigger one hard and one recoverable failure and confirm metadata differs correctly
- Confirm websocket/snapshot consumers still parse payloads successfully
