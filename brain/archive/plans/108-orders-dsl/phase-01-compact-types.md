Back to [[plans/108-orders-dsl/overview]]

# Phase 1 — Compact Wire Types and Expansion

## Goal

Define the compact JSON types the scheduler emits and the expansion function that converts them to internal `OrdersFile` structs. Pure data types + pure function — no integration yet.

## Routing

Provider: `codex` | Model: `gpt-5.4`

## Changes

### New file: `internal/orderx/compact.go`

**Types:**

- `CompactStage` — stage in the compact wire format:
  - `Do string` — task key (optional — omit for ad-hoc stages that only have a `prompt`)
  - `With string` — provider (required)
  - `Model string` — model (required)
  - `Runtime string` — optional runtime
  - `Prompt string` — optional prompt text
  - `ExtraPrompt string` — optional supplemental instructions
  - `Extra map[string]json.RawMessage` — opaque extension data
  - `Group int` — optional parallel group ID (same semantics as today)

- `CompactOrder` — order in the compact wire format:
  - `ID string` — required
  - `Title string` — optional
  - `Plan []string` — optional
  - `Rationale string` — optional
  - `Stages []CompactStage` — required

- `CompactOrdersFile` — top-level wire format:
  - `Orders []CompactOrder` — required
  - `ActionNeeded []string` — optional

**Functions:**

- `ParseCompactOrders(data []byte) (CompactOrdersFile, error)` — strict JSON parse with `DisallowUnknownFields`

- `ExpandCompactOrders(compact CompactOrdersFile) (OrdersFile, error)` — converts compact → internal:
  - `Do` → `TaskKey` (direct copy, empty string if omitted — ad-hoc stage)
  - `Do` → `Skill` (copy `Do` value to `Skill` — dispatch uses `stage.Skill` directly for methodology injection)
  - `With` → `Provider` (direct copy)
  - `Model` → `Model` (direct copy)
  - `Runtime`, `Prompt`, `ExtraPrompt`, `Extra`, `Group` → direct copy
  - All stages get `Status: StageStatusPending`
  - All orders get `Status: OrderStatusActive`

### New file: `internal/orderx/compact_test.go`

Test cases:
- Parse a simple 3-stage order → correct CompactOrdersFile
- Parse with optional fields (runtime, extra_prompt, extra, group)
- Parse with action_needed
- Reject unknown fields (strict parsing)
- Reject stage without `with`
- Reject stage without `model`
- Parse ad-hoc stage (no `do`, just `prompt` + `with` + `model`) → valid
- Reject stage with neither `do` nor `prompt` (must have at least one)
- Expand simple order → correct OrdersFile with pending/active status
- Expand preserves all pass-through fields (prompt, extra_prompt, extra, group, plan, rationale)
- Expand sets `Skill` = `Do` value (dispatch uses `stage.Skill` for methodology injection)
- Expand ad-hoc stage (no `do`) → `TaskKey` and `Skill` both empty
- Expand order without runtime → runtime left empty
- Expand empty orders array → empty OrdersFile (valid, signals nothing to schedule)

## Verification

- `go test ./internal/orderx/... -run TestCompact`
- `go vet ./internal/orderx/...`
