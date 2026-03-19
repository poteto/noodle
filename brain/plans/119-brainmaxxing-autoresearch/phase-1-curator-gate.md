# Phase 1: Curator Quality Gate + Skill Refinement in Meditate

## Goal

Two additions to meditate: (1) confidence-based triage so high-risk brain changes go through an explicit curator step, and (2) a skill refinement loop that replaces guesswork with measured experimentation when improving skills.

## Why

**Curator gate.** Meditate currently applies all changes directly in step 6. A bad deletion or incorrect merge silently degrades every future session. The curator gate follows [[principles/encode-lessons-in-structure]] — structurally enforce brain quality.

**Skill refinement.** Meditate already reviews skills against principles (current step 5). When it finds issues, it guesses at text fixes. The refinement loop replaces guessing with hill-climbing: mutate the skill, test it, keep only improvements. This follows [[principles/prove-it-works]] — skill improvements are measured, not assumed.

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

### `meditate/references/eval-guide.md` (new)

Guidance on writing binary evals for skill refinement:

- Evals must be binary (pass/fail, never scales)
- Different graders must agree on the answer (no subjectivity)
- Evals should not be gameable without genuine improvement
- Keep to 3-6 evals (more causes teaching-to-the-test)
- No overlapping evals (each tests a distinct quality dimension)
- Each eval format: name, yes/no question, explicit pass condition, explicit fail condition

### `meditate/SKILL.md`

Restructure the process into two tracks: **brain maintenance** (audit → curate → apply) and **skill refinement** (review → refine → promote).

#### Track A: Brain Maintenance (existing flow, hardened)

- Existing steps 1-4 (snapshots, auditor, reviewer, review reports): reports now contain typed edit proposals
- New step 4b (Triage): compute `risk_level` centrally from `action` + `confidence`:
  - **auto-apply**: `add-wikilink` (any confidence), `condense` (high confidence only)
  - **needs-curation**: `delete` (always), `merge` (always), `update` to principles (always), `create` (always), skill file edits (always), any action with low/medium confidence
- New step 4c (Conflict detection): check for proposals with overlapping affected files (`file_path ∪ merge_sources`). If found, group them and send as a unit to curator for resolution.
- New step 4d (Curator): spawn Curator subagent for all needs-curation items. Always run curator when needs-curation items exist — no count-based skip.
- Step 5 (Apply): apply auto-apply proposals directly. For curator-accepted proposals: auto-apply if confidence is high, stage as preview diff for user if confidence is medium/low. Skip curator-rejected proposals. Verify `base_hash` (and `source_hashes` for merges) match current files before applying.

Early-exit: if no needs-curation proposals exist after triage, skip curator entirely.

#### Track B: Skill Refinement (new — extends existing skill review)

**Trigger conditions.** The reviewer's skill review (Section 3 of its report) may identify skills with recurring principle violations, contradictions, or missed structural enforcement. Skill refinement runs when ALL of:
- The reviewer flagged a skill for improvement (not just a minor wording nit)
- Eval criteria exist for the skill (either pre-defined in `<skill>/evals.json` or gathered from the user during meditate)
- The user hasn't opted out of refinement for this meditate run

If any condition is unmet, fall back to existing behavior: propose a text fix via the curator gate (Track A).

**The refinement loop:**

1. **Isolate**: Copy the entire target skill directory to a working location (worktree if available, temp directory otherwise). Record a `manifest_hash` of all files. Only SKILL.md is mutable; references/ and scripts/ are read-only context. Test subagents must be pointed at the copy via explicit path, not skill name resolution.
2. **Baseline**: Run the copied skill against test prompts (from `evals.json` or user-provided) 5 times. Spawn one grading subagent per experiment batch. The grader receives eval criteria + all test outputs. The grader does NOT receive mutation descriptions or rationale. Record baseline pass rate.
3. **Analyze**: Read execution traces from failed runs. Identify the single most common failure pattern (ASI — full traces, not just scores).
4. **Hypothesize + Mutate**: Propose a single targeted change to the copy's SKILL.md. Prefer removing or clarifying over adding ([[principles/subtract-before-you-add]]). Apply the change.
5. **Test + Score**: Re-run all test prompts 5 times. Grade via separate subagent (same rules: eval criteria + outputs, no mutation info). Accept only if pass rate improved by >= 5% relative.
6. **Log**: Record in `<skill>-refinement-changelog.md`: experiment number, mutation, rationale, score delta, verdict, key failure trace.
7. **Repeat**: Back to step 3. Stop when: budget cap reached (default: 5 experiments), 95%+ pass rate for 2 consecutive experiments, or no improvement after 2 experiments.
8. **Promote**: Show user the changelog, score trajectory, and diff. Verify `manifest_hash` matches the original skill directory. If any file changed, warn and abort or re-test. User approves promotion or reverts.

**Key constraints:**
- One mutation per experiment. Isolates causation.
- Monotonic ratchet with proportional threshold. Noise rejected.
- Evals defined upfront and not modified during the loop.
- Grading subagent separate from mutation proposer. Grader sees eval criteria + output. Never sees mutation rationale.
- Original skill untouched until explicit promotion with manifest-hash check.
- One grader per experiment batch (not per run). Keeps cost bounded.

## Test Cases

### Curator gate

1. Brain with a clearly outdated note → auditor emits `delete` proposal → triage computes needs-curation → curator accepts → applied
2. Brain with two nearly-identical notes → auditor emits `merge` proposal → needs-curation → curator verifies unique content preserved → accepts
3. Auditor and reviewer both target the same file → conflict detection groups them → curator resolves
4. Brain with only minor issues (3 missing wikilinks) → all auto-apply → curator skipped
5. Skill file edit proposed by reviewer → needs-curation → curator reviews before application
6. Single needs-curation item → still goes through curator

### Skill refinement

7. Skill with recurring quality rejections + evals.json present → refinement triggers, baseline recorded, mutations attempted, changelog written, original untouched until promotion
8. Skill flagged for improvement but no eval criteria → falls back to text-fix proposal via curator
9. Skill already good (high baseline) → refinement terminates early after 2 experiments with no improvement
10. Concurrent edit to original skill during refinement → promotion detects manifest_hash mismatch, warns user
