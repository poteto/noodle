---
schema_version: 1
expected_failure: false
bug: false
source_hash: 7890779834d89cc87b46412de8987f3283a547cd8a3438fbe1d306a3c037a382
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
      "active_summary_total": 1,
      "first_spawn": {
        "name": "pipeline-1-0-execute",
        "provider": "claude",
        "model": "claude-opus-4-6"
      }
    }
  }
}
```
