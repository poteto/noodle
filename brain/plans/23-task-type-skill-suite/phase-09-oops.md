Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 9: Oops — User-Project Infrastructure Fix

## Goal

Create `.agents/skills/oops/SKILL.md` as a full task-type skill. Oops fixes user-project infrastructure failures — broken tests, build failures, environment drift — NOT Noodle internals (that's repair/debugging).

## Current State

- No `.agents/skills/oops/` exists

## Patterns to Incorporate

From **Operator**: Decompose → Implement → Verify → Commit, lint-before-commit.

## Principles

- [[principles/fix-root-causes]] — trace to root cause, never paper over symptoms
- [[principles/suspect-state-before-code]] — on restart bugs, check persistent state first
- [[principles/observe-directly]] — read actual error output, don't infer

## Changes

- Create `.agents/skills/oops/SKILL.md` — **use the `skill-creator` skill**
- Add `noodle:` frontmatter: `blocking = false`
- Fix flow: Reproduce → Diagnose → Fix → Verify → Commit
- Include suspect-state-before-code as an explicit diagnostic step
- Include "check for the pattern" — if a bug exists in one place, grep for it elsewhere
- Scope boundary: user-project infrastructure only, not Noodle internals

## Data Structures

- Input: error description from the loop (what failed, error output, which session triggered it)
- Output: committed fix + brief summary of root cause

## Verification

- Static: SKILL.md has frontmatter, principles, fix flow, scope boundary
- Static: `noodle:` frontmatter exists
- Runtime: Spawn an oops session for a broken test. Verify:
  - Root cause identified (not just symptom)
  - Fix verified (tests pass after)
  - Commit describes root cause
  - Scope stays within user-project
