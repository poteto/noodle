Back to [[plans/81-stage-groups/overview]]

# Phase 6 — Update schedule skill and control commands

**Routing:** claude / claude-opus-4-6

## Goal

Teach the schedule skill to emit `group` fields when parallelizing stages, and verify control commands (`pause`, `cancel`, `retry`) work with multi-cook orders.

## Changes

### `.agents/skills/schedule/SKILL.md`

Add guidance for the `group` field:

- Document that `stages[].group` (integer, optional, default 0) controls parallel execution
- Stages in the same group run concurrently; groups execute sequentially
- When a plan has phases suitable for different models, the scheduler should assign concurrent phases to the same group number
- Example: phases 1-3 are mechanical Codex work → group 0; phases 4-5 require Opus judgment → group 1

Update the order example to show a multi-group order.

### `loop/cook_control.go` — control commands

Review `pauseOrder`, `cancelOrder`, `retryOrder` (or equivalent control handlers). These currently expect one active cook per order:

- **Pause**: must pause all active cooks for the order, not just one
- **Cancel**: must cancel all active cooks and all pending stages in the current group
- **Retry**: should retry from the failed stage's group (reset all stages in that group to pending)

Use `cooksByOrder(orderID)` helper from Phase 3 to find all cooks.

### `.agents/skills/schedule/SKILL.md` — fix stale content

While updating, fix known discrepancies discovered during audit:
- Runtime is `"process"`, not `"tmux"`
- Add `generated_at` field to examples
- Update event types to match current codebase

## Verification

- `go test ./loop/...` — control command tests pass with multi-cook orders
- Manual test: schedule a multi-group order, pause it, verify all active stages pause
- Manual test: cancel a multi-group order mid-execution, verify cleanup
- Review SKILL.md diff for accuracy against current schema
