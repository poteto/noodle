---
id: 81
created: 2026-02-28
status: active
---

# Parallel Stage Groups

Back to [[plans/index]]

## Context

Orders contain stages that execute sequentially. The scheduler cannot express "run these stages in parallel" — every stage waits for the previous one to finish. This prevents model-aware scheduling: phases 1-3 of a plan might be mechanical Codex work that can run concurrently, while phases 4-5 require Opus judgment and should run after.

## Scope

**In scope:**
- Add `group` field to Stage struct (integer, default 0)
- Dispatch all pending stages in the current group concurrently
- Advance to the next group only when all stages in the current group complete
- Track multiple active cooks per order
- Update schema docs and schedule skill
**Out of scope:**
- DAG-style `depends_on` between individual stages
- Partial group advancement (one stage done → start next group early)
- Group-level failure policies beyond "fail the order"
- Changes to how orders-next.json is promoted

## Constraints

The `activeCooksByOrder` map in `loop/types.go` currently maps `orderID → *cookHandle` (one cook per order). This is the deepest structural assumption — every cook spawn, completion, reconcile, and control command references it.

**Approach chosen: composite key map.** Change the map to `map[string]*cookHandle` keyed by `orderID:stageIndex` (e.g., `"68:0"`, `"68:1"`). This preserves the flat map structure while allowing multiple cooks per order. Alternative considered: `map[string][]*cookHandle` (order → slice of cooks), but requires more complex lookups and doesn't improve readability.

**Group failure policy:** When any stage in a group fails, the entire order fails (current behavior — `failStage` removes the order). Remaining active stages in the group get their sessions killed and are set to `cancelled`.

## Applicable skills

- `execute` — implementation methodology
- `go-best-practices` — Go patterns
- `testing` — test-driven verification

## Phases

1. [[plans/81-stage-groups/phase-01-types]] — Add Group field to Stage, update schema
2. [[plans/81-stage-groups/phase-02-tracking]] — Multi-cook tracking per order
3. [[plans/81-stage-groups/phase-03-dispatch]] — Group-aware dispatch in dispatchableStages
4. [[plans/81-stage-groups/phase-04-completion]] — Group-aware advancement and failure
5. [[plans/81-stage-groups/phase-05-reconcile]] — Crash recovery with parallel groups
6. [[plans/81-stage-groups/phase-06-skill]] — Update schedule skill and control commands

## Verification

```
go test ./... && go vet ./...
sh scripts/lint-arch.sh
pnpm test:smoke
```
