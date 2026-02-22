# Loop Fixture: Missing Sync + Empty Queue Uses Oops Fallback

## Setup
```json
{
  "queue_items": [],
  "mise_results": [
    {
      "warnings": [
        "backlog sync script missing; returning empty backlog"
      ]
    }
  ],
  "phases": {
    "debugging": "",
    "oops": "oops"
  }
}
```

## Expected
```json
{
  "actions": {
    "repair_task_scheduled": true,
    "oops_task_scheduled": true,
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
    "spawn_calls": { "eq": 1 },
    "runtime_repair_spawn_calls": { "eq": 1 },
    "normal_spawn_calls": { "eq": 0 },
    "created_worktrees": { "eq": 1 }
  },
  "routing": {
    "runtime_repair_skill": { "equals": "oops" },
    "runtime_repair_name": { "prefix": "repair-runtime-" },
    "runtime_repair_prompt": { "contains": "Scope: mise.sync" }
  }
}
```

## Expected Error

```json
{
  "absent": true
}
```
