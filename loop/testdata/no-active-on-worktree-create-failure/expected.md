---
schema_version: 1
expected_failure: false
bug: false
source_hash: a47f02aacb60af3d524b6a2723a69a85cfee1ae63a4ed995489a17e2e66eebd1
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
