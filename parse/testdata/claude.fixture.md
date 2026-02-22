# Claude Parse Fixture

## Input

```ndjson
{"type":"system","subType":"init","_ts":"2026-02-22T18:00:00Z"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"go test ./..."}}]},"_ts":"2026-02-22T18:00:01Z"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Running focused tests now."}]},"_ts":"2026-02-22T18:00:02Z"}
{"type":"result","total_cost_usd":1.23,"usage":{"input_tokens":120,"output_tokens":34},"_ts":"2026-02-22T18:00:03Z"}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","name":"Bash","text":"Exit code 1: test failed","is_error":true}]},"_ts":"2026-02-22T18:00:04Z"}
```

## Expected Events

```ndjson
{"type":"init","message":"session started","timestamp":"2026-02-22T18:00:00Z"}
{"type":"action","message":"$ go test ./...","timestamp":"2026-02-22T18:00:01Z"}
{"type":"action","message":"text:Running focused tests now.","timestamp":"2026-02-22T18:00:02Z"}
{"type":"result","message":"turn complete","timestamp":"2026-02-22T18:00:03Z","cost_usd":1.23,"tokens_in":120,"tokens_out":34}
{"type":"error","message":"Bash: Exit code 1: test failed","timestamp":"2026-02-22T18:00:04Z"}
```

## Expected Error

```json
```
