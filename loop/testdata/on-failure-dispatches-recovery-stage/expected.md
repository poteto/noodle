---
schema_version: 1
expected_failure: false
bug: false
source_hash: 25de38e9278a0a9a034fb3da8270079ed60a8bbfccde4a973e77216926929517
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
        "name": "fail-1-0-oops",
        "provider": "claude",
        "model": "claude-opus-4-6"
      }
    }
  }
}
```
