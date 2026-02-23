---
schema_version: 1
expected_failure: false
bug: false
source_hash: 8a23ac679d95241dbd61e8665fe929bdfecb907fd6c13a8c797af6262dc1a4b3
---

## Expected

```json
{
  "contains": [
    "'codex' '--ask-for-approval' 'never' 'exec'",
    "'--skip-git-repo-check'",
    "'--ask-for-approval' 'never'",
    "'--sandbox' 'workspace-write'",
    "'--json'",
    "'--model' 'gpt-5.3-codex'",
    "< '/tmp/prompt.txt'",
    "2> '/tmp/stderr.log'"
  ],
  "omits": [
    "2>&1"
  ]
}
```
