# Loop Fixture: Adopt Running Runtime Repair Session

## Setup
```json
{
  "queue_items": [],
  "mise_results": [
    {
      "error": "backlog sync failed"
    }
  ],
  "running_runtime_repair_session_id": "repair-runtime-20260222-185500-1-abc123"
}
```

## Expected
```json
{
  "actions": {
    "repair_task_scheduled": false,
    "normal_task_scheduled": false
  },
  "state": {
    "runtime_repair_in_flight": true,
    "paused": true
  },
  "transitions": [
    "paused"
  ],
  "counts": {
    "spawn_calls": { "eq": 0 },
    "runtime_repair_spawn_calls": { "eq": 0 },
    "normal_spawn_calls": { "eq": 0 },
    "created_worktrees": { "eq": 0 }
  }
}
```

## Expected Error

```json
{
  "absent": true
}
```
