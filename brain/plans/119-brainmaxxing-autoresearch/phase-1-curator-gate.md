# Phase 1: Curator Quality Gate for Meditate

## Goal

Add confidence-based triage to meditate so high-risk brain changes (deletions, merges, principle edits) go through an explicit accept/reject curator step, while safe changes (wikilink additions, typo fixes, condensing) auto-apply. Use typed edit proposals with conflict detection.

## Why

Meditate currently applies all changes directly in step 6. A bad deletion or incorrect merge silently degrades every future session because every session reads the brain. The curator gate follows [[principles/encode-lessons-in-structure]] — structurally enforce brain quality rather than trusting the LLM's judgment on every edit.

## Changes

### `meditate/references/agents.md`

**Typed edit proposals.** Both Auditor and Reviewer reports must contain structured edit proposals, not prose findings. Each proposal includes:

- `file_path`: which brain/skill file to modify
- `action`: one of `delete`, `merge`, `update`, `condense`, `create`, `add-wikilink`
- `content`: the proposed new content (for merge/update/create) or null (for delete)
- `merge_sources`: array of file paths being combined (for merge actions)
- `base_hash`: hash of the target file at snapshot time — enables staleness detection at apply time
- `source_hashes`: for merge actions, hash of each `merge_sources` file at snapshot time — enables conflict detection across the full touched-file set
- `confidence`: high / medium / low — how certain the agent is
- `rationale`: one-line justification

Agents do NOT emit `risk_level`. That is policy computed centrally by meditate's triage step.

Add a new **Curator** agent spec. The Curator receives:
- The needs-curation proposals
- The brain snapshot (same artifact the auditor/reviewer already use)
- The actual file contents for any files targeted by merge/update proposals

For each proposal, the Curator:
- Checks for contradictions with existing principles (read brain/principles.md)
- Checks semantic similarity with other brain notes (flag near-duplicates vs genuinely distinct)
- For merge proposals: verifies unique content from both sources is preserved
- Emits a verdict: accept (with rationale) or reject (with rationale)
- Does NOT apply changes — returns verdicts only

### `meditate/SKILL.md`

Restructure the process:

- Existing steps 1-4 (snapshots, auditor, reviewer, review reports): reports now contain typed edit proposals
- New step 4b (Triage): compute `risk_level` centrally from `action` + `confidence`:
  - **auto-apply**: `add-wikilink` (any confidence), `condense` (high confidence only)
  - **needs-curation**: `delete` (always), `merge` (always), `update` to principles (always), `create` (always), skill file edits (always), any action with low/medium confidence
- New step 4c (Conflict detection): check for proposals with overlapping affected files (`file_path ∪ merge_sources`). If found, group them and send as a unit to curator for resolution.
- New step 4d (Curator): spawn Curator subagent for all needs-curation items. Always run curator when needs-curation items exist — no count-based skip.
- **Move step 5 (skill routing) after curator triage.** Skill file edits are needs-curation and must go through the curator gate.
- Step 6 (Apply): apply auto-apply proposals directly. For curator-accepted proposals: auto-apply if confidence is high, stage as preview diff for user if confidence is medium/low. Skip curator-rejected proposals. Before applying each proposal, verify `base_hash` (and `source_hashes` for merges) match current files — if not, the file changed since snapshot and the proposal is stale (re-snapshot or skip).

Early-exit: if no needs-curation proposals exist after triage, skip curator entirely.

## Test Cases

1. Brain with a clearly outdated note (references deleted file) → auditor emits `delete` proposal with high confidence → triage computes needs-curation (deletions always curated) → curator accepts → applied
2. Brain with two nearly-identical notes → auditor emits `merge` proposal → needs-curation → curator verifies unique content preserved → accepts → applied
3. Auditor and reviewer both target the same file (condense + add-wikilink) → conflict detection groups them → curator resolves composition → applied as single edit
4. Brain with only minor issues (3 missing wikilinks) → all compute as auto-apply → curator skipped entirely
5. Skill file edit proposed by reviewer → triage computes needs-curation → curator reviews before application
6. Single needs-curation item (one deletion) → still goes through curator (no count-based skip)
