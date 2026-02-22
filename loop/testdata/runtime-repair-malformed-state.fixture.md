# Loop Fixture: Malformed Runtime State Schedules Repair

## Setup
```json
{
  "queue_items": [],
  "mise_results": [
    {
      "error": "queue state malformed: invalid character '}' at byte 12"
    }
  ]
}
```

## Expected
```json
{
  "actions": {
    "repair_task_scheduled": true,
    "oops_task_scheduled": false,
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
    "runtime_repair_skill": { "equals": "debugging" },
    "runtime_repair_name": { "prefix": "repair-runtime-" },
    "runtime_repair_prompt": { "contains": "Scope: mise.build" }
  }
}
```

## Expected Error

```json
{
  "absent": true
}
```
