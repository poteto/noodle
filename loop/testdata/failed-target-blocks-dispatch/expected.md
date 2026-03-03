---
schema_version: 1
expected_failure: false
bug: false
source_hash: 35717051ebdc40f44b08c8d62ba1748f18055e6a58eb1bd1b67f19fe7748bf87
---

## Runtime Dump

```json
{
  "states": {
    "state-01": {
      "transition": "running",
      "normal_task_scheduled": true,
      "spawn_calls": 1,
      "normal_spawn_calls": 1,
      "created_worktrees": 1,
      "active_summary_total": 1,
      "first_spawn": {
        "name": "blocked-1-0-execute",
        "provider": "claude",
        "model": "claude-opus-4-6"
      }
    }
  }
}
```
