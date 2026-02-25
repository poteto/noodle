---
id: 37
created: 2026-02-24
status: ready
---

# Skip Schedule With Queue

## Context

Every fresh `noodle start` with an empty queue triggers a schedule bootstrap that takes 60-120s before any work dispatches. If a previous run left a populated queue.json with real work items, the loop ignores them because `prepareQueueForCycle()` only checks `len(queue.Items) == 0`. It never considers the case where items already exist and could be dispatched immediately.

The bootstrap decision lives in `loop/loop.go:273-284`:

```go
if len(queue.Items) == 0 &&
    len(l.activeByID) == 0 &&
    len(l.adoptedTargets) == 0 {
    if len(brief.Plans) == 0 && len(brief.NeedsScheduling) == 0 {
        l.state = StateIdle
        return Queue{}, false, nil
    }
    queue = bootstrapScheduleQueue(l.config, "", l.deps.Now().UTC())
}
```

When `queue.Items` is non-empty, the code falls through to routing defaults and dispatch -- which is already correct. The problem is strictly that restarts always start with an empty in-memory queue, read from a potentially empty queue.json, and unconditionally bootstrap even when queue.json had real items from a prior cycle.

## Scope

**In:**
- Modify `prepareQueueForCycle()` to skip the schedule bootstrap when queue.json already contains non-schedule items
- Filter out stale schedule items from a previous run's queue (a leftover `"id":"schedule"` item should not block fresh scheduling)
- Add fixture tests for the new skip path and the stale-schedule-filter path

**Out:**
- Staleness detection based on timestamps or age -- items from a previous run are safe to re-dispatch because the dispatch/spawn path is already idempotent (busy-target checks, failed-target checks, worktree dedup all exist in `planCycleSpawns`)
- Changes to `consumeQueueNext()` or the queue-next promotion path
- Changes to the schedule skill or its prompt

## Constraints

- The existing `normalizeAndValidateQueue()` call (line 260) already strips items with invalid task keys or missing fields. Items that survive validation are safe to dispatch.
- `planCycleSpawns()` already skips items whose targets are busy, adopted, or failed. Stale items referencing targets that no longer exist in the mise brief will simply never match a plan and sit harmlessly until the next schedule cycle cleans them up.
- The bootstrap condition must still fire when queue.json is truly empty (no items at all) and there is schedulable work. The change only affects the case where non-schedule items already exist.

## Phases

- [[plans/37-skip-prioritize-with-queue/phase-01-add-bootstrap-skip-logic]]
- [[plans/37-skip-prioritize-with-queue/phase-02-tests]]

## Verification

```bash
go test ./loop/... && go vet ./loop/...
```
