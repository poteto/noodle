# Stamp Processor Fixture

## Input

```ndjson
{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."}}]}}
{"type":"event_msg","timestamp":"2026-02-22T18:20:01Z","payload":{"type":"task_complete","message":"done","cost":0.42,"tokens_in":90,"tokens_out":20}}
```

## Expected Stamped

```ndjson
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
