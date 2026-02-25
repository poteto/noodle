---
schema_version: 1
expected_failure: false
bug: false
source_hash: 0f7f8a1ee7fcc42ce8d0a83e857d121b5e511b34865c84434cd40beecfaedba6
---

## Runtime Dump

```json
{
  "states": {
    "state-01": {
      "transition": "running",
      "runtime_repair_in_flight": false,
      "repair_task_scheduled": false,
      "oops_task_scheduled": false,
      "normal_task_scheduled": true,
      "spawn_calls": 2,
      "runtime_repair_spawn_calls": 0,
      "normal_spawn_calls": 2,
      "created_worktrees": 2,
      "first_spawn": {
        "name": "42-task-42",
        "skill": "execute",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
