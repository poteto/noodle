---
schema_version: 1
expected_failure: false
bug: false
source_hash: b20c5b4b397ff49babf578ef3461d1fcc93075ab05a4970db391cd5ec72ffc5c
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
      "created_worktrees": 0,
      "active_summary_total": 1,
      "first_spawn": {
        "name": "schedule",
        "skill": "schedule",
        "provider": "codex",
        "model": "gpt-5.4"
      }
    }
  }
}
```
