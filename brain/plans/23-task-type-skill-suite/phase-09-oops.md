Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 9: Oops + Repair — Infrastructure Fix Skills

## Goal

Create two task-type skills that share the same fix methodology but differ in scope:

- `.agents/skills/oops/SKILL.md` — fixes **user-project** infrastructure failures (broken tests, build failures, environment drift)
- `.agents/skills/repair/SKILL.md` — fixes **Noodle-internal** infrastructure failures (stale queue, config errors, tmux issues, missing skills)

Both use the debugging utility skill for root-cause methodology. Repair is blocking (Noodle must be healthy before scheduling other work); oops is non-blocking.

## Current State

- No `.agents/skills/oops/` exists
- No `.agents/skills/repair/` exists (currently hardcoded as `TaskKeyRepair` in registry, routes to debugging skill)

## Patterns to Incorporate

From **Operator**: Decompose → Implement → Verify → Commit, lint-before-commit.

## Principles

- [[principles/fix-root-causes]] — trace to root cause, never paper over symptoms
- [[principles/suspect-state-before-code]] — on restart bugs, check persistent state first
- [[principles/observe-directly]] — read actual error output, don't infer

## Changes

### Oops skill

- Create `.agents/skills/oops/SKILL.md` — **use the `skill-creator` skill**
- Add `noodle:` frontmatter: `blocking = false`
- Fix flow: Reproduce → Diagnose → Fix → Verify → Commit
- Include suspect-state-before-code as an explicit diagnostic step
- Include "check for the pattern" — if a bug exists in one place, grep for it elsewhere
- Scope boundary: user-project infrastructure only, not Noodle internals

### Repair skill

- Create `.agents/skills/repair/SKILL.md` — **use the `skill-creator` skill**
- Add `noodle:` frontmatter: `blocking = true`
- Same fix flow as oops: Reproduce → Diagnose → Fix → Verify → Commit
- Scope boundary: Noodle-internal only (`.noodle/` state, queue, config, tmux sessions, skill resolution)
- Diagnostic checklist: `.noodle/` state files, queue consistency, config validity, tmux session health
- Invokes the debugging utility skill for root-cause methodology

## Data Structures

- Input: error description from the loop (what failed, error output, which session triggered it)
- Output: committed fix + brief summary of root cause

## Verification

- Static: Both SKILL.md files have frontmatter, principles, fix flow, scope boundary
- Static: Both have `noodle:` frontmatter (oops: `blocking = false`, repair: `blocking = true`)
- Runtime: Spawn an oops session for a broken test. Verify:
  - Root cause identified (not just symptom)
  - Fix verified (tests pass after)
  - Commit describes root cause
  - Scope stays within user-project
- Runtime: Spawn a repair session for a stale queue. Verify:
  - Scope stays within Noodle internals
  - `.noodle/` state is repaired
  - Queue is consistent after fix
