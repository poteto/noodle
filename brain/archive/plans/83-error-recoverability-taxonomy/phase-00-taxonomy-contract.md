# Phase 0: Taxonomy Contract

Back to [[plans/83-error-recoverability-taxonomy/overview]]

## Goal

Establish one canonical, typed backend taxonomy for ownership and recoverability
before changing behavior in individual subsystems.

## Changes

- Add a small internal package for failure classification types and helpers
- Define canonical buckets and ownership model used by all later phases
- Define mapping rules from existing behavior to canonical categories

## Data structures

- `FailureOwner` (`backend`, `scheduler_agent`, `cook_agent`, `runtime`)
- `FailureScope` (`system`, `order`, `stage`, `session`)
- `FailureRecoverability` (`hard`, `recoverable`, `degrade`)
- `FailureClass` (`backend_invariant`, `backend_recoverable`, `agent_mistake`,
  `agent_start_unrecoverable`, `agent_start_retryable`, `warning_only`)

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Architecture / judgment | `claude` | `claude-opus-4-6` | Contract design and naming coherence |

## Verification

### Static

- Type definitions compile and are used by at least one mapping table test
- No stringly-typed category constants outside the canonical package
- `go test ./... && go vet ./...`

### Runtime

- Unit tests prove class-to-recoverability mapping is deterministic
- Edge cases: unknown class defaults to explicit fallback (no silent behavior)
