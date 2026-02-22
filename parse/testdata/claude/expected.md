---
schema_version: 1
expected_failure: false
bug: claude
---

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

