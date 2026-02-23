Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 5: Oops — User-Project Infrastructure Fix

## Goal

Create `.agents/skills/oops/SKILL.md` as a full task-type skill. Oops fixes user-project infrastructure failures — broken tests, build failures, environment drift — NOT Noodle internals (that's repair/debugging).

## Current State

- No `.agents/skills/oops/` exists

## Patterns to Incorporate

From **Operator**:
- **Decompose → Implement → Verify → Commit** — structured fix flow, not ad-hoc patching
- **Lint-before-commit** — verify the fix doesn't introduce new issues

## Principles

- [[principles/fix-root-causes]] — trace to root cause, never paper over symptoms
- [[principles/suspect-state-before-code]] — on restart bugs, check persistent state first
- [[principles/observe-directly]] — read actual error output, don't infer from context

## Changes

- Create `.agents/skills/oops/SKILL.md` — **use the `skill-creator` skill**
-- Define the fix flow: Reproduce → Diagnose → Fix → Verify → Commit
- Include suspect-state-before-code as an explicit diagnostic step
- Include "check for the pattern" — if a bug exists in one place, grep for it elsewhere
- Define scope boundary: user-project infrastructure only, not Noodle internals
- Keep it lean — oops sessions should be fast and focused


## Data Structures

- Input: error description from the loop (what failed, error output, which session triggered it)
- Output: committed fix + brief summary of root cause and what was changed

## Verification

- Static: SKILL.md has frontmatter, principles, fix flow, scope boundary definition
- Runtime: Spawn an oops session for a broken test. Confirm:
  - Root cause is identified (not just the symptom)
  - Fix is verified (tests pass after the fix)
  - Commit message describes the root cause, not just "fixed test"
  - Scope stays within user-project (no Noodle config/code changes)
