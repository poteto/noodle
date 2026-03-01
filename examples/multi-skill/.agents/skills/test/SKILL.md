---
name: test
description: Runs the project test suite and reports results.
schedule: "After execute stages complete, to verify changes"
---

# Test

Run the project's test suite against recent changes.

## Steps

1. Identify which packages or files changed in the most recent commits.
2. Run the relevant test commands (e.g., `go test ./...`, `npm test`).
3. If tests fail, report which tests failed and why.
4. If tests pass, confirm the result.
