# Backlog Sync Fixture

## Input
```ndjson
{"id":"42","title":"Fix login bug","status":"open","tags":["auth","urgent"]}
{"id":"43","title":"Add test","status":"in_progress"}
```

## Expected
```json
[
  {"id":"42","title":"Fix login bug","status":"open","tags":["auth","urgent"]},
  {"id":"43","title":"Add test","status":"in_progress","tags":[]}
]
```
