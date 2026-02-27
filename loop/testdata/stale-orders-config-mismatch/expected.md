---
schema_version: 1
expected_failure: false
bug: false
source_hash: 55e9677e4b46617b5c27dbf14607266f8b00df04a9f0fae8ec35384fe4405303
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
        "name": "stale-1-0-execute",
        "skill": "execute",
        "provider": "old-provider",
        "model": "old-model"
      }
    }
  }
}
```
