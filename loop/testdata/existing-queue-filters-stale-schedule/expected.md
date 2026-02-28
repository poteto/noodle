---
schema_version: 1
expected_failure: false
bug: false
source_hash: 810ab1d93c6ac6ae7e8cda511979d1d6104f45c31ab5df214337fb1f6465c302
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
      "created_worktrees": 1,
      "first_spawn": {
        "name": "42-0-execute",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
