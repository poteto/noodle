---
schema_version: 1
expected_failure: false
bug: false
source_hash: ed08fa5878490d2c62aee1c3261b25224e99da681b52ceb0d4f32435d2a57aef
---

## Expected

```json
{
  "contains": [
    "'codex' 'exec'",
    "'--dangerously-bypass-approvals-and-sandbox'",
    "'--skip-git-repo-check'",
    "'--json'",
    "'--model' 'gpt-5.4'",
    "< '/tmp/prompt.txt'",
    "2> '/tmp/stderr.log'"
  ],
  "omits": [
    "2>&1",
    "'--sandbox'",
    "'--ask-for-approval'"
  ]
}
```
