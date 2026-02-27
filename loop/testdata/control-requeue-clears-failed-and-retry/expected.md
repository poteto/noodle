---
schema_version: 1
expected_failure: false
bug: false
source_hash: f86848cf995302d0d65f15d4a45340a60635aa66a4edb369464a73464b23b73b
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
        "name": "requeue-1-0-execute",
        "skill": "execute",
        "provider": "claude",
        "model": "claude-opus-4-6"
      }
    }
  }
}
```
