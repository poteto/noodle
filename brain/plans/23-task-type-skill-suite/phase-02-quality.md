Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 2: Quality — Post-Cook Quality Gate

## Goal

Create `.agents/skills/quality/SKILL.md` as the post-cook quality gate. The quality reviewer examines a completed cook result holistically — not just "did tests pass" (the execute agent handles that) but "is this the right solution, done well."

## Current State

- `skills/quality/SKILL.md` — 10-line stub with a checklist and verdict format
- No `.agents/skills/quality/` exists

## Patterns to Incorporate

From **CTO**:
- **Evidence-first review** — lead with artifacts, not impressions
- **Principle-anchored evaluation** — check work against engineering principles
- **Advocate, don't block** — accept with feedback when issues are minor. Reject only for genuine quality risks.
- **Structured assessment** — explicit sections, not free-form narrative

From **Manager**:
- **Verify artifacts, not reports** — `git diff --stat` the actual changes, read the actual code
- **Scope discipline** — check for scope violations (extra changes beyond the task)

## Principles

- [[principles/trust-the-output-not-the-report]] — inspect the diff and runtime behavior, not the agent's summary
- [[principles/verify-runtime]] — confirm the feature path works, not just that tests pass
- [[principles/outcome-oriented-execution]] — evaluate against the intended outcome, not intermediate steps

## Changes

- Create `.agents/skills/quality/SKILL.md` — **use the `skill-creator` skill**
- This skill is autonomous-only — quality review is always a cook session, never interactive
- **Read brain principles** — the quality agent reads `brain/principles.md` and follows all linked principle files at the start of every review. Findings must be anchored to specific principles.
- Define assessment sections: Scope Check, Code Quality, Principle Compliance, Test Evidence, Runtime Verification
- Define accept/reject contract with actionable feedback on rejection
- **Rejection handling** — on rejection, the quality agent writes its verdict to `.noodle/quality/<session-id>/verdict.json`. The loop retries normally (same retry mechanism as crash retries). The prioritize skill sees quality rejection history in the mise brief and decides whether to retry, rescope, or surface to the chef — no special Go escalation logic needed.
- Include scope-violation detection — flag files changed that weren't part of the task
- **Create todos from findings** — when the quality agent identifies issues that don't warrant rejection (e.g. minor code smells, missing tests for edge cases, documentation gaps), it creates backlog items via the configured backlog adapter with a severity rating (high/medium/low). This feeds back into the prioritize cycle.

## Data Structures

- Input: cook session result — task description, commit hash, changed files, test output, session log
- Output: verdict JSON — `{ "accept": bool, "feedback": "concise rationale", "issues": [{ "description": "...", "severity": "high|medium|low", "principle": "principle-name" }], "scope_violations": ["..."], "todos_created": ["..."] }`

## Verification

- Static: SKILL.md has frontmatter, principles, assessment sections, verdict contract
- Runtime: Spawn a quality review session reviewing a completed cook. Confirm:
  - Reviewer reads the actual diff (git diff --stat)
  - Reviewer checks for scope violations
  - Verdict JSON is well-formed
  - Rejection includes actionable remediation steps
  - Acceptance includes concise positive rationale
  - Findings reference specific brain principles
  - Non-blocking issues are filed as backlog items with severity ratings
