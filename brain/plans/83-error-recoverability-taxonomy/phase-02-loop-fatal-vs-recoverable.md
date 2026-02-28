# Phase 2: Loop Fatal vs Recoverable

Back to [[plans/83-error-recoverability-taxonomy/overview]]

## Goal

Classify loop failures into system-hard, order-hard, and degrade-continue
without changing core orchestration semantics.

## Changes

- Apply taxonomy to cycle/build/persist/reconcile paths
- Explicitly mark order-level terminal failures versus system-level fatals
- Preserve existing retry/fallback behavior while making reason class explicit

## Data structures

- `LoopFailureEnvelope`
- `CycleFailureClass`
- `OrderFailureClass`

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Implementation | `codex` | `gpt-5.3-codex` | Focused loop plumbing and tests |

## Verification

### Static

- Loop package tests cover each class (`hard`, `recoverable`, `degrade`)
- No direct string matching for recoverability decisions in migrated paths
- `go test ./... && go vet ./...`

### Runtime

- Force `orders-next` promotion error and confirm degrade-continue classification
- Force state flush failure and confirm system-hard classification
- Force stage failure and confirm order-hard (not process-hard) classification
