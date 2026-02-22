---
schema_version: 1
expected_failure: false
bug: false
source_hash: b8cae8a1e1ea05ec7318c88a365c7c1266fc7604ebc58811c500774713d78ef6
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
