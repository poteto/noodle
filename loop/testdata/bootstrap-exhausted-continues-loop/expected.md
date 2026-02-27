---
schema_version: 1
expected_failure: false
bug: false
source_hash: 93cec1aa2b88deae85fa1a64c9491239a2da9ef57fd2c5f3bc489060191ef4e1
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
