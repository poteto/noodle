Back to [[plans/37-skip-prioritize-with-queue/overview]]

# Phase 1: Add bootstrap skip logic

## Goal

When `prepareQueueForCycle()` reads a queue.json that already has non-schedule items, skip the schedule bootstrap and dispatch those items directly. Filter out any stale leftover schedule item so it does not occupy a concurrency slot or block real work.

## Changes

### Add `hasNonScheduleItems` helper (`loop/schedule.go`)

Small predicate that iterates `queue.Items` and returns true if any item has an ID that is not `"schedule"` (using the existing `isScheduleItem` check).

### Add `filterStaleScheduleItems` helper (`loop/schedule.go`)

Returns a new slice with schedule items removed. A leftover schedule item from a crashed previous run should not be re-dispatched -- the loop will bootstrap a fresh one if needed after the real items drain. This function is only called when non-schedule items exist, so it never strips the only item.

### Modify bootstrap condition in `prepareQueueForCycle()` (`loop/loop.go:273-284`)

Current logic:
```go
if len(queue.Items) == 0 &&
    len(l.activeByID) == 0 &&
    len(l.adoptedTargets) == 0 {
    // bootstrap or idle
}
```

New logic:
```go
if len(l.activeByID) == 0 && len(l.adoptedTargets) == 0 {
    if hasNonScheduleItems(queue) {
        // Skip bootstrap -- dispatch existing items.
        // Filter out any stale schedule item riding along.
        if filtered := filterStaleScheduleItems(queue); len(filtered.Items) != len(queue.Items) {
            filtered.GeneratedAt = l.deps.Now().UTC()
            queue = filtered
            if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
                return Queue{}, false, err
            }
        }
    } else if len(queue.Items) == 0 || !hasNonScheduleItems(queue) {
        // Queue is empty OR contains only stale schedule items — treat
        // both the same: bootstrap fresh if there is schedulable work,
        // otherwise go idle. A leftover schedule item from a crashed
        // prior run should never be dispatched; the loop will create a
        // fresh one here if needed.
        if !hasNonScheduleItems(queue) && len(queue.Items) > 0 {
            // Discard the stale schedule-only queue before deciding.
            queue = Queue{GeneratedAt: l.deps.Now().UTC()}
        }
        if len(brief.Plans) == 0 && len(brief.NeedsScheduling) == 0 {
            l.state = StateIdle
            return Queue{}, false, nil
        }
        queue = bootstrapScheduleQueue(l.config, "", l.deps.Now().UTC())
        if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
            return Queue{}, false, err
        }
    }
}
```

The key behavioral changes:
1. When non-schedule items exist and nothing is active/adopted, skip bootstrap and dispatch them.
2. Stale schedule items are filtered out so they do not consume a concurrency slot.
3. When the queue has only stale schedule items (no real work items), treat it the same as an empty queue — bootstrap fresh if schedulable work exists, otherwise go idle. This prevents the 60-120s delay from dispatching a stale schedule item that would just reproduce the queue.
4. The empty-queue bootstrap path is unchanged.

> **Note (adopted targets interaction):** The outer `len(l.adoptedTargets) == 0` guard means the entire skip path only fires when no targets have been adopted from a prior cycle. If adopted targets exist, the code falls through to normal dispatch regardless of queue contents. This is intentional — adopted targets imply in-flight work that the loop should not disrupt — but it means the skip optimization is limited to completely fresh starts with no carryover state.

> **Note (GeneratedAt on filter):** When `filterStaleScheduleItems` removes stale schedule items, the returned queue's `GeneratedAt` should be refreshed to `l.deps.Now().UTC()` before writing back. The filtered queue represents a new logical snapshot (not the original generation time), and downstream staleness checks rely on `GeneratedAt` being current.

## Data structures

No new types. `Queue` and `QueueItem` are unchanged.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Small, mechanical Go changes with clear spec |

## Verification

```bash
go test ./loop/... && go vet ./loop/...
```

- Existing `empty-state-should-schedule` fixture must still pass (empty queue still bootstraps).
- Manual: place a queue.json with real items in `.noodle/`, run `noodle start`, verify items dispatch immediately without a 60-120s schedule delay.
