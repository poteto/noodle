---
schema_version: 1
expected_failure: false
bug: false
source_hash: b0adae42c25538db8cbede8f9476927d22c9421c853e947c18e5648d95ab90af
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
      "first_spawn": {
        "name": "schedule",
        "skill": "schedule",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
