---
id: 29
created: 2026-02-24
status: ready
---

# Backlog-Only Scheduling with Context Passthrough

## Context

Two problems, one solution:

**1. No context passthrough.** The schedule agent can't pass supplemental instructions to cooks. If it knows from `recent_history` that a previous attempt failed because tests weren't run, it has no way to tell the next cook "run tests this time." The `prompt` field is for *what* to do, `rationale` is for *why* it was scheduled — neither is for *how*.

**2. Brittle plan integration.** Noodle's Go code cross-references backlog IDs with plan IDs via `needs_scheduling` to decide what's ready. This means plans are mandatory for scheduling, the plan format is baked into Go, and the coupling between backlog/plans/mise is fragile. The plan step is also too opinionated for most users.

The fix: add an `extra_prompt` field for context passthrough (phases 1-4), then simplify to backlog-only scheduling where the backlog adapter is the single integration point and plans are optional context surfaced through backlog items (phases 5-7).

## Scope

**In (phases 1-4 — context passthrough):**
- New `ExtraPrompt string` field on `orderx.Stage` (or use existing `Stage.Extra` map)
- Inject `Scheduling context:` section in `buildCookPrompt()`
- Soft size guardrail: truncate to ~1000 chars
- Update schedule skill and schema docs
- Tests

**In (phases 5-7 — backlog-only scheduling):**
- Remove native plan reader from mise builder (`plan.ReadAll()`) — atomic with skill rewrite
- Backlog adapter protocol gains optional `plan` path field on items (new field, not migration)
- Schedule skill reads plan files from backlog items and injects phase context via `extra_prompt`
- Update loop idle gate, bootstrap scheduler instructions, schemadoc
- First-run bootstrap: no adapter → prompt user to create one with context about their work source
- Delete plans-sync from adapter; `plan` package stays (used by `noodle plan` CLI)

**Out:**
- Structured context fields (tags, key-value pairs) — free-form string is sufficient
- Hard validation rejection (truncation is enough)
- Changes to the `rationale` field's semantics
- Changing Noodle's own `brain/plans/` format — it stays, just becomes the adapter's concern
- Deleting the plan skill (`.claude/skills/plan/`) — it's an interactive Claude skill, not a Noodle adapter

## Constraints

- `extra_prompt` is free-form string, not structured. The scheduler writes natural language.
- `buildCookPrompt()` must not emit blank lines when `extra_prompt` is empty.
- Truncation happens at order read/normalize time.
- Phases 1-4 ship independently — they add value even without the backlog overhaul.
- Phases 5-7 are sequential. Phase 5 is internally atomic (plan reader removal + skill rewrite ship together). Phase 6 (bootstrap) should ship soon after — without it, first-run projects with no adapter can idle with no path forward.
- The loop idle gate (`loop/loop.go`) currently idles when `plans` are empty. After removal, it must idle on empty backlog instead — see "Schedule-nothing hot loop" below.
- Bootstrap scheduler instructions (`loop/builtin_bootstrap.go`) reference plan concepts — must be updated.
- The backlog adapter is the *only* integration point after this. A user who uses Linear writes a Linear adapter. A user who uses markdown writes a markdown adapter. Noodle core doesn't know or care.

## Known gaps

### 1. `order.Plan` fate after Phase 5

`buildCookPrompt()` (`loop/util.go:13`) takes `plan []string` from `order.Plan` and renders a plan header in the cook's prompt. Today the schedule skill populates `order.Plan` because it reads plan data from `mise.json`'s `plans[]` array.

After Phase 5, plans are gone from mise.json. The schedule skill discovers plans via `backlog[].plan` paths instead. Phase 5 must specify: the schedule skill populates `order.Plan` from the backlog item's `plan` field (the path to the overview file). This keeps `buildCookPrompt()` working without changes. The skill reads the plan files, puts paths in `order.Plan` for the prompt header, and puts phase-specific context in `stage.ExtraPrompt` for the how.

### 2. Schedule-nothing hot loop

After the idle gate changes from `len(brief.Plans) == 0` to `len(brief.Backlog) == 0`, a new failure mode appears: the user has active backlog items but the schedule skill decides none are actionable (all blocked, all in-progress, etc.). It writes no `orders-next.json`. The loop sees non-empty backlog, doesn't idle, re-spawns schedule, which again does nothing. Repeat.

Fix: the schedule skill must be able to signal "nothing to schedule right now." Two options:
- **Empty orders file:** Write `{"orders":[]}` to `orders-next.json`. The loop sees an explicit empty result and idles until the next backlog change. Requires `consumeOrdersNext` to treat an empty orders array as a "schedule ran, nothing to do" signal distinct from "no file exists."
- **Cooldown:** After a schedule session completes with no orders produced, the loop waits for a backlog-change event (adapter re-sync) or a time-based cooldown before re-spawning schedule.

The cooldown approach is simpler and doesn't change the orders protocol. Phase 5 should specify this.

### 3. Adapter emits plan ID, not path

The current adapter regex at `.noodle/adapters/main.go:23` captures only the numeric ID from `[[plans/15-bootstrap-onboarding/overview]]` → `"15"`. Phase 5 wants full relative paths like `"brain/plans/29-queue-item-context-passthrough/overview.md"`. The regex needs to capture the full slug, and the adapter needs to reconstruct the path. Phase 5's adapter section should specify the new regex and output format.

### 4. Web UI impact

If the web UI renders anything from `mise.json`'s `plans[]` array, removing it breaks the frontend. Phase 5 should include a grep for `plans` in the TypeScript/React code and update or remove any plan-dependent UI components.

### 5. Phase 3 stale naming

Phase 3 references `noodle schema queue` and `queue-next.json` — should be `noodle schema orders` and `orders-next.json` post-Plan 49.

## Architecture note (post-Plan 49)

The original plan was written against the flat `QueueItem` model. Plan 49 (work orders redesign) landed and replaced it with `Order` + `Stage` pipelines in `internal/orderx/`. Key differences:
- `internal/queuex/` → `internal/orderx/` (package renamed/replaced)
- `QueueItem` → `Order` (pipeline) containing `Stage` (unit of work)
- `Stage.Extra map[string]json.RawMessage` already exists for extension data
- `needs_scheduling` and `schedulablePlanIDs()` were never implemented / already removed
- No separate `loop.QueueItem` type — loop uses `type Stage = orderx.Stage` alias
- Schedule skill writes `orders-next.json` (not `queue-next.json`)

## Applicable skills

- `go-best-practices` — Go patterns, testing conventions
- `testing` — TDD workflow

## Phases

**Context passthrough (ships first):**
- [[plans/29-queue-item-context-passthrough/phase-01-add-extra-prompt-field-to-queueitem]]
- [[plans/29-queue-item-context-passthrough/phase-02-inject-extra-prompt-into-cook-prompt]]
- [[plans/29-queue-item-context-passthrough/phase-03-update-schedule-skill-and-schema-docs]]
- [[plans/29-queue-item-context-passthrough/phase-04-tests]]

**Backlog-only scheduling (depends on phases 1-4):**
- [[plans/29-queue-item-context-passthrough/phase-05-atomic-plan-removal-and-skill-rewrite]] — atomic: remove plan reader + add backlog plan field + rewrite schedule skill
- [[plans/29-queue-item-context-passthrough/phase-06-first-run-backlog-adapter-bootstrap]] — first-run backlog adapter bootstrap
- [[plans/29-queue-item-context-passthrough/phase-07-delete-plans-sync-and-clean-up]] — delete plans-sync and clean up

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

After phases 1-4: schedule agent can write `extra_prompt`, cooks receive it as `Scheduling context:`.

After phases 5-7: `mise.json` has no `plans` or `needs_scheduling`, backlog items optionally carry full plan paths (not just IDs), schedule skill reads plans itself and populates `order.Plan` for cook prompt headers, schedule-nothing produces empty orders file and triggers cooldown (no hot loop), `noodle start` without a backlog adapter prompts to create one, `noodle plan` CLI still works, web UI has no broken plan references.
