---
schema_version: 1
expected_failure: false
bug: false
source_hash: 1a4f62a74af144652d0f7d18bd6837bdd7289fedb97cd0602d16d2109108c91c
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
