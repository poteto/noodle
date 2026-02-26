---
id: 49
created: 2026-02-25
status: ready
---

# Work Orders Redesign

## Context

The queue currently models work as a flat list of independent `QueueItem` structs in `queue.json`. When the scheduler creates a pipeline (execute → quality → reflect), all items are dumped into this flat list with no structural relationship. Dependencies are implicit — inferred from ID naming conventions (`quality-29`, `reflect-29`) and shared `Plan[]` references.

This creates a cascade problem: when an item fails or is cancelled, its dependents remain in the queue as orphans. There's no mechanism to detect or clean them up. The scheduler has to rediscover the broken state on its next run.

The root cause is that the queue conflates two concerns: **what work needs to happen** (pipelines with stages) and **what's ready to dispatch** (the next thing to run). The `depends_on` bolt-on was considered and rejected — it patches the symptom without fixing the model.

## Scope

**In scope:**
- Replace `QueueItem`/`Queue` with `Order`/`Stage`/`OrdersFile` across all packages
- `orders.json` replaces `queue.json` as source of truth; `orders-next.json` replaces `queue-next.json`
- Loop advances stages mechanically (no LLM between stages)
- Failure routing — orders carry optional `OnFailure` stages that run when a stage fails instead of just cancelling remaining stages
- Quality verdict → merge integration (#65) — loop reads `.noodle/quality/` verdicts in the merge decision path
- Stage metadata — `Extra` field on Stage for arbitrary skill-specific data
- Simplified queue filtering — don't port the existing nested conditionals, simplify per #60
- All callers migrated: loop, control commands, schedule skill, snapshot, API, web UI
- No backward compatibility — old types and files deleted

**Out of scope:**
- Parallel stages within an order (stages are strictly sequential)
- Cross-order dependencies (orders are independent)
- Persisted order history/archive (terminal orders — completed or failed — are removed from orders.json immediately, same as current `skipQueueItem` behavior)
- Full event/trigger system (#66) — that builds on top of orders after this lands

## Constraints

- **Idempotent stage advancement:** Crash between marking stage completed and persisting must converge on re-run. Atomic file writes already handle this; the state machine must be safe to replay.
- **Single writer:** Only the loop writes `orders.json`. The scheduler writes `orders-next.json`. No concurrent mutation.
- **Stage ≈ old QueueItem:** A stage carries the same fields a QueueItem did (TaskKey, Prompt, Provider, Model, Runtime, Skill) plus an `Extra` field (`map[string]json.RawMessage`) for arbitrary skill-specific metadata. The Order carries the grouping fields (ID, Title, Plan, Rationale).
- **Any task type can be a stage:** Debate, execute, quality, reflect — all are valid stage task keys. The scheduler can prepend a debate stage to an order when a design question needs resolution before execution (e.g. `[debate → execute → quality → reflect]`). If the debate concludes the work shouldn't proceed, the loop cancels remaining stages naturally.
- **Failure routing:** An Order carries optional `OnFailure []Stage` — a secondary pipeline that runs when any stage fails (after retries are exhausted). If `OnFailure` is empty, the order fails and remaining stages are cancelled (current behavior). If `OnFailure` is populated, the loop switches to executing those stages instead. Example: scheduler creates `{Stages: [execute, quality, reflect], OnFailure: [debugging]}` — if quality rejects, debugging runs instead of cancelling reflect.
- **Quality verdict gates merge:** When a stage completes and the loop would normally merge its worktree, the loop first checks `.noodle/quality/<session-id>.json`. If a verdict exists and `accept == false`, the stage is treated as failed (triggering OnFailure or order failure). This makes the quality skill's verdict enforceable, not just advisory.
- **Inter-stage context flows through main branch.** Each stage runs in its own worktree, merges to main on success, and the next stage reads that output from main. No explicit context-passing field on Stage — the filesystem is the handoff mechanism. This matches the current model where quality reads the cook's diff from main after merge.

## Alternatives considered

1. **`depends_on` field on QueueItem** — Adds edges to a flat list. Requires cascade logic, dispatch gating, and DAG traversal. Bolts complexity onto a model that doesn't want it.
2. **Batch/group ID** — Simpler than DAG, but doesn't encode ordering within the group. Cascade is coarse (cancel entire group vs. selective).
3. **Just-in-time scheduling** — Scheduler only creates next step. Eliminates orphans but adds 30-60s LLM latency between every pipeline stage.
4. **Work orders (chosen)** — Scheduler creates orders with ordered stages. Loop advances mechanically. Ordering is structural (array position). Cascade = stop advancing. No new concepts beyond Order/Stage.
5. **Failure as a separate event system** — Build a full event/trigger system for failure handling. Rejected: premature. OnFailure stages on Order are simpler and cover the primary use case (quality rejection → debugging) without a new subsystem. The event system (#66) can build on top later.
6. **Quality verdict as a separate integration** — Wire verdict checking independently of work orders. Rejected: the merge path gets rewritten in this plan anyway. Integrating verdict checks here avoids touching the same code twice.

## Applicable skills

- `go-best-practices` — Go patterns for types, lifecycle, concurrency
- `testing` — fixture framework, test-driven workflow
- `react-best-practices` — React component patterns for UI phase
- `ts-best-practices` — TypeScript type safety for UI types

## Phases

1. [[plans/49-work-orders-redesign/phase-01-define-order-and-stage-types]]
2. [[plans/49-work-orders-redesign/phase-02-orders-file-i-o]]
3. [[plans/49-work-orders-redesign/phase-03-stage-lifecycle-functions]]
4. [[plans/49-work-orders-redesign/phase-04-loop-core-migration]]
5. [[plans/49-work-orders-redesign/phase-05-control-commands-and-failed-targets]]
6. [[plans/49-work-orders-redesign/phase-06-schedule-handling-and-skill-contract]]
7. [[plans/49-work-orders-redesign/phase-07-snapshot-and-api]]
8. [[plans/49-work-orders-redesign/phase-08-web-ui]]
9. [[plans/49-work-orders-redesign/phase-09-test-migration-and-cleanup]]

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

Full loop fixture suite must pass after phase 9. Individual phases verify incrementally via unit tests for new code.
