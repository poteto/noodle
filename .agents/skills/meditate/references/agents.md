# Agent Prompt Specs

All agents are spawned as `subagent_type: "general-purpose"`. All agents are **read-only** — they return structured markdown reports, no edits.

## Common prompt inclusions

Every agent prompt should include:
- Brain snapshot path: `/tmp/brain-snapshot.md` — Read this one file for all brain content
- Brain vault path: `brain/`
- Use Glob/Grep only for **codebase verification**, not for reading brain/skill files
- Return a structured markdown report

## Auditor

**Inputs:** brain snapshot, auto-memory directory path, CLAUDE.md path.

Prompt spec:

- Read `/tmp/brain-snapshot.md` and parse it to build a wikilink map — no individual brain file reads needed
- Use the file headers (`=== path ===`) as the on-disk file list for orphan detection
- Cross-reference each note against the current codebase state (check if referenced files, patterns, tools, or decisions still exist) — the only part that requires Read/Grep/Glob calls
- Flag notes as:
  - **Outdated**: References code, tools, patterns, or decisions that no longer exist or have changed
  - **Redundant**: Says the same thing as another note
  - **Low-value**: Fails the test: "Would Claude reliably get this wrong without this note, AND does it come up often or cause real damage?" If not both, flag it.
  - **Verbose**: Could convey the same information in fewer words
  - **Orphaned**: Exists on disk but is not linked from any index or other brain file
- **Audit CLAUDE.md**: Flag sections that are outdated, redundant with brain notes or skill instructions, or could be condensed. Check that instructions match actual project structure.
- **Audit auto-memory files**: Read MEMORY.md and linked files. Flag:
  - **Stale session state**: Entries referencing completed work, old session IDs, or finished tasks
  - **Duplicated in brain**: Entries that duplicate brain note content
- Produce a report with brain, CLAUDE.md, and memory findings separated. Each item: what's flagged, why, and suggested action (update, merge, condense, or delete).

## Synthesizer

**Inputs:** brain snapshot, auditor report.

Prompt spec:

- Read `/tmp/brain-snapshot.md` (skip notes the auditor flagged for deletion)
- **Connect**: Propose missing `[[wikilinks]]` between notes that reference the same concepts
- **Tensions**: Flag principles that appear to conflict; propose how to resolve or clarify the boundary
- **Clarify**: Propose rewording where a note's relationship to a principle is unclear
- Do NOT propose merging principles — they are intentionally independent
- Produce a report with proposed connections, tensions, and clarifications

## Distiller

**Inputs:** brain snapshot, auditor report.

Prompt spec:

- Read `/tmp/brain-snapshot.md` (skip notes the auditor flagged for deletion) — focus on codebase notes, preferences, plan retrospectives, and gotchas
- Look for **recurring patterns** across operational notes that reveal unstated engineering principles
- A valid new principle must be: (1) genuinely independent — not derivable from existing principles, (2) evidenced by 2+ separate notes, (3) actionable — changes how you'd approach future work
- Do NOT propose restatements of existing principles applied to a new domain
- Produce a report: each proposed principle with the insight, evidence (which notes), why it's independent, and suggested path under `brain/principles/`

## Skill Reviewer

**Inputs:** brain snapshot, skills snapshot (`/tmp/skills-snapshot.md`), auditor/synthesizer/distiller reports, `brain/principles.md`.

Prompt spec:

- Read both snapshots
- For each skill, check against brain principles:
  - Does it contradict any principle?
  - Does it miss a structural enforcement opportunity? (per encode-lessons-in-structure — can an instruction become a lint rule, script, metadata flag, or runtime check?)
  - Does it duplicate instructions that a mechanism already handles?
  - Is it missing a principle that would improve reliability?
- Cross-reference with the other reports — could a recurring pattern be encoded into a skill?
- Audit each skill's `description` frontmatter for context bloat — cut what Claude can infer (exhaustive error types, synonym trigger lists, restated purpose). Keep only distinctive triggers and core purpose.
- Produce a report: each finding with skill name, principle(s), gap, and concrete proposal
- Prioritize structural enforcement over textual instructions

## Report Template

Use this format when presenting the consolidated summary in Step 5:

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
