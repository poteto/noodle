# Loop Fixture: Runtime Repair Exceeds Max Attempts

## Setup
```json
{
  "queue_items": [],
  "mise_results": [
    {
      "error": "backlog sync failed"
    }
  ],
  "cycle_inputs": [
    {
      "runtime_repair_session_status": "failed"
    },
    {
      "runtime_repair_session_status": "failed"
    },
    {
      "runtime_repair_session_status": "failed"
    }
  ]
}
```

## Expected
```json
{
  "step_errors": [
    {
      "absent": true
    },
    {
      "absent": true
    },
    {
      "contains": "runtime issue unresolved after 3 repair attempt(s)"
    }
  ],
  "actions": {
    "repair_task_scheduled": true,
    "normal_task_scheduled": false
  },
  "state": {
    "runtime_repair_in_flight": false,
    "paused": true
  },
  "transitions": [
    "paused",
    "paused",
    "paused",
    "paused"
  ],
  "counts": {
    "spawn_calls": { "eq": 3 },
    "runtime_repair_spawn_calls": { "eq": 3 },
    "normal_spawn_calls": { "eq": 0 },
    "created_worktrees": { "eq": 3 }
  }
}
```

## Expected Error

```json
{
  "absent": true
}
```
