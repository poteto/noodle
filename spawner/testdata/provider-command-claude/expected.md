---
schema_version: 1
expected_failure: false
bug: false
source_hash: 4b48a345a0e70843dfe04d8c1585d61cdf2f32aaba996cee822a0e006ee678ab
---

## Expected

```json
{
  "contains": [
    "'claude'",
    "'--output-format' 'stream-json'",
    "'--model' 'claude-sonnet-4-6'",
    "'--max-turns' '5'",
    "'--max-budget-usd' '2.50'",
    "'--append-system-prompt' 'skill-system'",
    "< '/tmp/prompt.txt'",
    "2>&1"
  ],
  "omits": []
}
```
