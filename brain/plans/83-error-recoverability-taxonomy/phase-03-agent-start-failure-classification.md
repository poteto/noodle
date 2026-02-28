# Phase 3: Agent Start Failure Classification

Back to [[plans/83-error-recoverability-taxonomy/overview]]

## Goal

Split “agent could not be started” into retryable/fallback/unrecoverable
classes and wire those classes through dispatch/spawn boundaries.

## Changes

- Classify runtime fallback paths (`non-process` -> `process`) as recoverable
- Classify runtime misconfiguration and process start failures as unrecoverable
  for the attempted session
- Preserve stage reset-to-pending behavior for retryable dispatch failures

## Data structures

- `AgentStartFailureClass`
- `DispatchFailureEnvelope`
- `RuntimeFallbackOutcome`

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Implementation | `codex` | `gpt-5.3-codex` | Dispatcher/runtime mapping is mostly mechanical |

## Verification

### Static

- Dispatcher and loop spawn tests assert class assignment for start failures
- Runtime fallback paths retain existing branch coverage
- `go test ./... && go vet ./...`

### Runtime

- Invalid runtime configured: classified unrecoverable start failure
- Non-process runtime failure with process available: classified recoverable fallback
- Process start failure: classified unrecoverable for session start
