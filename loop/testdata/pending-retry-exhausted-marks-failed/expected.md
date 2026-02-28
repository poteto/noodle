---
schema_version: 1
expected_failure: false
bug: false
source_hash: a61fc5d8af13097ba35f114c4fc760a14a4898e7acf141fa578b82da68240944
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
