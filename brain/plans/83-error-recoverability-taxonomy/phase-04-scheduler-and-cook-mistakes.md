# Phase 4: Scheduler and Cook Mistakes

Back to [[plans/83-error-recoverability-taxonomy/overview]]

## Goal

Treat scheduler/cook agent mistakes as explicit agent-owned failures, separate
from backend invariants and runtime failures.

## Changes

- Classify rejected scheduler outputs (`orders-next`) as scheduler-agent mistakes
- Classify quality/review/request-changes outcomes as cook-agent mistakes
- Ensure loop feedback channels carry agent ownership classification

## Data structures

- `AgentMistakeEnvelope`
- `SchedulerMistakeReason`
- `CookMistakeReason`

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Architecture / judgment | `claude` | `claude-opus-4-6` | Ownership boundaries and semantic taxonomy |

## Verification

### Static

- Tests validate scheduler rejection paths emit scheduler-agent classification
- Review/control tests validate cook-agent classification on request-changes/reject
- `go test ./... && go vet ./...`

### Runtime

- Supply invalid `orders-next` and confirm scheduler-agent recoverable classification
- Trigger quality reject and confirm cook-agent classification with order-level scope
- Confirm no backend-invariant class is emitted for agent-content mistakes
