Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 5: Quality — Post-Cook Quality Gate

## Goal

Create `.agents/skills/quality/SKILL.md` as the post-cook quality gate. Examines completed cook results holistically — not just "did tests pass" but "is this the right solution, done well."

## Current State

- No `.agents/skills/quality/` exists

## Patterns to Incorporate

From **CTO**: evidence-first review, principle-anchored evaluation, advocate don't block, structured assessment.
From **Manager**: verify artifacts not reports, scope discipline.

## Principles

- [[principles/trust-the-output-not-the-report]] — inspect the diff, not the agent's summary
- [[principles/prove-it-works]] — confirm the feature path works, not just that tests pass
- [[principles/outcome-oriented-execution]] — evaluate against the intended outcome

## Changes

- Create `.agents/skills/quality/SKILL.md` — **use the `skill-creator` skill**
- Add `noodle:` frontmatter: `blocking = false`
- Include `references/verdict-schema.md`
- **Read brain principles** at the start of every review. Findings must be anchored to specific principles.
- Assessment sections: Scope Check, Code Quality, Principle Compliance, Test Evidence, Runtime Verification
- Accept/reject contract with actionable feedback on rejection
- **Verdict files** — write verdict to `.noodle/quality/<session-id>.json` (flat file per session, not a subdirectory). The loop retries on rejection (same mechanism as crash retries). The prioritize skill sees rejection history in the mise brief and decides whether to retry, rescope, or surface to chef.
- **Scope-violation detection** — flag files changed that weren't part of the task
- **Create todos from findings** — non-blocking issues filed as backlog items via the configured backlog adapter with severity ratings (high/medium/low)

## Data Structures

- Input: cook session result — task description, commit hash, changed files, test output, session log
- Output: `.noodle/quality/<session-id>.json` — `{ accept, feedback, issues: [{ description, severity, principle }], scope_violations, todos_created }`

## Verification

- Static: SKILL.md has frontmatter, principles, assessment sections, verdict contract
- Static: `noodle:` frontmatter and `references/verdict-schema.md` exist
- Runtime: Spawn a quality session reviewing a completed cook. Verify:
  - Reviewer reads actual diff (git diff --stat)
  - Reviewer checks for scope violations
  - Verdict JSON is well-formed
  - Rejection includes actionable remediation steps
  - Findings reference specific brain principles
  - Non-blocking issues filed as backlog items with severity
