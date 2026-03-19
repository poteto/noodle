# Phase 3: Optimize Skill — GEPA-Inspired Skill Hill-Climbing

## Goal

Add a new `optimize` skill to brainmaxxing that autonomously improves any skill's SKILL.md through iterative mutation, testing, and selection — an autoresearch loop for skill prompt optimization.

**Scope note:** This is specifically *skill prompt optimization*, not brain-quality measurement or full learning-loop evaluation. Measuring whether brain edits improve downstream outcomes requires longitudinal tracking and is future work.

## Why

Skills are brainmaxxing's highest-leverage artifact — they run on every task of their type, so a 10% improvement in a skill compounds across every future invocation. Currently, skill improvement is manual. The optimize skill automates this loop using the same hill-climbing pattern that GEPA and autoresearch-skill use, but implemented natively as a brainmaxxing skill (no Python dependency).

This follows [[principles/prove-it-works]] — skill improvements are measured, not assumed. And [[principles/encode-lessons-in-structure]] — the improvement is encoded in the skill file itself.

## Design decisions from adversarial review

**Worktree/copy isolation with base-version-checked promotion.** The optimizer never touches the live skill. It works on a copy of the entire skill directory (SKILL.md + references/ + scripts/). Only SKILL.md is mutable; the rest is read-only context. Promotion requires a base-version check: hash the original at start, compare at promote time. If the original changed (another optimize run or manual edit), re-test the candidate against the new base or abort. This follows [[principles/serialize-shared-state-mutations]].

**Grader receives eval criteria + output, but NOT mutation rationale.** Round 1 proposed fully blind grading (grader sees only output). Round 2 reviewers (all three) flagged this as unusable — the grader can't check "starts with a conventional-commit prefix" without knowing that criterion. Fix: the grader receives the eval criteria and the test output. What it must NOT receive is the mutation description, rationale, or any information about what changed in the skill. This separates evaluator from optimizer without making grading impossible.

**Proportional scoring threshold.** Round 1 used bare improvement. Round 2 used absolute +2 delta. Both were flagged as statistically ungrounded. Fix: require >= 5% relative improvement in pass rate (e.g., 80% → 84%, not 80% → 81%). This scales with experiment size.

**Standalone only — no skill-creator integration branches.** Ship the standalone loop first. Skill-creator integration (richer stats, eval viewer) is future work after the standalone path is proven. This follows [[principles/subtract-before-you-add]].

## Changes

### New file: `.agents/skills/optimize/SKILL.md`

The skill definition. Key sections:

**Setup interview**: Gather from user:
- Target skill path (the entire skill directory, not just SKILL.md)
- 3-5 test prompts (representative inputs covering different use cases)
- 3-6 binary eval criteria (yes/no questions with explicit pass/fail conditions)
- Runs per experiment (default: 5)
- Budget cap (max experiments, default: 10)

**Isolation**: Copy the entire target skill directory to a working location. If worktree tooling is available (Noodle), use a worktree. Otherwise, copy to a temp directory. Record a `manifest_hash` — the combined hash of all files in the skill directory — for promotion safety.

Only SKILL.md within the copy is mutable. References/ and scripts/ are read-only context — they're there so test runs exercise the real skill, not a partial artifact. Test subagents must be pointed at the copy via explicit path, not skill name resolution (Noodle resolves skills by name from search paths — relying on name resolution would silently test the original).

**The loop**:
1. **Baseline**: Run the copied skill against all test prompts N times. After all runs complete, spawn one grading subagent per experiment (not per run) that grades all N outputs in a batch. The grader receives: (a) the eval criteria and (b) all test outputs for this experiment. The grader does NOT receive: the mutation description, rationale, or any information about what changed. Record baseline pass rate.
2. **Analyze**: Read execution traces from failed runs. Identify the single most common failure pattern. This is ASI — the full trace, not just the score.
3. **Hypothesize**: Propose a single, targeted change to the copy's SKILL.md that addresses the top failure. Explain rationale. Prefer removing or clarifying over adding ([[principles/subtract-before-you-add]]).
4. **Mutate**: Apply the change to the working copy's SKILL.md.
5. **Test**: Re-run all test prompts N times with the mutated copy. Grade via one separate subagent per experiment batch (same rules: eval criteria + outputs, no mutation info).
6. **Score**: Compare against current best. Accept only if pass rate improved by >= 5% relative (e.g., 80% → 84%).
7. **Log**: Record in changelog.md: experiment number, mutation description, rationale, score delta, verdict, key failure trace excerpt.
8. **Repeat**: Go to step 2. Stop when: user interrupts, budget cap reached, or 95%+ pass rate sustained for 2 consecutive experiments.

**Promote**: On completion, show the user: changelog, score trajectory (baseline → final), and the diff between original and optimized SKILL.md. Before promoting:
- Verify `manifest_hash` matches the current hash of the original skill directory (all files, not just SKILL.md)
- If any file changed (concurrent edit, another optimize run, or manual reference update), warn the user and either re-test against the new base or abort
- User approves promotion (copy optimized SKILL.md to original location) or reverts (discard working copy)

**Key constraints**:
- One mutation per experiment. No multi-variable changes.
- Monotonic ratchet with proportional threshold. Noise and regressions are rejected.
- Evals defined upfront and not modified during the loop.
- Grading subagent is separate from mutation proposer. Grader sees eval criteria + output. Grader never sees mutation rationale.
- Original skill untouched until explicit promotion with manifest-hash check.
- Isolation unit is the full skill directory, not just SKILL.md.

### New file: `.agents/skills/optimize/references/eval-guide.md`

Guidance on writing effective binary evals:

- Evals must be binary (pass/fail, never scales)
- Different graders must agree on the answer (no subjectivity)
- Evals should not be gameable without genuine improvement
- Keep to 3-6 evals (more causes teaching-to-the-test)
- No overlapping evals (each tests a distinct quality dimension)
- Each eval format: name, yes/no question, explicit pass condition, explicit fail condition
- Bad evals: "Is the output good?" (subjective), "Does it follow best practices?" (vague)
- Good evals: "Does the commit message start with a conventional commits type prefix?" (binary, objective)

### New file: `.agents/skills/optimize/references/changelog-format.md`

Template for the experiment changelog:

- Experiment number, timestamp
- Mutation: what changed (one sentence) and why (one sentence)
- Score: before → after pass rate (N/M evals passed across R runs)
- Delta: +X% relative improvement (or "no improvement")
- Verdict: accepted / rejected
- Key failure trace: excerpt from the most informative failed run (ASI)
- Running best score and cumulative accepted mutations

## Test Cases

1. Optimize a simple skill (commit message formatter) with 3 test prompts and 4 binary evals in a worktree. Verify: baseline recorded, mutations attempted, changelog written, original skill directory untouched until promotion.
2. Run optimization where the skill is already good (high baseline). Verify the loop terminates early when no mutation achieves the 5% threshold after 2-3 experiments.
3. Run optimization with a budget cap of 3. Verify the loop stops at 3 regardless of score.
4. Verify grading subagent receives eval criteria + test output, but NOT the mutation rationale.
5. Interrupt mid-optimization. Verify original skill is untouched (worktree isolation).
6. Simulate concurrent edit: change the original SKILL.md during optimization. Verify promotion detects the base_hash mismatch and warns the user.
