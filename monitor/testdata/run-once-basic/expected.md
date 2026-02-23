---
schema_version: 1
expected_failure: false
bug: false
source_hash: 70d334f9851df7b447cd7e83c1ea55005e280c8ddb896b9a2c636fe537b988f4
---

## Run Once Dump

```json
{
  "states": {
    "state-01": {
      "returned_meta_count": 2,
      "ticket_count": 2,
      "status_by_session": {
        "cook-a": "running",
        "cook-b": "failed"
      },
      "action_by_session": {
        "cook-a": "apply patch",
        "cook-b": "run tests"
      },
      "cost_by_session": {
        "cook-a": 0.2,
        "cook-b": 0.05
      },
      "ticket_status_by_target": {
        "task-1": "active",
        "task-2": "active"
      }
    },
    "state-02": {
      "returned_meta_count": 2,
      "ticket_count": 2,
      "status_by_session": {
        "cook-a": "failed",
        "cook-b": "failed"
      },
      "action_by_session": {
        "cook-a": "apply patch",
        "cook-b": "run tests"
      },
      "cost_by_session": {
        "cook-a": 0,
        "cook-b": 0
      },
      "ticket_status_by_target": {
        "task-1": "active",
        "task-2": "active"
      }
    }
  }
}
```
