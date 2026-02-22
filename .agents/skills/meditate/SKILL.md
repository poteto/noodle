---
name: meditate
description: Audit the brain Obsidian vault, CLAUDE.md, and auto-memory files for outdated information, discover connections between learnings to form higher-level principles, and review skills against brain principles to find structural encoding opportunities. Uses an agent team to parallelize the work. Use when the user says "meditate", "audit the brain", "clean up the brain", "find connections", "review skills", "prune memories", or wants to review and evolve their knowledge base.
---

# Meditate

## Cook Mode

When the cook mode marker is present in the prompt:

- **Step 7**: Apply all high-confidence changes autonomously. For ambiguous or risky changes (deletions of brain files, major principle rewrites), file a todo instead of applying.
- Do not wait for human feedback on the diff.
- Commit all changes with the `/commit` skill.

**Core principle: the brain and skills are the foundation of the entire workflow.** Every brain note, skill description, and CLAUDE.md instruction shapes how Claude behaves across all sessions. The bar for inclusion is extremely high — a note must be **high-signal** (Claude would reliably get this wrong without it), **high-frequency** (comes up in most sessions or most tasks of a type), or **high-impact** (getting it wrong causes significant damage or wasted work). Content that doesn't meet at least one of these criteria is noise that dilutes the signal. Be aggressive about cutting — a lean, precise brain outperforms a comprehensive but bloated one.

## Process

**Use Tasks to track progress.** Create a task for each step below (TaskCreate), mark each in_progress when starting and completed when done (TaskUpdate). Check TaskList after each step.

### 1. Build snapshots

Run `<skill-path>/scripts/snapshot.sh` twice — once for brain files, once for skills. This eliminates redundant Read calls across agents.

```bash
sh <skill-path>/scripts/snapshot.sh brain/ /tmp/brain-snapshot.md
sh <skill-path>/scripts/snapshot.sh .agents/skills/ /tmp/skills-snapshot.md
```

Each file is delimited with `=== path/to/file.md ===` headers (full paths for easy lookup). Pass `/tmp/brain-snapshot.md` to all agents. Pass `/tmp/skills-snapshot.md` to the skill reviewer (step 4).

Also locate the auto-memory directory (`~/.claude/projects/<project>/memory/`) and pass its path to the auditor. Memory files are small enough to read directly — no snapshot needed.

### 2. Run the auditor first

Spawn the **auditor** (subagent_type: `general-purpose`) and wait for completion — its report is input to step 3.

**Include in the auditor prompt:** the snapshot path (`/tmp/brain-snapshot.md`) and the auto-memory directory path.

**auditor**:
- Read `/tmp/brain-snapshot.md` and parse it to build a wikilink map — no individual brain file reads needed
- Use the file headers (`=== path ===`) as the on-disk file list for orphan detection
- Cross-reference each note against the current codebase state (check if referenced files, patterns, tools, or decisions still exist and are accurate) — this is the only part that requires Read/Grep/Glob calls
- Flag notes that are:
  - **Outdated**: Reference code, tools, patterns, or decisions that no longer exist or have changed
  - **Redundant**: Say the same thing as another note
  - **Low-value**: Doesn't meet the bar of high-signal, high-frequency, or high-impact. The test: "Would Claude reliably get this wrong without this note, AND does it come up often or cause real damage?" If not both, flag it.
  - **Verbose**: Could convey the same information in fewer words
  - **Orphaned**: Exist on disk (listed in snapshot headers) but are not linked from any index or other brain file
- **Audit CLAUDE.md**: Read the project's CLAUDE.md. Flag sections that are outdated (reference removed features/tools), redundant with brain notes or skill instructions, or could be condensed. Check that instructions still match actual project structure and workflows.
- **Audit auto-memory files**: Read MEMORY.md and any linked files. Apply the same staleness criteria. Also flag:
  - **Stale session state**: Entries referencing completed work, old session IDs, or finished tasks that no longer need to persist
  - **Duplicated in brain**: Entries that duplicate content already in brain notes — memory should hold only what doesn't fit in the brain vault
- Produce a report listing each flagged item with the reason and suggested action (update, merge, condense, or delete). Separate brain, CLAUDE.md, and memory findings for clarity.

### Early-exit gate

If the auditor finds fewer than 3 actionable items across brain, CLAUDE.md, and memory combined, skip Steps 3-4 and proceed directly to Step 5 with only the auditor's report. The synthesis and distillation overhead is not justified for a healthy vault.

### 3. Spawn synthesizer and distiller in parallel

After the auditor completes, spawn both agents in parallel. Include the snapshot path and the auditor's report in their prompts.

**synthesizer** (subagent_type: `general-purpose`)
- Read `/tmp/brain-snapshot.md` for all brain content (skip notes the auditor flagged for deletion)
- **Connect**: Propose missing `[[wikilinks]]` between notes that reference the same concepts.
- **Tensions**: Flag principles that appear to conflict and propose how to resolve or clarify the boundary.
- **Clarify**: Propose rewording where a note's relationship to a principle is unclear (e.g. "this is the operational procedure for X principle").
- Do NOT propose merging principles into each other — principles are intentionally independent.
- Produce a report with proposed connections, tensions, and clarifications.

**distiller** (subagent_type: `general-purpose`)
- Read `/tmp/brain-snapshot.md` for all brain content (skip notes the auditor flagged for deletion) — focus on codebase notes, preferences, plan retrospectives, and gotchas
- Look for **recurring patterns** across the detailed/operational notes that reveal the user's core engineering values, decision-making style, and architectural instincts — principles they live by but haven't yet named
- A valid new principle must be: (1) genuinely independent — not derivable from existing principles, (2) evidenced in at least 2-3 separate notes or experiences, (3) actionable — it changes how you'd approach future work
- Do NOT propose principles that are just restatements of existing ones applied to a new domain
- Produce a report listing each proposed principle with: the insight, the evidence (which notes), why it's independent, and a suggested file path under `brain/principles/`

### 4. Run skill reviewer

After the auditor, synthesizer, and distiller all complete, spawn the **skill reviewer** (subagent_type: `general-purpose`). This agent needs the full picture — all three reports plus both snapshots.

**Include in the skill reviewer prompt:** the brain snapshot path (`/tmp/brain-snapshot.md`), the skills snapshot path (`/tmp/skills-snapshot.md`), the auditor/synthesizer/distiller reports, and the brain principles index (`brain/principles.md`).

**skill reviewer**:
- Read `/tmp/skills-snapshot.md` to get all skill contents, and `/tmp/brain-snapshot.md` for brain principles
- For each skill, check against brain principles:
  - Does the skill contradict any principle?
  - Does the skill miss an opportunity to structurally enforce a principle? (per [[principles/encode-lessons-in-structure]] — can an instruction become a lint rule, script, metadata flag, or runtime check?)
  - Does the skill duplicate instructions that a mechanism already handles?
  - Is the skill missing a principle that would improve its reliability?
- Cross-reference with the auditor/synthesizer/distiller reports — if a recurring failure or pattern was identified, could it be encoded into a skill rather than (or in addition to) a brain note?
- Produce a report listing each finding with: skill name, principle(s), gap, and concrete proposal
- Prioritize proposals that replace textual instructions with structural enforcement
- When encoding a principle into a skill, translate it into specific, actionable instructions — exact commands, file paths, checklist items, or scripts.

### 5. Review reports

After all four agents complete, present the user with a summary:

```
## Audit Results — Brain
- X notes flagged (Y outdated, Z redundant, V low-value, U verbose, W orphaned)
- [list each with one-line reason]

## Audit Results — CLAUDE.md
- X sections flagged (Y outdated, Z redundant, V verbose)
- [list each with one-line reason]

## Audit Results — Memory
- X entries flagged (Y outdated, Z redundant, V stale session state, U duplicated in brain)
- [list each with one-line reason]

## Synthesis Results
- M new connections found
- T tensions identified
- [list each with one-line summary]

## Distiller Results
- P new principles proposed
- [list each with one-line summary and evidence count]

## Skill Review Results
- S skills reviewed, N findings
- [list each with: skill name, principle gap, proposed fix]
```

### 6. Route skill-specific learnings

Some findings belong in skill files, not `brain/`. Check all reports for skill-specific learnings and update the skill's SKILL.md or references/ directly. Read the skill first to avoid duplication.

### 7. Apply changes

Apply all changes directly based on the reports. The user will review the diff and give feedback.

- **Outdated notes**: Update or delete
- **Redundant notes**: Merge into the stronger note, delete the weaker
- **Low-value notes**: Delete
- **Verbose notes**: Condense in place
- **New connections**: Add `[[wikilinks]]` to existing notes
- **Tensions**: Reword to clarify the boundary
- **New principles**: Only from the distiller's report, only if genuinely independent. Write brain files following existing style and update `brain/principles.md` index
- **CLAUDE.md issues**: Rewrite or delete the flagged section
- **Stale memories**: Delete entirely stale entries and linked files. Rewrite partially stale entries.

## Agent Guidelines

When spawning agents, include in their prompts:
- The brain snapshot path: `/tmp/brain-snapshot.md` — agents Read this one file for all brain content
- The skills snapshot path: `/tmp/skills-snapshot.md` — for the skill reviewer only
- The auto-memory directory path — for the auditor only
- The CLAUDE.md path — for the auditor only
- The brain vault path: `brain/`
- Instruction to use Glob/Grep tools only for **codebase verification** (not for reading brain/skill files)
- Instruction to return a structured markdown report
- No edits — read-only agents
