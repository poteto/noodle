Back to [[plans/81-stage-groups/overview]]

# Phase 1 — Add Group field to Stage

**Routing:** codex / gpt-5.4

## Goal

Add the `Group` integer field to the Stage struct and update the schema documentation. No behavioral changes — this phase is pure data model.

## Changes

### `internal/orderx/types.go`

Add `Group int` to Stage struct with `json:"group,omitempty"`. Zero value (omitted from JSON) means group 0 — backward compatible with existing orders that have no group field.

### `internal/schemadoc/specs.go`

Add field doc for `orders[].stages[].group`: type `number (optional)`, description "parallel group ID; stages in the same group run concurrently, groups execute sequentially (default: 0)".

Update the constraint from "Each order is a pipeline of stages executed sequentially" to "Each order is a pipeline of stage groups executed sequentially. Stages within the same group run concurrently."

### `internal/orderx/types.go` — helper functions

Add two helpers on Order:
- `CurrentGroup() int` — returns the group number of the first stage with status `pending`, `active`, or `merging`. Returns -1 if all stages are `completed`, `failed`, or `cancelled`.
- `StagesInGroup(group int) []int` — returns stage indices belonging to a group

## Verification

- `go test ./internal/orderx/...` — existing tests still pass
- `go test ./internal/schemadoc/...` — schema test passes
- `go generate ./generate/...` — regenerate SKILL.md snapshot
- Verify `Group: 0` is omitted from JSON output (omitempty on int zero)
