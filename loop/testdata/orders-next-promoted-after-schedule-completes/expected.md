---
schema_version: 1
expected_failure: false
bug: false
source_hash: a795a420f7b4157955f55d36cf724b94f8d687652a34f0c5e41db8c4b4ee1881
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
        "name": "plan-42-0-execute",
        "provider": "claude",
        "model": "claude-opus-4-6"
      }
    }
  }
}
```
