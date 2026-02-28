---
schema_version: 1
expected_failure: false
bug: false
source_hash: e5807ef99ea2a64f80aac7b2baab7ed68c8b53b534bcdc36e819f1df05168b0c
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
        "name": "order-new-0-execute",
        "provider": "claude",
        "model": "claude-opus-4-6"
      }
    },
    "state-02": {
      "transition": "running",
      "normal_task_scheduled": true,
      "spawn_calls": 1,
      "normal_spawn_calls": 1,
      "created_worktrees": 1,
      "first_spawn": {
        "name": "order-new-0-execute",
        "provider": "claude",
        "model": "claude-opus-4-6"
      }
    }
  }
}
```
