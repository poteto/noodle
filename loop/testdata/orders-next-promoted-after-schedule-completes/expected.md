---
schema_version: 1
expected_failure: false
bug: false
source_hash: b619e89a3e4f6e84e9e456046791d82504d3a22aea0f48ecad2aeb1f204eea83
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
        "name": "plan-42-0-execute",
        "provider": "claude",
        "model": "claude-opus-4-6"
      }
    }
  }
}
```
