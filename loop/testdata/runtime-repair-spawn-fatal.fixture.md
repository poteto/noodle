# Loop Fixture: Runtime Repair Spawn Failure Is Fatal

## Setup
```json
{
  "queue_items": [],
  "mise_results": [
    {
      "error": "plans sync failed"
    }
  ],
  "spawner_error": "agent unavailable"
}
```

## Expected
```json
{
  "actions": {
    "repair_task_scheduled": true,
    "normal_task_scheduled": false
  },
  "state": {
    "runtime_repair_in_flight": false,
    "paused": true
  },
  "transitions": [
    "paused"
  ],
  "counts": {
    "spawn_calls": { "eq": 1 },
    "runtime_repair_spawn_calls": { "eq": 1 },
    "normal_spawn_calls": { "eq": 0 },
    "created_worktrees": { "eq": 1 }
  },
  "routing": {
    "runtime_repair_name": { "prefix": "repair-runtime-" }
  }
}
```

## Expected Error

```json
{
  "contains": "runtime repair unavailable"
}
```
