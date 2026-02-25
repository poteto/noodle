---
schema_version: 1
expected_failure: false
bug: false
source_hash: 22055b3c9345154da5baad2694959fa19a799cb2ddba712b4eab655a70297640
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
        "name": "prioritize",
        "skill": "prioritize",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
