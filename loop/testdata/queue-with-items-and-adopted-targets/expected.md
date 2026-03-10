---
schema_version: 1
expected_failure: false
bug: false
source_hash: 2aa34c8689ecab900dde2acd532a84a191507174210671cee4e2c79ccdc9a40e
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
        "name": "42-0-execute",
        "provider": "codex",
        "model": "gpt-5.4"
      }
    }
  }
}
```
