# Phase 05: Event Pipeline Performance

Back to [[plans/112-codebase-simplification-audit/overview]]

## Goal

Reduce avoidable hot-path parsing overhead in event/parse/stamp flows.

## Depends on

- [[plans/112-codebase-simplification-audit/phase-02-canonical-contract-cutline]]

## Findings in scope

- `65`, `66`, `68`

## Changes

- Reduce repeated full-line JSON decode/encode in hot path processing (`65`).
- Improve ticket materialization complexity and key lookup behavior (`66`).
- Reduce frequency of loop event truncation path rewrites/scans (`68`).

## Done when

- Hot path avoids redundant full decodes for unchanged payload portions.
- Ticket materialization scales without full global rescans for steady-state updates.
- Truncation path operates incrementally rather than full-rewrite per cycle.

## Verification

### Static
- `go test ./event ./parse ./stamp`
- `go test -race ./event ./parse ./stamp`

### Runtime
- Run large-line/perf fixtures and compare throughput/alloc baselines before and after.

## Rollback

- Keep performance refactors in isolated commits per concern (parse, ticket, truncation).
- If throughput regresses, revert individual optimization commits without affecting correctness.
