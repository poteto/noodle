---
schema_version: 1
expected_failure: false
bug: false
source_hash: c2f0c8104915ad50f93d4060d1656ddbe3b802f84c57acfe68b80a456f16338d
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
        "name": "failing-1-0-oops",
        "skill": "oops",
        "provider": "claude",
        "model": "claude-opus-4-6"
      }
    }
  }
}
```
