---
name: meditate
description: >-
  Audit and evolve the brain vault — prune outdated content, discover cross-cutting principles,
  review skills for structural encoding opportunities. Triggers: "meditate", "audit the brain",
  "prune brain", "review skills", "consolidate notes", "prune notes".
schedule: "Standalone single-stage order after several reflect cycles have accumulated. Expensive — don't over-schedule."
---

# Meditate

## Autonomous Session Mode

When running non-interactively (Cook, Oops, Repair):

- Apply all high-confidence changes autonomously. For ambiguous or risky changes (brain file deletions, principle rewrites), file a todo instead.
- Commit all changes with the `/commit` skill.

**Quality bar:** A note earns its place by being **high-signal** (Claude would reliably get this wrong without it), **high-frequency** (comes up in most sessions or most tasks of a type), or **high-impact** (getting it wrong causes significant damage or wasted work). Everything else is noise. A lean, precise brain outperforms a comprehensive but bloated one.

## Process

**Use Tasks to track progress** (TaskCreate per step, TaskUpdate to in_progress/completed).

### 1. Build snapshots

```bash
sh <skill-path>/scripts/snapshot.sh brain/ /tmp/brain-snapshot.md
sh <skill-path>/scripts/snapshot.sh .agents/skills/ /tmp/skills-snapshot.md
```

Files are delimited with `=== path/to/file.md ===` headers. Also locate the auto-memory directory (`~/.claude/projects/<project>/memory/`).

### 2. Auditor (blocking — its report feeds step 3)

Spawn `general-purpose` subagent. See `references/agents.md § Auditor` for the full prompt spec. Inputs: brain snapshot, auto-memory path, CLAUDE.md path.

Audits brain notes, CLAUDE.md, and auto-memory for staleness, redundancy, low-value content, verbosity, and orphans. Returns a categorized report.

**Early-exit gate:** If the auditor finds fewer than 3 actionable items, skip step 3 and go directly to step 4.

### 3. Reviewer (after auditor completes)

Spawn one `general-purpose` subagent. See `references/agents.md § Reviewer` for the full prompt spec. Inputs: brain snapshot, skills snapshot, auditor report, `brain/principles.md`.

Combines three concerns in a single pass:
- **Synthesis**: Proposes missing wikilinks, flags principle tensions, suggests clarifications.
- **Distillation**: Identifies recurring patterns that reveal unstated principles. New principles must be (1) independent, (2) evidenced by 2+ notes, (3) actionable.
- **Skill review**: Cross-references skills against brain principles. Finds contradictions, missed structural enforcement, redundant instructions. Audits description frontmatters for context bloat.

### 4. Review reports

Present the user with a consolidated summary. See `references/agents.md § Report Template` for the format.

### 5. Route skill-specific learnings

Check all reports for findings that belong in skill files, not `brain/`. Update the skill's SKILL.md or references/ directly. Read the skill first to avoid duplication.

### 6. Apply changes

Apply all changes directly. The user reviews the diff.

- **Outdated notes**: Update or delete
- **Redundant notes**: Merge into the stronger note, delete the weaker
- **Low-value notes**: Delete
- **Verbose notes**: Condense in place
- **New connections**: Add `[[wikilinks]]`
- **Tensions**: Reword to clarify boundaries
- **New principles**: Only from the distillation section, only if genuinely independent. Write brain files and update `brain/principles.md`
- **Merge principles**: Look for principles that are subsets or specific applications of each other — merge the narrower into the broader and pick a name that captures the combined insight
- **Bloated skill descriptions**: Tighten frontmatter descriptions — cut what Claude can infer, keep only distinctive triggers and purpose
- **CLAUDE.md issues**: Rewrite or delete
- **Stale memories**: Delete or rewrite

### 7. Housekeep

Clean stale `.noodle/` state (old debates, completed sessions, stale plan artifacts). Update `brain/index.md` for any files added or removed.

## Summary

```
## Meditate Summary
- Pruned: [N notes deleted, M condensed, K merged]
- Extracted: [N new principles, with one-line + evidence count each]
- Skill review: [N findings, M applied]
- Housekeep: [state files cleaned]
```
