---
schema_version: 1
expected_failure: false
bug: false
source_hash: 6e1fe5d2916745ef527b89b883c3347c2fd1a9a967738922e38471d90b50743a
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
