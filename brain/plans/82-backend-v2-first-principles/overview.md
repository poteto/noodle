---
id: 82
created: 2026-02-28
status: active
---

# Backend V2 First-Principles Rewrite

## Context

Noodle's current backend works, but accumulated structural complexity in loop state tracking, mode semantics, and projection paths increases correctness and UX risk as scale and runtime diversity grow. We now have enough operational knowledge (adoption/recovery edge cases, control API drift, projection drift, runtime differences) to redesign the backend from first principles while preserving Noodle's core product vision.

The redesign target is not "new tech"; it is a cleaner structural core:

- files remain the API
- skills remain the only extension point
- Go remains deterministic mechanics, LLMs remain judgment

## Scope

**In scope:**

- Canonical order-centric runtime state model for orchestration
- Deterministic event/reducer loop for all state mutations
- Runtime capability contract unifying process/sprites/cursor behavior
- Unified human involvement mode replacing split oversight knobs
- Projection layer for files + snapshot + websocket backed by one state source
- Schema/skill alignment and explicit cutover plan with legacy deletion

**Out of scope:**

- Database/message queue introduction
- Plugin API beyond skills
- New role taxonomy or major product workflow expansion
- Backward compatibility shims for removed config/runtime contracts

## Constraints

- Preserve "everything is a file" as the external API contract
- Preserve single-writer discipline for shared mutable files
- No dual-path legacy support after cutover (migrate, then delete)
- Cross-platform behavior (macOS/Linux/Windows) remains required
- Error messages must describe failure state, not expectations

### Alternatives Considered

1. Incremental refactor of current loop internals.
2. Full reducer-centric rewrite around canonical state/events.
3. Split control-plane service plus separate execution services.

Chosen: **(2)**.  
Reason: (1) keeps known structural debt alive; (3) adds operational complexity that conflicts with Noodle's simplicity and files-first posture.

## Public Interfaces and Contracts (Planned Changes)

1. Replace split oversight controls with one global `mode` field:
`auto | supervised | manual`.
2. Replace control action `autonomy` with `mode`; add explicit manual `dispatch`.
3. Introduce runtime capability contract (`steerable`, `polling`, `remote_sync`, `heartbeat`).
4. Make canonical backend state explicit (`State`, `OrderNode`, `StageNode`, `AttemptNode`) and project files/UI from it.
5. Keep `orders-next.json` as scheduler ingress and `orders.json` as projected agent-visible view.

## Applicable Skills

- `go-best-practices`
- `testing`
- `noodle`
- `skill-creator` (for schedule/noodle skill contract updates)

## Phases

1. [[plans/82-backend-v2-first-principles/phase-01-canonical-state-model]]
2. [[plans/82-backend-v2-first-principles/phase-02-event-ingestion-and-idempotency]]
3. [[plans/82-backend-v2-first-principles/phase-03-reducer-and-effect-engine]]
4. [[plans/82-backend-v2-first-principles/phase-04-runtime-capability-contract]]
5. [[plans/82-backend-v2-first-principles/phase-05-order-dispatch-and-completion]]
6. [[plans/82-backend-v2-first-principles/phase-06-unified-mode-and-control-api]]
7. [[plans/82-backend-v2-first-principles/phase-07-projection-layer-files-snapshot-ws]]
8. [[plans/82-backend-v2-first-principles/phase-08-skill-and-schema-alignment]]
9. [[plans/82-backend-v2-first-principles/phase-09-cutover-and-legacy-deletion]]
10. [[plans/82-backend-v2-first-principles/phase-10-scale-and-resilience-verification]]

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
pnpm --filter noodle-ui typecheck
pnpm test:smoke
```

