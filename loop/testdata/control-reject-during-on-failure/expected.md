---
schema_version: 1
expected_failure: false
bug: false
source_hash: b282bba44d04811c2938f659624eff5bc4eaf97245db9baf3a8a305db0067efd
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
