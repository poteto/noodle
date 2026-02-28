Back to [[plans/29-queue-item-context-passthrough/overview]]

# Phase 5 — Remove native plan reader and rewrite schedule skill (atomic)

## Goal

Remove plan reading from Go code and simultaneously update the schedule skill to work without `plans`/`needs_scheduling`. These changes must ship together — removing plan fields from mise without updating the skill would break scheduling immediately.

This phase combines the concerns of the old phases 5, 6, and 7 into a single atomic pass.

## Changes

**`mise/builder.go`:**
- Remove the `plan.ReadAll()` call and all plan-related logic
- Remove `PlanSummary` type and conversion logic (e.g. `toPlanSummaries`)
- Remove `plans` field from the mise output
- The builder now just calls `adapter.SyncBacklog()` + gathers session state, tickets, resources, routing, task types

**`mise/types.go` (`mise.Brief`):**
- Remove `Plans` field
- Update all dependents: `loop/loop.go`, `dispatcher/preamble.go`, `mise/builder_test.go`, `loop/loop_test.go`, `loop/fixture_test.go`, `loop/bootstrap_test.go`, `loop/queue_audit_test.go`
- Update all loop test fixtures that construct `mise.Brief` with `Plans:` (grep for `Plans: []mise.PlanSummary`)

**`mise.json` schema:**
- Remove `plans[]` array

**`adapter/types.go`:**
- Add `Plan string` field to `BacklogItem` — optional, omitempty
- This is a relative path to the plan overview file (e.g., `brain/plans/29-queue-item-context-passthrough/overview.md`)
- Note: `BacklogItem` has no `Plan` field today — this is a new field, not a migration

**`adapter/sync.go`:**
- Parse and pass through the `plan` field from adapter NDJSON output

**`internal/schemadoc/specs.go`:**
- Remove plan-related FieldDocs from the mise target
- Add `backlog[].plan` FieldDoc entry (schemadoc enforces full leaf coverage — missing docs will fail validation)

**`loop/loop.go` (idle gate + plan watcher):**
- Currently idles when `len(brief.Plans) == 0` even if backlog items exist
- Update idle condition to idle on `len(brief.Backlog) == 0` — note that `mise/builder.go` already calls `filterActiveBacklog()` which strips `status: done` items, so this effectively means "no active backlog items"
- Also idle when the schedule-nothing cooldown is active (see `consumeOrdersNext` changes above) — this prevents hot-looping when backlog has items but none are actionable
- Remove the `brain/plans` directory watcher — the loop no longer needs to trigger cycles on plan file changes
- Remove the plan-change event handler that transitions from idle to running on plan file edits

**`loop/builtin_bootstrap.go`:**
- Update bootstrap scheduler instructions that hardcode `needs_scheduling`

**Noodle's own backlog adapter (`.noodle/adapters/main.go`):**
- `backlog-sync` already parses `[[plans/...]]` wikilinks from todo items, but the current regex only captures the numeric ID (e.g., `"15"`), not the full slug
- Update regex to capture the full plan slug (e.g., `29-queue-item-context-passthrough`) and reconstruct the relative path
- Emit as the `plan` field with the full relative path
- Example: `{"id":"29","title":"...","status":"open","plan":"brain/plans/29-queue-item-context-passthrough/overview.md"}`

**Default backlog adapter (`defaults/adapters/backlog-sync`):**
- The default shell script currently strips `[[plans/...]]` wikilinks from titles (line 36: `s/\[\[plans\/[^]]+\]\]//g`)
- Update it to extract the plan path into a `plan` JSON field before stripping from title
- Add adapter test fixtures covering items with and without `plan` fields (`adapter/fixture_test.go`, `adapter/testdata/`)
- Update `defaults_adapters_test.go` assertions — existing test has plan wikilink input but doesn't assert `plan` output field

**`.agents/skills/schedule/SKILL.md`:**
- When a backlog item has a `plan` field, read the plan overview and phase files
- Determine the next unfinished phase (first phase not marked done)
- Schedule an `execute` task for that phase
- Populate `order.Plan` with the plan file path(s) — this feeds `buildCookPrompt()`'s plan header, which still reads `order.Plan`. Without this, cooks lose plan context in their prompt header.
- Use `extra_prompt` to inject: the plan overview context, the specific phase brief, and any cross-phase dependencies
- When a backlog item has no `plan` field, schedule it as a simple task
- **Nothing-to-schedule signal:** When the skill determines no backlog items are actionable (all blocked, all in-progress, etc.), it must still write `orders-next.json` with an empty orders array (`{"orders":[]}`). This prevents the loop from re-spawning schedule immediately. Without this, the loop sees non-empty backlog, doesn't idle, and hot-loops schedule spawns.
- This moves plan intelligence from Go code into the LLM

**`loop/orders.go` (`consumeOrdersNext`):**
- After consuming an `orders-next.json` with an empty `orders` array, set a "schedule ran, nothing to do" flag. The loop uses this to suppress schedule re-spawn until the next backlog-change event (adapter re-sync) or a configurable cooldown (default: 5 minutes).
- Existing behavior (no `orders-next.json` file = schedule hasn't run) is unchanged.

**Web UI check:**
- Grep for `plans` in `ui/` TypeScript/React code. If any component renders plan data from `mise.json`, update or remove it. The plan data is no longer in mise.json after this phase.

## Routing

| Provider | Model |
|----------|-------|
| `claude` | `claude-opus-4-6` |

Judgment needed for untangling the plan reader from the builder, updating the skill, and ensuring nothing breaks.

## Verification

### Static
- `go build ./...` and `go test ./...` pass
- `go vet ./...` clean
- `noodle schema mise` no longer lists plan fields but does list `backlog[].plan`
- `grep -rn "needs_scheduling\|schedulablePlanIDs\|PlanSummary" --include="*.go"` returns no hits outside archived/test fixtures

### Runtime
- `noodle start --once` generates a `mise.json` without `plans` or `needs_scheduling`
- Backlog items with `[[plans/...]]` links have `plan` field in mise output with full relative path (not just numeric ID)
- Schedule agent reads plan files from backlog items and produces orders with `order.Plan` populated and `extra_prompt` containing phase context
- Schedule agent handles backlog items without plans as simple tasks
- Cook prompts include plan header (from `order.Plan`) when plan is present
- Loop does not idle when active backlog items exist (even without plans)
- Schedule agent writes `{"orders":[]}` when nothing is actionable — loop enters cooldown, does not hot-loop
- No plan-related components broken in web UI (or updated/removed if they existed)
