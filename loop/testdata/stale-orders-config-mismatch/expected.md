---
schema_version: 1
expected_failure: false
bug: false
source_hash: d680d50cfc39b497b4c93858ada550da1b7ea4343636a5fdd581f725107f2326
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
        "name": "stale-1-0-execute",
        "skill": "execute",
        "provider": "old-provider",
        "model": "old-model"
      }
    }
  }
}
```
