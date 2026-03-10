---
schema_version: 1
expected_failure: false
bug: false
source_hash: 75d5558dcf7cad0cea37e2291e8aead9fc8cb9bd0bc04d0e7a67cb20c2ef49d4
---

## Runtime Dump

```json
{
  "states": {
    "state-01": {
      "transition": "running",
      "normal_task_scheduled": true,
      "spawn_calls": 2,
      "normal_spawn_calls": 2,
      "created_worktrees": 2,
      "active_summary_total": 2,
      "first_spawn": {
        "name": "order-existing-0-execute",
        "skill": "execute",
        "provider": "claude",
        "model": "claude-opus-4-6"
      }
    },
    "state-02": {
      "transition": "running",
      "normal_task_scheduled": true,
      "spawn_calls": 2,
      "normal_spawn_calls": 2,
      "created_worktrees": 2,
      "active_summary_total": 2,
      "first_spawn": {
        "name": "order-existing-0-execute",
        "skill": "execute",
        "provider": "claude",
        "model": "claude-opus-4-6"
      }
    }
  }
}
```
