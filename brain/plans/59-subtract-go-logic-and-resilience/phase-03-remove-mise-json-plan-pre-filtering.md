Back to [[plans/59-subtract-go-logic-and-resilience/overview]]

# Phase 3: Remove mise.json plan pre-filtering

Covers: #61 (Go side + schedule skill update)

**Important: this phase includes the schedule skill update (formerly phase 4) so they ship atomically.** If the Go code removes `needs_scheduling` but the schedule skill still expects it, scheduling breaks.

## Goal

Remove the `schedulablePlanIDs` function and the `needs_scheduling` field from mise.json. Include all plans in mise.json unconditionally. Update the schedule skill to handle the unfiltered plan list. The schedule skill decides what's actionable — not Go pre-processing.

## Changes

### Go — mise package
- `mise/builder.go` — Delete `schedulablePlanIDs()` function (~lines 162-193). Remove the call site that populates `NeedsScheduling`. All plans flow through to mise.json unconditionally.
- `mise/types.go` — Remove `NeedsScheduling []int` from the `Brief` struct.
- `mise/builder_test.go` — Delete `TestSchedulablePlanIDsFallsBackToPlanID`. Add test: all plans appear in `brief.Plans` regardless of backlog status.

### Go — loop callers of NeedsScheduling
- `loop/loop.go` (~lines 337, 343, 354, 394) — Remove all references to `brief.NeedsScheduling`. The idle-vs-work check at line 394 (`len(brief.NeedsScheduling) == 0`) should use `len(brief.Plans) == 0` instead (or remove the condition — the schedule skill handles empty plans).
- `loop/queue.go` (~line 62) — Remove `NeedsScheduling` parameter pass-through to `queuex.NormalizeAndValidate`.
- `internal/queuex/queue.go` (~line 137) — Remove `schedulablePlanIDs []int` parameter from `NormalizeAndValidate`. Remove any filtering that uses it.

### Go — schema docs
- `internal/schemadoc/specs.go` (~line 61) — Remove `needs_scheduling` from queue schema documentation.

### Go — tests
- `loop/queue_audit_test.go` (~lines 357, 444) — Update `mise.Brief{}` construction to remove `NeedsScheduling` field.
- `loop/queue_test.go` (~line 164) and `internal/queuex/queue_test.go` (~line 179) — Update test calls to match new `NormalizeAndValidate` signature.
- `internal/schemadoc/render_test.go` (~line 86) — Update if it asserts on `needs_scheduling` schema content.

### Schedule skill
- `.agents/skills/schedule/SKILL.md` — Remove references to `needs_scheduling` (~lines 22, 26, 38). Update scheduling instructions: the skill reads `plans` array directly, cross-references with `backlog` to determine which plans have open work, and decides what to schedule. Use `skill-creator` skill for this change.

## Data structures

`Brief` loses `NeedsScheduling []int`. `NormalizeAndValidate` loses `schedulablePlanIDs []int` parameter. The `Plans` field becomes the sole source of plan data.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Touches many files including skill writing — needs judgment |

## Verification

### Static
- `go test ./mise/... ./loop/... ./internal/queuex/... ./internal/schemadoc/...` passes
- `go vet ./...` clean
- Grep for `schedulablePlanIDs`, `NeedsScheduling`, and `needs_scheduling` — zero hits in Go code and schedule skill

### Runtime
- Build mise.json with a mix of plans (some with done backlog items, some without) → all appear in `plans` array
