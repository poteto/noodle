Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 8: Meditate — Brain Cleanup and Principle Extraction

## Goal

Rewrite `.agents/skills/meditate/SKILL.md` for cook sessions. Meditate is the periodic deep audit — it runs after several reflects to clean up the brain vault and extract higher-level principles.

## Current State

- `.agents/skills/meditate/SKILL.md` — well-designed but heavy (4 agent teams, snapshot scripts, 7-step process)

## Principles

- [[principles/subtract-before-you-add]] — deletion is the primary output
- [[principles/encode-lessons-in-structure]] — look for brain notes that could become lint rules or skill instructions
- [[principles/redesign-from-first-principles]] — check if brain notes still reflect the current architecture
- [[principles/guard-the-context-window]] — the skill itself must be lean

## Changes

- Rewrite `.agents/skills/meditate/SKILL.md` — **use the `skill-creator` skill**
- Add `noodle:` frontmatter: `blocking = false`
- Streamlined process — single agent, no sub-agent teams in cook sessions
- Three core activities:
  1. **Prune**: flag and remove stale, redundant, low-value brain notes
  2. **Extract**: find patterns across multiple learnings that reveal unstated principles
  3. **Housekeep**: clean up stale `.noodle/` state (old debate directories, completed task state)
- Quality bar: high-signal, high-frequency, or high-impact — everything else is noise

## Data Structures

- Input: brain vault contents, skill contents
- Output: brain file updates (deletions, edits, new principles), summary

## Verification

- Static: SKILL.md has frontmatter, principles, three-activity process, quality bar
- Static: `noodle:` frontmatter exists
- Runtime: Spawn a meditate session. Verify:
  - Brain notes are read and evaluated (not just listed)
  - At least one stale/redundant item identified (or vault declared healthy)
  - New principles evidenced by 2+ existing notes
  - No agent teams spawned
