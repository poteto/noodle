---
name: quality
description: Post-cook quality gate. Reviews completed cook work for correctness, scope discipline, and principle compliance. Writes verdict to .noodle/quality/.
noodle:
  blocking: false
  schedule: "After each cook session completes"
---

# Quality

Review completed cook work. Write verdict to `.noodle/quality/<session-id>.json`. See `references/verdict-schema.md` for schema.

Operate fully autonomously. Never ask the user.

## Step 0 -- Load Principles

Read `brain/principles.md` and follow every `[[wikilink]]`. These govern the review. Do NOT skip this.

## Assessment Pipeline

Run in order. Stop early and reject on any high-severity finding.

### 1. Scope Check

- Read the plan phase the cook was assigned (from the cook's initial prompt or mise context).
- Run `git diff --stat` and `git log --oneline` for the cook's commits.
- Flag files changed outside the plan phase's stated scope as `scope_violations`.

### 2. Code Quality

- `go vet ./...` and `go test ./...` must pass.
- Run `sh scripts/lint-arch.sh` if it exists.
- Check error handling follows project conventions (failure-state messages, not expectation messages).

### 3. Principle Compliance

- For each changed file, check against loaded principles.
- Common violations: bolted-on changes instead of redesign (redesign-from-first-principles), missing runtime verification (verify-runtime), unnecessarily added complexity (subtract-before-you-add).

### 4. Test Evidence

- New behavior must have new tests.
- Tests must assert outcomes, not implementation details.
- Check coverage of edge cases mentioned in the plan phase.

### 5. Runtime Verification

- If the plan phase specifies runtime checks, verify they were performed.
- "It compiles" is not verification.

## Verdict

Write `.noodle/quality/<session-id>.json` matching `references/verdict-schema.md`.

**Accept**: All checks pass, scope clean, tests present and passing.

**Reject**: Include specific actionable feedback. Reference the exact file, line context, and principle violated. The retry cook must be able to fix issues from feedback alone.

## Non-blocking Issues

Issues that don't warrant rejection (style nits, minor improvements, documentation gaps): file as backlog items via the backlog adapter. Record their IDs in `todos_created`.

## Principles

- [[trust-the-output-not-the-report]]
- [[verify-runtime]]
- [[outcome-oriented-execution]]
