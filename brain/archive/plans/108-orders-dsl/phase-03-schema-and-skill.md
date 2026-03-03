Back to [[plans/108-orders-dsl/overview]]

# Phase 3 — Schema Docs and Scheduler Skill

## Goal

Update the schema prompt (what the LLM sees when scheduling) and the scheduler skill instructions to describe the compact format. After this phase, the scheduler emits the new format.

**Sequencing:** This phase MUST land in the same commit as phase 2. The new parser rejects old-format output, and the scheduler emits old format until this phase updates it. Shipping phase 2 alone would dead-end scheduling.

## Routing

Provider: `claude` | Model: `claude-opus-4-6` (prompt design, skill authoring)

Use `skill-creator` skill when updating the scheduler skill.

## Changes

### Update: `internal/schemadoc/specs.go`

Replace the orders schema target to reflect compact wire types:

- Change `RootType` from `reflect.TypeOf(orderx.OrdersFile{})` to `reflect.TypeOf(orderx.CompactOrdersFile{})`
- Update `FieldDocs` to describe compact fields:
  - `orders[].stages[].do` — "task key (must match a task_types[].key from mise)"
  - `orders[].stages[].with` — "provider for this stage"
  - `orders[].stages[].model` — "model for this stage"
  - Remove `task_key`, `skill`, `status` (stage), `status` (order) field docs
- Update `Constraints` — remove mentions of status fields, note that skill is resolved automatically

### Update: `internal/schemadoc/specs_test.go`

Update schema rendering tests to match new field docs.

### Update: `.agents/skills/schedule/SKILL.md`

Rewrite examples and instructions:
- All JSON examples: `task_key` → `do`, `provider` → `with`, remove `skill` and `status`
- Remove "Stage Lifecycle" section about writing `"status": "pending"` (no longer relevant)
- Update "Output" section to describe the compact format
- Update "Orders Model" section — stages no longer carry status or skill

### Update: `docs/concepts/scheduling.md`

Update the orders format documentation with new field table and examples.

## Verification

- `go test ./internal/schemadoc/...`
- Manual: `go run . schema orders` and verify output shows compact format
- `go vet ./...`
- `pnpm check` (full suite — final verification)
