Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 7: Reflect — Learning Capture

## Goal

Rewrite `.agents/skills/reflect/SKILL.md` for cook sessions. Reflect captures learnings from mistakes, corrections, and human interventions. Its scope is narrow: scan the session for things that went wrong or were corrected, persist those learnings to the brain.

## Current State

- `.agents/skills/reflect/SKILL.md` — interactive-focused, references deleted role-based skills in examples

## Principles

- [[principles/encode-lessons-in-structure]] — prefer lint rules and skill instructions over brain notes
- [[principles/guard-the-context-window]] — keep reflect lean and fast

## Changes

- Rewrite `.agents/skills/reflect/SKILL.md` — **use the `skill-creator` skill**
- Create `task.toml`: `blocking = false, review = false`
- Narrow scope: only capture mistakes and human corrections — not routing rules, not brain organization
- Structural enforcement: "can this lesson be a lint rule or skill instruction?" before writing a brain note
- One-topic-per-file brain note convention
- Remove references to deleted role-based skills (CEO, CTO, Director, Manager, Operator)

## Data Structures

- Input: completed cook session log, human corrections (if any)
- Output: brain file updates (new notes, edits to existing notes)

## Verification

- Static: SKILL.md has frontmatter, principles, narrow scope definition, structural enforcement step
- Static: `task.toml` exists
- Runtime: Spawn a reflect session after a cook. Verify:
  - Agent scans session for mistakes/corrections
  - New brain notes are one-topic-per-file
  - Agent checks "can this be a lint rule?" before writing
  - No references to deleted skills in output
