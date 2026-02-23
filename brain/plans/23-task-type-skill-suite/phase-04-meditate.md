Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 4: Meditate — Cook-Session-First Brain Cleanup and Principle Extraction

## Goal

Rewrite `.agents/skills/meditate/SKILL.md` to be cook-session-first. Meditate is the periodic deep audit — it runs after several reflects to clean up the brain vault and extract higher-level principles from accumulated learnings. The current version is effective but too complex for a cook session (4 sequential agent teams, snapshot scripts, 7-step process).

The core job is two things: (1) prune stale/redundant brain notes, and (2) find patterns across multiple learnings that reveal unstated principles.

## Current State

- `.agents/skills/meditate/SKILL.md` — well-designed but heavy. Uses 4 specialized agents (auditor, synthesizer, distiller, skill reviewer), custom snapshot scripts, and a 7-step orchestration process. Works well interactively but consumes significant context and cost in a cook session.

## Principles

- [[principles/subtract-before-you-add]] — deletion is the primary output. A lean brain outperforms a comprehensive but bloated one.
- [[principles/encode-lessons-in-structure]] — look for recurring brain notes that could become lint rules or skill instructions
- [[principles/redesign-from-first-principles]] — check if brain notes still reflect the current architecture
- [[principles/guard-the-context-window]] — the meditate skill itself must be lean

## Changes

- Rewrite `.agents/skills/meditate/SKILL.md` — **use the `skill-creator` skill**
- Cook-session-first: streamlined process that a single agent can execute without spawning 4 sub-agents
- Focus on two core activities:
  1. **Prune**: flag and remove stale, redundant, low-value, verbose brain notes
  2. **Extract**: find patterns across multiple learnings that reveal unstated principles
- Quality bar: high-signal, high-frequency, or high-impact — everything else is noise
- Drop the early-exit gate (cook sessions should always do useful work)
- Keep agent-team approach for interactive mode (where cost is less constrained)

## Data Structures

- Input: brain vault contents, skill contents
- Output: brain file updates (deletions, edits, new principles, new connections), summary report

## Verification

- Static: SKILL.md has frontmatter, principles, streamlined two-activity process, quality bar definition
- Runtime: Spawn a meditate cook session. Confirm:
  - Brain notes are actually read and evaluated (not just listed)
  - At least one stale/redundant item is identified (or explicitly states vault is healthy)
  - Any proposed new principle is evidenced by 2+ existing notes
  - No agent team spawned in autonomous mode
  - Changes are committed with clear commit messages
