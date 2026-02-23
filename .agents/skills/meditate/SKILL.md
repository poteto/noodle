---
name: meditate
description: Periodic brain cleanup. Prunes stale notes, extracts principles from patterns, and cleans .noodle/ state.
noodle:
  blocking: false
  schedule: "Periodically after several reflect cycles have accumulated"
---

# Meditate

Single-agent brain cleanup. Three activities: prune, extract, housekeep.

**Quality bar:** A note earns its place by being high-signal, high-frequency, or high-impact. Everything else is noise.

## Process

### 1. Prune

Read `brain/index.md`, then read each referenced file.

Flag notes that are:
- **Outdated** — references code, tools, or decisions that no longer exist
- **Redundant** — says the same thing as another note
- **Low-value** — fails: "Would Claude reliably get this wrong without it, AND does it come up often or cause real damage?"
- **Verbose** — same information possible in fewer words
- **Orphaned** — on disk but not linked from any index

Actions: delete, condense, or merge into the stronger note.

### 2. Extract

Scan remaining notes for recurring patterns across 2+ learnings that reveal unstated principles.

New principle requirements — all three must hold:
1. Genuinely independent (not derivable from existing principles)
2. Evidenced by 2+ separate notes or experiences
3. Actionable (changes how you'd approach future work)

Write new principles to `brain/principles/` and update `brain/principles.md` index.

### 3. Housekeep

Clean stale `.noodle/` state:
- Old debate directories (`.noodle/debates/`) for completed/abandoned tasks
- Completed session state
- Stale plan-mode artifacts in `brain/plans/`

### 4. Update indexes

Update `brain/index.md` for any files added or removed.

## Autonomous session mode

When running non-interactively (cook session): apply all high-confidence changes autonomously. For ambiguous changes (brain file deletions, principle rewrites), file a todo instead.

## Summary

```
## Meditate Summary
- Pruned: [N notes deleted, M condensed, K merged]
- Extracted: [N new principles, with one-line + evidence count each]
- Housekeep: [state files cleaned]
```
