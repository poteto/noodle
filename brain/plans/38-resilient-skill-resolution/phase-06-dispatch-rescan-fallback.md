Back to [[plans/38-resilient-skill-resolution/overview]]

# Phase 6: Dispatch-time re-scan fallback

**Routing:** `codex` / `gpt-5.3-codex` — mechanical change, clear spec

## Goal

Belt-and-suspenders for fsnotify: if the registry is stale, force a synchronous re-scan before items are rejected. This catches cases where fsnotify missed an event or the debounce window hasn't elapsed yet.

## Design

The key constraint: unknown task types are rejected during queue normalization (`NormalizeAndValidate` in `queuex/queue.go`), which runs *before* dispatch. A pre-dispatch check is too late. The re-scan must happen at the queue validation layer.

Additionally, the current code routes *any* normalize/validate error into `handleRuntimeIssue()`, which pauses the loop for oops repair. For "unknown task type" errors specifically, we should drop the item and continue — not trigger runtime repair. This requires distinguishing "unknown task type" from other validation errors.

Combined with phase 2 (non-fatal resolution at dispatch) and phase 4 (fsnotify), the full flow becomes:
1. Queue normalization finds an unknown task type
2. Instead of immediately rejecting → force `rebuildRegistry()` + re-validate
3. Still unknown after re-scan → drop the item with audit event (phase 5 pattern), **not** runtime repair
4. At dispatch time, if skill file is still missing → soft-fail from phase 2 (session runs without methodology)

This ensures new skills added between cycles are caught at every layer.

## Changes

**Add typed error for unknown task type:**
- `internal/queuex/queue.go` — add `ErrUnknownTaskType` sentinel error. Change `NormalizeAndValidate` to wrap unknown-task-type errors with it: `fmt.Errorf("queue item %q: %w", id, ErrUnknownTaskType)`. Return all validation errors (not just the first), so the caller can distinguish unknown-task-type errors from other validation failures.

**Re-scan + drop at queue validation:**
- `loop/loop.go` — in `prepareQueueForCycle()` (or wherever `NormalizeAndValidate` is called):
  1. Run validation
  2. If errors include `ErrUnknownTaskType`: call `rebuildRegistry()` once and retry validation
  3. After retry, any remaining `ErrUnknownTaskType` errors → drop those items from queue using `auditQueue()` pattern (phase 5), write audit events, log to stderr. Do NOT route through `handleRuntimeIssue`.
  4. Non-unknown-task-type errors still route to `handleRuntimeIssue` as before (these are real problems like malformed JSON).

**Pre-dispatch skill file check:**
- `loop/cook.go` — `ensureSkillFresh(skillName string) bool` helper: tries `resolver.Resolve()`, if not found calls `rebuildRegistry()`, retries. Returns true if skill was found. Call this from `spawnCook()` before building the dispatch request as an additional safety net for skill file existence (registry says the task type exists, but the SKILL.md file may have been deleted after the last scan).
- `loop/prioritize.go` — same pre-check before dispatch. For prioritize specifically, if still not found after re-scan, fall through to bootstrap agent (phase 7).
- `loop/runtime_repair.go` — same pre-check for oops skill before dispatch. If still not found, fall through to built-in prompt (phase 3).

## Data structures

- `queuex.ErrUnknownTaskType` — new sentinel error
- Uses existing `rebuildRegistry()` from phase 4/5 and `skill.ErrNotFound` from phase 2.

## Verification

```bash
go test ./loop/... && go test ./internal/queuex/... && go vet ./...
```

Tests:
- Queue contains item with task_type matching a skill added to disk after startup (not yet in registry). Validation fails with `ErrUnknownTaskType`, triggers re-scan, retry succeeds. Item is dispatched.
- Queue contains item with genuinely nonexistent task_type. Re-scan still doesn't find it. Item is dropped with audit event. Loop does NOT pause for runtime repair.
- Queue contains item with malformed JSON. Error is NOT `ErrUnknownTaskType`. Routes to `handleRuntimeIssue` as before.
- Skill file deleted after registry rebuild but before dispatch. `ensureSkillFresh` triggers re-scan, file still missing. Dispatch proceeds with soft-fail (phase 2).
