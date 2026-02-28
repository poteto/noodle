---
id: 83
created: 2026-02-28
status: active
---

# Error Recoverability Taxonomy

## Context

Backend errors currently exist as many local `error` returns with implicit behavior
(stop loop, retry later, degrade, or fail one order). We need a concrete,
shared model that makes recoverability explicit and predictable for operators,
agents, and future refactors.

The immediate product need is clear classification of:

- hard backend invariants and unrecoverable failures
- backend-recoverable failures
- scheduler-agent mistakes
- cook-agent mistakes
- agent-start failures (recoverable vs unrecoverable)

## Scope

**In scope:**

- Define one canonical error taxonomy for backend ownership and recoverability
- Classify `start` boundary failures (`abort`, `prompt-repair`, `warning-only`)
- Classify loop-cycle failures (`fatal-system`, `fatal-order`, `degrade-continue`)
- Classify agent-start failures (retryable, fallback, unrecoverable)
- Classify scheduler/cook agent mistakes as agent-owned failures
- Surface classification metadata in backend state/events for observability
- Normalize touched boundary error messages to failure-state wording

**Out of scope:**

- Rewriting every existing error string in one pass
- UI redesign for error rendering
- Runtime architecture rewrite (covered by plan 82)
- Backward-compatibility shim layers for old error contracts

## Constraints

- Classification must be explicit in code, not inferred from string matching
- Preserve boundary discipline: classify at boundaries, propagate typed data inward
- Error messages in touched code must describe failure state, not expectations
- No dual-path legacy adapters for classification
- Keep phase scope small (2-3 files per phase where possible)

### Alternatives considered

1. **Docs-only taxonomy**  
   Fast, but does not enforce behavior or prevent drift.
2. **Per-package ad-hoc booleans**  
   Easy incrementally, but fragments semantics and repeats mapping logic.
3. **Canonical typed taxonomy with incremental adoption (chosen)**  
   Single source of truth, enforceable contracts, gradual migration path.

## Applicable skills

- `go-best-practices`
- `testing`
- `review`

## Phases

0. [[plans/83-error-recoverability-taxonomy/phase-00-taxonomy-contract]]
1. [[plans/83-error-recoverability-taxonomy/phase-01-startup-boundary-classification]]
2. [[plans/83-error-recoverability-taxonomy/phase-02-loop-fatal-vs-recoverable]]
3. [[plans/83-error-recoverability-taxonomy/phase-03-agent-start-failure-classification]]
4. [[plans/83-error-recoverability-taxonomy/phase-04-scheduler-and-cook-mistakes]]
5. [[plans/83-error-recoverability-taxonomy/phase-05-observability-and-projections]]
6. [[plans/83-error-recoverability-taxonomy/phase-06-message-policy-and-rollout]]

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

Runtime verification:

- Run `noodle start --once` with valid config and confirm no regression in cycle behavior
- Run `noodle start` with induced failures (bad config, lock contention, invalid control line)
  and confirm expected classification bucket appears
- Trigger scheduler `orders-next` rejection and confirm classification is agent-owned + recoverable
- Trigger agent-start failure (invalid runtime/provider path) and confirm correct
  recoverability class
