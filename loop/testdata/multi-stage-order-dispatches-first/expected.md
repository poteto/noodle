---
schema_version: 1
expected_failure: false
bug: false
source_hash: ea5e3b1565b131842eafbb5f3f3b19c5c43c2a6c41d9a7f34e5c5dc563760d8d
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
      "first_spawn": {
        "name": "pipeline-1-0-execute",
        "provider": "claude",
        "model": "claude-opus-4-6"
      }
    }
  }
}
```
