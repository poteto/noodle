Back to [[archived_plans/23-task-type-skill-suite/overview]]

# Phase 9: Oops — Infrastructure Fix

## Goal

Create `.agents/skills/oops/SKILL.md` as a full task-type skill. Oops fixes infrastructure failures — broken tests, build failures, environment drift, stale Noodle state, config errors, tmux issues. One skill covers both user-project and Noodle-internal failures; the agent reads the error context and scopes accordingly.

## Current State

- No `.agents/skills/oops/` exists
- Repair is currently a separate hardcoded task type (`TaskKeyRepair`) that routes to the debugging skill — merged into oops

## Patterns to Incorporate

From **Operator**: Decompose → Implement → Verify → Commit, lint-before-commit.

## Principles

- [[principles/fix-root-causes]] — trace to root cause, never paper over symptoms
- [[principles/fix-root-causes]] — on restart bugs, check persistent state first
- [[principles/prove-it-works]] — read actual error output, don't infer

## Changes

- Create `.agents/skills/oops/SKILL.md` — **use the `skill-creator` skill**
- Add `noodle:` frontmatter: `blocking = false`
- Fix flow: Reproduce → Diagnose → Fix → Verify → Commit
- Include "suspect state before code" as an explicit diagnostic step (see fix-root-causes)
- Include "check for the pattern" — if a bug exists in one place, grep for it elsewhere
- Noodle-internal diagnostic checklist: `.noodle/` state files, queue consistency, config validity, tmux session health
- No separate scope boundary — the agent reads the error context and determines whether the fix is in user-project code or Noodle state. The prioritize skill decides urgency based on what broke.

## Data Structures

- Input: error description from the loop (what failed, error output, which session triggered it)
- Output: committed fix + brief summary of root cause

## Verification

- Static: SKILL.md has frontmatter, principles, fix flow, Noodle diagnostic checklist
- Static: `noodle:` frontmatter exists
- Runtime: Spawn an oops session for a broken test. Verify:
  - Root cause identified (not just symptom)
  - Fix verified (tests pass after)
  - Commit describes root cause
- Runtime: Spawn an oops session for a stale queue. Verify:
  - `.noodle/` state is repaired
  - Queue is consistent after fix
