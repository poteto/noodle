---
id: 115
created: 2026-03-11
status: active
---

# Canonical State Convergence

Back to [[plans/index]]

## Context

Noodle now has two architectures at once. The cleaner one lives in `internal/{ingest,reducer,dispatch,projection,snapshot}` and is deterministic, typed, and heavily tested. The production execution path still runs mostly through imperative `loop/` state mutation, with canonical events acting as a shadow model rather than the source of truth.

Todo `115` exists to finish that migration. The goal is not to add a third hybrid. The goal is to make canonical state plus projection authoritative for dispatch planning, completion routing, merge advancement, recovery, and snapshots, then delete the duplicated lifecycle logic that still lives in `loop/`.

Former plan `95` ([[archive/plans/95-orders-json-ownership/overview]]) is folded into this migration rather than executed separately. While legacy `orders.json` still exists as a bridge artifact, `115` also owns the work to stop treating disk `orders.json` as live workflow authority: promotion should merge against in-memory state, normal operation should stop reloading or watching `orders.json`, any temporary integrity stamping belongs to this migration only, and the final projection cutover should retire the remaining `orders.json`-authority instructions and shims.

## Scope

**In scope:**
- Durable canonical snapshot/effect-ledger persistence on the production path
- One-time bootstrap from legacy `orders.json` / `pending-review.json` into the first durable canonical checkpoint
- Restoration of canonical event/effect identity after restart
- Scheduler promotion ingestion from `orders-next.json` into canonical order creation
- Canonical dispatch planning plus dispatch-start lifecycle ownership as the only source of truth for what can run next and what is running now
- Canonical completion routing plus review-state ownership for live and adopted sessions
- Canonical merge advancement and failure transitions
- Recovery/reconcile changes needed to rehydrate and repair canonical state safely after restart
- Pending-review state ownership inside canonical state, with restart recovery under canonical authority
- Projection-backed orders/state/snapshot generation, including the authority cutover where `orders.json` becomes pure projection output
- Server and UI migration onto projection-backed snapshots, with compatibility bridges where needed
- Deletion of superseded `loop/` lifecycle paths after equivalence is proven

**Out of scope:**
- New product features such as interactive sessions, backlog UI, or sub-agent tracking
- Runtime plugin work beyond what this migration needs to preserve existing behavior
- Redesigning the external snapshot JSON shape during the first cutover
- Rewriting the reducer into a fully asynchronous effect engine in the same plan
- Incremental websocket delta delivery or any new client/server streaming protocol beyond projection-backed full snapshots
- New provider capabilities or protocol changes

## Constraints

- Preserve restart correctness and crash-window determinism already exercised by `internal/integration/resilience_test.go`
- Keep canonical state mutation serialized on the main goroutine; no background worker may mutate canonical state directly
- Restore monotonic canonical event identity after restart; event IDs, projection versions, and effect IDs must always advance past the last durable checkpoint
- Define the migration-start bootstrap explicitly: on the first restart without a canonical checkpoint, import legacy order and pending-review state once, write the first durable checkpoint immediately, and never treat the legacy files as workflow authority again
- Treat `orders-next.json` promotion as a boundary concern; scheduler special cases and cooldown behavior must remain intact, but the promotion path itself must become canonical ingestion rather than a legacy side path
- Make dispatch-start ownership explicit during the planner cutover; avoid a period where planner inputs are canonical but `stage -> active` still depends on legacy `orders.json` writes
- Keep `/api/snapshot` and websocket `snapshot` payloads stable while moving their source onto projection-backed full snapshots
- Migrate live completion, adopted-session completion, pending-review parking, approval, request-changes, and reject together or through an explicit bridge; avoid “works until restart” splits
- Define ownership for every snapshot field before the server/UI cutover, and keep pending-review ownership unambiguous: either projected canonical state or tightly-scoped runtime enrichment backed by a canonical artifact, but not both
- Do not delete legacy write paths for lifecycle state until the read-model source has also moved; avoid a period where execution is canonical but operator surfaces are legacy-backed
- Choose one structural snapshot owner during the cutover; do not let `projection` and `internal/snapshot` grow independent topology models
- Prefer deletion after each cutover step rather than carrying long-lived compatibility layers

## Alternatives considered

1. **Big-bang loop rewrite** — replace the live loop with reducer/effect execution in one jump. Cleanest end state, highest migration risk, weakest restart proof. Rejected.
2. **Snapshot/UI first** — move the operator surface to projection before backend execution converges. Improves observability but risks a UI that reports a state the loop does not actually execute. Rejected.
3. **Incremental execution-path cutover** (chosen) — first make canonical checkpoints, legacy bootstrap, restart identity, and promotion ingestion observable on the production path, then cut over dispatch-start ownership, completion/review ownership, merge/recovery, and projection-backed full snapshots in sequence; keep external snapshot compatibility while deleting superseded legacy paths at each step. Chosen because it preserves prove-it-works and enables deletion after each phase.

## Applicable skills

- `go-best-practices` — backend lifecycle, concurrency, ordered shutdown, runtime boundaries
- `testing` — parity harnesses, fixtures, resilience coverage, regression tests
- `ts-best-practices` — snapshot contract types and field-ownership clarity during the cutover
- `react-best-practices` — client snapshot adoption without effect-heavy drift

## Phases

1. [[plans/115-canonical-state-convergence/phase-01-parity-harness]]
2. [[plans/115-canonical-state-convergence/phase-02-canonical-checkpoint-and-promotion-foundation]]
3. [[plans/115-canonical-state-convergence/phase-03-dispatch-planner-cutover]]
4. [[plans/115-canonical-state-convergence/phase-04-completion-routing-cutover]]
5. [[plans/115-canonical-state-convergence/phase-05-merge-recovery-and-pending-review-cutover]]
6. [[plans/115-canonical-state-convergence/phase-06-projection-backed-snapshot-cutover]]
7. [[plans/115-canonical-state-convergence/phase-07-legacy-deletion-and-hardening]]

## Verification

End-of-phase rule:
- every phase must run `pnpm test:smoke`
- if `pnpm test:smoke` fails, the phase is incomplete unless the failure is explicitly expected, documented in the phase brief, and covered by compensating checks

```bash
pnpm check
pnpm test:smoke
go test -race ./...
go test ./internal/integration/... ./internal/reducer/... ./internal/dispatch/... ./internal/projection/... ./internal/snapshot/... ./loop/... ./server/... ./dispatcher/...
go vet ./...
sh scripts/lint-arch.sh
cd ui && pnpm tsc --noEmit && pnpm test
```

Manual proof:
- verify the production loop now persists canonical snapshot + effect-ledger state before any execution-path cutover
- verify the first restart without a canonical checkpoint performs a one-time legacy bootstrap, writes the initial canonical checkpoint, and does not keep reading legacy files as authority afterward
- verify restart restores the next canonical event/effect identity strictly past the durable checkpoint
- verify schedule output promotion creates canonical orders and preserves cooldown/bootstrap semantics
- run a normal schedule -> dispatch -> review -> merge flow and verify projected orders/snapshot advance correctly
- rerun `pnpm test:smoke` after each phase boundary and treat unexpected smoke failures as phase blockers
- restart mid-stage and verify adopted-session recovery plus pending-review recovery block duplicate dispatch and reach the same end state
- restart with a stage in `merging` and verify reconcile converges without direct legacy order surgery
- verify `orders.json` becomes projection output only at the explicit authority-cutover phase and stays read-only afterward
- compare legacy and canonical projection hashes on representative fixture scenarios before deleting legacy paths
