---
schema_version: 1
expected_failure: false
bug: false
source_hash: a569e52040a4ca6809b880192217b82913bfa5846be996e3e5decb1632761b90
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
        "name": "retry-order-1-0-execute",
        "skill": "execute",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
