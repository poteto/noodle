---
schema_version: 1
expected_failure: false
bug: false
source_hash: 1695dfb777d1f77f8ec2573d51ef5e11e2d7ab280a04086cbf26ac87a1a4844c
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
        "name": "fail-1-0-oops",
        "provider": "claude",
        "model": "claude-opus-4-6"
      }
    }
  }
}
```
