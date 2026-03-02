---
name: quality
description: Post-cook quality gate. Reviews completed cook work for correctness, scope discipline, and principle compliance. Emits stage_message events with findings.
schedule: "Follow-up stage after execute. Cross-provider review preferred — if codex executed, schedule quality on claude; if claude executed, schedule on codex."
---

# Quality

Review completed cook work. Emit a `stage_message` event with your assessment. See `references/stage-message-schema.md` for the event schema.

Operate fully autonomously. Never ask the user.

## Step 0 -- Load Principles

Read `brain/principles.md` and follow every `[[wikilink]]`. These govern the review. Do NOT skip this.

## Assessment Pipeline

Run in order. Stop early and reject on any high-severity finding.

### 1. Scope Check

- Read the plan phase the cook was assigned (from the cook's initial prompt or mise context).
- Run `git diff --stat` and `git log --oneline` for the cook's commits.
- Flag files changed outside the plan phase's stated scope as scope violations.

### 2. Code Quality

- `go vet ./...` and `go test ./...` must pass.
- Run `sh scripts/lint-arch.sh` if it exists.
- Check error handling follows project conventions (failure-state messages, not expectation messages).

### 3. Principle Compliance

- For each changed file, check against loaded principles.
- Common violations: bolted-on changes instead of redesign (redesign-from-first-principles), missing verification (prove-it-works), unnecessarily added complexity (subtract-before-you-add).

### 4. Test Evidence

- New behavior must have new tests.
- Tests must assert outcomes, not implementation details.
- Check coverage of edge cases mentioned in the plan phase.

### 5. Runtime Verification

- If the plan phase specifies runtime checks, verify they were performed.
- "It compiles" is not verification.

## Verdict

Emit a `stage_message` event with your assessment:

```sh
noodle event emit --session $NOODLE_SESSION_ID stage_message --payload '<json>'
```

**Accept**: Emit with `blocking: false`. The pipeline continues and the scheduler sees the green light.

```json
{ "message": "All checks pass. Tests green, scope clean.", "blocking": false }
```

**Reject**: Emit with `blocking: true` (or omit `blocking`). The scheduler reads findings and decides (retry, add oops stage, or abort).

```json
{ "message": "Rejected: 3 high issues. [1] Missing test for edge case in handleCompletion. [2] Scope violation: modified cook_merge.go outside plan phase scope. [3] Error message uses expectation form.", "blocking": true }
```

Write the message as natural language the scheduler can act on. Include specific file paths, line context, and principle violations so the scheduler can brief the retry cook.

## Non-blocking Issues

Issues that don't warrant rejection (style nits, minor improvements, documentation gaps): file as backlog items via the backlog adapter.

## Principles

- [[prove-it-works]]
- [[outcome-oriented-execution]]
