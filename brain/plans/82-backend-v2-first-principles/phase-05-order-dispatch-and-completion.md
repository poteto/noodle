# Phase 5: Order Dispatch and Completion Engine

Back to [[plans/82-backend-v2-first-principles/overview]]

## Goal

Drive dispatch, completion, retry, and failure routing entirely from canonical order/stage state transitions.

## Changes

- Rebuild dispatch planning against canonical state + capacity + busy constraints
- Encode completion advancement and failure routing as reducer transitions
- Ensure retry lineage is tracked at attempt level
- Remove divergent behavior between "active map" and on-disk stage status

## Data Structures

- `DispatchPlan`
- `CompletionRecord`
- `RetryPolicy`
- `OrderBusyIndex`

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Implementation | `codex` | `gpt-5.3-codex` | Dispatch and completion transition migration |

## Verification

### Static

- Dispatch planner consumes only canonical state views
- Completion handlers emit events/effects instead of direct multi-path mutation
- `go test ./... && go vet ./...`

### Runtime

- Integration fixtures for sequential stages and grouped stages
- Retry exhaustion tests and failure routing tests
- Edge cases: adopted session completion, merge conflict parking, schedule stage special-casing
