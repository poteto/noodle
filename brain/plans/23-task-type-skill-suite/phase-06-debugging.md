Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 6: Debugging — Utility Skill for Root-Cause Diagnosis

## Goal

Light amendments to `.agents/skills/debugging/SKILL.md`. Debugging is a **utility skill** — it doesn't map to a task type. Other skills invoke it when they encounter errors: oops sessions use it for infrastructure failures, execute sessions use it when tests fail unexpectedly, repair sessions use it for Noodle internal issues. The existing root-cause methodology is well-written; this phase just adds autonomous mode and Noodle-specific context.

## Current State

- `.agents/skills/debugging/SKILL.md` — well-structured root-cause methodology. Already principle-grounded (links to fix-root-causes, suspect-state-before-code, observe-directly). Missing autonomous session mode and Noodle-specific repair context.
- `skills/debugging/SKILL.md` — 5-line stub in Noodle defaults

## Patterns to Incorporate

From **Operator**:
- **Decompose → Implement → Verify → Commit** — structured fix flow
- **Brain update on fix** — if the fix revealed a codebase gotcha, capture it

## Principles

Already well-grounded. Add:
- [[principles/observe-directly]] — for Noodle repair: check `.noodle/` state files directly
- [[principles/never-block-on-the-human]] — repair sessions must be fully autonomous

## Changes

- Rewrite `.agents/skills/debugging/SKILL.md` — **use the `skill-creator` skill**
- Add Autonomous Session Mode section for cook-session repair
- Add Noodle-specific repair context: common failure modes (missing skills, stale queue, config errors, tmux issues)
- Add `.noodle/` state file inspection as an explicit diagnostic step
- Add brain-update step: capture the gotcha if it's novel
- Keep the existing root-cause methodology — it's well-written
- Interactive mode stays as-is (user-facing debugging)
- Update `skills/debugging/SKILL.md` stub to be consistent

## Data Structures

- Input (repair mode): error description from the loop, which component failed, error output
- Input (interactive): user-reported bug or observed failure
- Output (repair mode): committed fix + brief root cause summary
- Output (interactive): fixed code + explanation to user

## Verification

- Static: SKILL.md has frontmatter, autonomous session mode, Noodle repair context, root-cause process
- Runtime: Spawn a repair session for a simulated config error (e.g., missing skill in noodle.toml). Confirm:
  - `.noodle/` state files are inspected
  - Root cause is identified
  - Fix is applied and verified
  - No AskUserQuestion in autonomous mode
