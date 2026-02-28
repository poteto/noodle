# Stage Message Schema

Emitted by agents via `noodle event emit --session $NOODLE_SESSION_ID stage_message --payload '<json>'`.

```json
{
  "message": "string — content for the scheduler, natural language",
  "blocking": "boolean — if true, scheduler must decide before stage advances. Omit or null = true"
}
```

## Blocking Behavior

- **`blocking: true` (or omitted)** — loop forwards message to scheduler, does NOT auto-advance. Scheduler must issue a control command (advance, add stage, etc.) before the pipeline continues.
- **`blocking: false`** — loop auto-advances AND forwards the message to the scheduler for information. The scheduler sees it but doesn't need to act.

## Examples

Quality accept:
```json
{ "message": "All checks pass. Tests green, scope clean.", "blocking": false }
```

Quality reject:
```json
{ "message": "Rejected: 3 high issues. [1] Missing test for edge case in handleCompletion. [2] Scope violation: modified cook_merge.go outside plan phase scope. [3] Error message uses expectation form.", "blocking": true }
```

Execute complete:
```json
{ "message": "Implementation complete. 3 files changed, 2 new tests added. Ready for review.", "blocking": true }
```
