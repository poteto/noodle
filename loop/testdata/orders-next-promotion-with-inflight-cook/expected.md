---
schema_version: 1
expected_failure: false
bug: false
source_hash: 08fa0cef7e366ca2a47ccb21bdab89323e1330aadfaa103f04f03924d0e5fcef
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
      "active_summary_total": 1,
      "first_spawn": {
        "name": "order-new-0-execute",
        "provider": "claude",
        "model": "claude-opus-4-6"
      }
    }
  }
}
```
