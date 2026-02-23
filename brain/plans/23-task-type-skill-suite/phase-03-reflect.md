Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 3: Reflect — Cook-Session-First Learning from Mistakes and Corrections

## Goal

Rewrite `.agents/skills/reflect/SKILL.md` to be cook-session-first. Reflect runs after cook sessions to capture learnings from mistakes, corrections, and human interventions. Its job is narrow: scan the session for things that went wrong or were corrected, and persist those learnings to the brain. Not routing rules, not brain organization — just learning capture.

## Current State

- `.agents/skills/reflect/SKILL.md` — interactive-focused. Presents a summary to the user. References old role-based skills (manager, director). Has an overly broad routing framework that tries to do too much.

## Principles

- [[principles/encode-lessons-in-structure]] — route learnings to where they have the most impact: lint rules > skill instructions > brain notes
- [[principles/fix-root-causes]] — if a mistake was made, capture why and how to prevent recurrence
- [[principles/guard-the-context-window]] — keep the skill lean. Cook sessions have limited context.

## Changes

- Rewrite `.agents/skills/reflect/SKILL.md` — **use the `skill-creator` skill**
- Cook-session-first: autonomous mode scans the session for mistakes/corrections, writes brain notes, commits, exits
- Remove references to old role-based skills (manager, director, operator)
- Narrow the scope: reflect captures learnings from mistakes and human corrections — that's it
- Structural enforcement check: "can this be a lint rule, a test, or a script?" before writing a brain note
- Interactive mode adds: present summary to user, ask for feedback
- Keep it lean — a reflect session should complete in under 2 minutes

## Data Structures

- Input: the session's conversation history / work output
- Output: brain file updates, skill file updates, todos filed

## Verification

- Static: SKILL.md has frontmatter, principles, focused learning-capture process
- Runtime: After a cook session that introduced a bug and fixed it, spawn a reflect session. Confirm:
  - The mistake and its root cause are captured
  - Structural enforcement was considered (could this be a lint rule?)
  - No duplicate brain notes created
  - Session completes quickly (no agent teams, no user prompts)
