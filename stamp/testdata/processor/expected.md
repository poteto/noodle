---
schema_version: 1
expected_failure: false
bug: false
regression: processor
source_hash: 2f30d0af025d188011a8082058d4c3958328e34cab189ebd561b1447e835c1d5
---

## Expected Stamped

```json
{"_ts":"2026-02-22T18:20:00Z","message":{"content":[{"input":{"command":"go test ./..."},"name":"Bash","type":"tool_use"}],"role":"assistant"},"type":"assistant"}
{"_ts":"2026-02-22T18:20:00Z","payload":{"cost":0.42,"message":"done","tokens_in":90,"tokens_out":20,"type":"task_complete"},"timestamp":"2026-02-22T18:20:01Z","type":"event_msg"}
```

## Expected Events

```ndjson
{"provider":"claude","type":"action","message":"$ go test ./...","timestamp":"2026-02-22T18:20:00Z"}
{"provider":"codex","type":"complete","message":"done","timestamp":"2026-02-22T18:20:00Z","cost_usd":0.42,"tokens_in":90,"tokens_out":20}
```

## Expected Error

```json

```
