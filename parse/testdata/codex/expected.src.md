---
schema_version: 1
expected_failure: false
bug: false
regression: codex
---

## Expected Events

```ndjson
{"type":"init","message":"codex session started","timestamp":"2026-02-22T18:10:00Z"}
{"type":"action","message":"$ npm test","timestamp":"2026-02-22T18:10:01Z"}
{"type":"action","message":"text:I will now update tests.","timestamp":"2026-02-22T18:10:02Z"}
{"type":"complete","message":"done","timestamp":"2026-02-22T18:10:03Z","cost_usd":0.8,"tokens_in":210,"tokens_out":55}
{"type":"error","message":"command failed: npm run lint","timestamp":"2026-02-22T18:10:04Z"}
```

## Expected Error

```json

```

