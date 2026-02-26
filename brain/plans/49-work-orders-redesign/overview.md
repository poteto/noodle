---
id: 49
created: 2026-02-25
status: ready
---

# Work Orders Redesign

Covers todos #49, #59, #60, #61, #62, #63, #64, #65.

## Context

The queue currently models work as a flat list of independent `QueueItem` structs in `queue.json`. When the scheduler creates a pipeline (execute → quality → reflect), all items are dumped into this flat list with no structural relationship. Dependencies are implicit — inferred from ID naming conventions (`quality-29`, `reflect-29`) and shared `Plan[]` references.

This creates a cascade problem: when an item fails or is cancelled, its dependents remain in the queue as orphans. There's no mechanism to detect or clean them up. The scheduler has to rediscover the broken state on its next run.

The root cause is that the queue conflates two concerns: **what work needs to happen** (pipelines with stages) and **what's ready to dispatch** (the next thing to run). The `depends_on` bolt-on was considered and rejected — it patches the symptom without fixing the model.

Meanwhile, the loop has accumulated hardcoded Go logic that belongs in skills or frontmatter: a baked-in bootstrap prompt that requires recompilation to evolve, pre-filtered plan lists that second-guess the scheduler, `taskType.Key == "execute"` checks for domain skill injection, and crash paths (missing sync script, merge conflicts) that should degrade gracefully. These are symptoms of the same root cause — Go code making decisions that should be declarative.

Applying [[principles/redesign-from-first-principles]]: rather than fixing these separately, redesign as if all requirements were known from day one. The orders migration rewrites dispatch, completion, scheduling, and merge paths — the same code these fixes touch. Subtracting the hardcoded logic first ([[principles/subtract-before-you-add]]) creates a simpler substrate for the migration.

## Scope

**In scope:**
- Subtract hardcoded Go logic: bootstrap as skill file (#59), domain_skill frontmatter (#64), sync script degradation (#62)
- Replace `QueueItem`/`Queue` with `Order`/`Stage`/`OrdersFile` across all packages
- `orders.json` replaces `queue.json` as source of truth; `orders-next.json` replaces `queue-next.json`
- Loop advances stages mechanically (no LLM between stages)
- Failure routing — orders carry optional `OnFailure` stages that run when a stage fails instead of just cancelling remaining stages
- Quality verdict → merge integration (#65) — loop reads `.noodle/quality/` verdicts in the merge decision path
- Merge conflicts → pending review (#63) — park for human resolution instead of permanent failure
- Domain skill dispatch wiring (#64) — registry-driven domain_skill replaces hardcoded execute checks
- Stage metadata — `Extra` field on Stage for arbitrary skill-specific data
- Simplified queue filtering (#60) — don't port nested conditionals, simplify per orders model
- Simplified mise.json (#61) — remove `NeedsScheduling` pre-filtering, include all plans, let schedule skill decide
- All callers migrated: loop, control commands, schedule skill, snapshot, API, web UI
- No backward compatibility — old types and files deleted

**Out of scope:**
- Parallel stages within an order (stages are strictly sequential)
- Cross-order dependencies (orders are independent)
- Persisted order history/archive (terminal orders removed from orders.json immediately)
- Full event/trigger system (#66) — builds on top of orders after this lands

## Constraints

- **Idempotent stage advancement:** Crash between marking stage completed and persisting must converge on re-run. Atomic file writes already handle this; the state machine must be safe to replay. Crash-replay of `consumeOrdersNext` may re-introduce orders already merged — duplicates must be treated as "already merged" and skipped, not rejected as validation errors.
- **Single writer:** Only the loop writes `orders.json`. The scheduler writes `orders-next.json`. No concurrent mutation.
- **Stage ≈ old QueueItem:** A stage carries the same fields a QueueItem did (TaskKey, Prompt, Provider, Model, Runtime, Skill) plus an `Extra` field (`map[string]json.RawMessage`) for arbitrary skill-specific metadata. The Order carries the grouping fields (ID, Title, Plan, Rationale).
- **Any task type can be a stage:** Debate, execute, quality, reflect — all are valid stage task keys. The scheduler can prepend a debate stage to an order when a design question needs resolution before execution.
- **Failure routing:** An Order carries optional `OnFailure []Stage` — a secondary pipeline that runs when any stage fails (after retries are exhausted). If `OnFailure` is empty, the order fails and remaining stages are cancelled. If populated, the loop switches to executing those stages instead. `OnFailure` is an array (not a single stage) because failure remediation may itself be a pipeline (e.g., `[debugging, execute, quality]` to retry after investigation). Single-stage OnFailure (just `[debugging]`) is the common case — no overhead from the array representation.
- **Quality verdict gates merge:** When a stage completes and the loop would merge, it first checks `.noodle/quality/<session-id>.json`. If a verdict exists and `accept == false`, the stage is treated as failed (triggering OnFailure or order failure). This check applies in BOTH the automatic `handleCompletion` path AND the manual `controlMerge` path — the control path must not bypass the quality gate.
- **Failed targets keyed by order ID, exempt during OnFailure.** `failedTargets` is keyed by order ID. When checking dispatchability, orders in `"failing"` status (running OnFailure stages) are exempt from the `failedTargets` check — OnFailure stages must be allowed to dispatch even though the order has a failure. Only when the order is terminally removed does the caller add it to `failedTargets`, preventing the scheduler from re-creating it.
- **Merge conflicts are recoverable failures:** Merge conflicts park for pending review with a reason string, not permanent failure. The human resolves the conflict via the web UI. Same OnFailure routing applies if configured.
- **Inter-stage context flows through main branch.** Each stage runs in its own worktree, merges to main on success, and the next stage reads that output from main. No explicit context-passing field.
- **Domain skill is declarative.** Task types declare `domain_skill` in frontmatter under `noodle:`. The loop reads it from the registry at dispatch time — no hardcoded task key checks.
- **Bootstrap is a skill file.** The bootstrap prompt lives at `.agents/skills/bootstrap/SKILL.md`, loaded via the skill resolver. Missing bootstrap skill logs an actionable error. Exhaustion after N attempts logs a warning with next steps instead of silently giving up.
- **Dispatch persists active status.** `spawnCook()` must set `Stage.Status = "active"` and persist to `orders.json` before spawning the session. The in-memory `activeByTarget` map does not survive restarts — without persisted status, a restart would re-dispatch already-running stages, violating the single-writer invariant.
- **Session ID uses `:` separator.** Session IDs follow the format `orderID:stageIndex:taskKey` (e.g., `29:0:execute`). The `:` separator is unambiguous — neither orderID (numeric), stageIndex (numeric), nor taskKey (alphanumeric identifier) can contain `:`. Hyphen (`-`) was rejected because taskKey could theoretically contain hyphens, making positional parsing ambiguous.
- **Initial order status is `"active"`.** Newly created orders (from scheduler or `controlEnqueue`) have `Status: "active"` with all stages `Status: "pending"`. An order with `Status: ""` is a validation error — `NormalizeAndValidateOrders` rejects it. This prevents silent drops from `dispatchableStages` which only dispatches `"active"` or `"failing"` orders.
- **`"failing"` + empty `OnFailure` = terminal failure.** If validation strips all `OnFailure` stages from an order that is already in `"failing"` status, the order becomes non-dispatchable and non-terminal — stuck forever. `failStage` handles this: if the order is already `"failing"` and has no remaining `OnFailure` stages, it removes the order terminally (same as failing during the OnFailure pipeline).
- **Sync script failure degrades gracefully.** Missing or broken backlog adapter sync script → empty backlog with warning, not cycle crash.
- **Dispatcher preamble must update with the loop.** `dispatcher/preamble.go` tells agents which `.noodle/` files exist. When the loop switches from `queue.json` to `orders.json` (phase 5), the preamble must update in the same phase — otherwise agents read stale file references.
- **`Stage.Extra` preserved through all transformations.** `ApplyOrderRoutingDefaults` and `NormalizeAndValidateOrders` must preserve `Extra` fields on stages. These functions fill/strip top-level fields — they must not reconstruct stages from known fields only.

### Design decisions

**Where does `domain_skill` live?**
- A: Top-level frontmatter — available to all skills. Wrong — only task types need domain context.
- **B: Under `noodle:` block** (chosen) — scoped to task types. Domain skill injection is a scheduling/dispatch concern alongside `schedule` and `permissions`.
- C: Config-level mapping — separates concern from skill. Wrong — the skill should declare what domain context it needs.

**Bootstrap skill storage:**
- A: `//go:embed` — always available, requires recompile. Contradicts goal.
- **B: Disk file in `.agents/skills/bootstrap/`** (chosen) — evolvable, created by first-run scaffolding. If missing, actionable error.
- C: Embedded default + disk override — more complex than needed for bootstrap.

## Alternatives considered

1. **`depends_on` field on QueueItem** — Adds edges to a flat list. Requires cascade logic, dispatch gating, and DAG traversal. Bolts complexity onto a model that doesn't want it.
2. **Batch/group ID** — Simpler than DAG, but doesn't encode ordering within the group. Cascade is coarse.
3. **Just-in-time scheduling** — Scheduler only creates next step. Eliminates orphans but adds 30-60s LLM latency between every pipeline stage.
4. **Work orders (chosen)** — Scheduler creates orders with ordered stages. Loop advances mechanically. Ordering is structural (array position). Cascade = stop advancing.
5. **Failure as a separate event system** — Premature. OnFailure stages on Order cover the primary use case without a new subsystem. The event system (#66) can build on top later.
6. **Quality verdict as a separate integration** — The merge path gets rewritten here anyway. Integrating verdict checks avoids touching the same code twice.
7. **Separate plan for subtract/resilience items** — Rejected. Domain skill wiring, mise.json simplification, and merge conflict handling all touch the same code this plan rewrites (cook.go dispatch, schedule contract, merge path). Redesigning from first principles means including them here.

## Applicable skills

- `go-best-practices` — Go patterns for types, lifecycle, concurrency
- `testing` — fixture framework, test-driven workflow
- `skill-creator` — bootstrap skill file, schedule skill contract
- `react-best-practices` — React component patterns for UI phase
- `ts-best-practices` — TypeScript type safety for UI types

## Phases

1. ~~[[plans/49-work-orders-redesign/phase-01-subtract-go-logic]]~~ ✓ `36629cb`
2. ~~[[plans/49-work-orders-redesign/phase-02-define-order-and-stage-types]]~~ ✓ `a872fd2`
3. ~~[[plans/49-work-orders-redesign/phase-03-orders-file-i-o]]~~ ✓ `f612914`
4. ~~[[plans/49-work-orders-redesign/phase-04-stage-lifecycle-functions]]~~ ✓ `f05194c`
5. ~~[[plans/49-work-orders-redesign/phase-05-loop-core-migration]]~~ ✓ `f3bc053`
6. ~~[[plans/49-work-orders-redesign/phase-06-control-commands-and-failed-targets]]~~ ✓ `d81b531`
7. ~~[[plans/49-work-orders-redesign/phase-07-schedule-handling-and-skill-contract]]~~ ✓ `dc2303b`
8. ~~[[plans/49-work-orders-redesign/phase-08-snapshot-and-api]]~~ ✓ `f261e11`
9. ~~[[plans/49-work-orders-redesign/phase-09-web-ui]]~~ ✓ `5c055de`
10. [[plans/49-work-orders-redesign/phase-10-test-migration-and-cleanup]] — **partial**

### Phase 10 remaining work

`go build ./...` passes. `go test ./...` all green. Core migration complete. Remaining items are cosmetic cleanup:

1. ~~**Test file queue.json string references**~~ ✓ done — updated to `"orders.json"` for consistency:
   - `dispatcher/tmux_session_test.go` — schedule prompt text mentions queue.json
   - `internal/filex/filex_test.go` — uses "queue.json" as arbitrary test filename
   - `internal/stringx/stringx.go` and `stringx_test.go` — docstring/test uses queue.json as example path
   - `internal/testutil/fixturedir/fixturedir_test.go` and `sync_test.go` — fixture helpers create queue.json
   - `server/server_test.go` — creates queue.json for snapshot test
   - `cmd_debug_test.go` — references queue.json in debug paths

2. ~~**Optional package rename**~~ ✓ done — `internal/queuex/` → `internal/orderx/`.

3. **Cross-phase integration tests** — The plan specifies integration tests for success pipeline, OnFailure pipeline, merge-conflict resolution, and snapshot→Board derivation. Unit tests cover individual functions but don't test multi-cycle state continuity.

4. **New fixture scenarios** — Multi-stage order, OnFailure routing, merge conflict, failed-target stickiness, requeue recovery, domain skill dispatch (end-to-end from orders.json through dispatch).

5. **Codex review** — Three independent reviews for major issues.

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

Full loop fixture suite must pass after phase 10. Individual phases verify incrementally via unit tests for new code.
