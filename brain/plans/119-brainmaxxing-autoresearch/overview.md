# Plan 119: Brainmaxxing Autoresearch Improvements

## Summary

Upgrade brainmaxxing from a learning-capture system to a learning-capture-and-optimization system. Two changes: harden meditate with a curator gate and skill refinement loop, and enrich reflect with structured findings.

**Target repo:** `~/code/brainmaxxing` (upstream skill pack). Changes install into any project that uses brainmaxxing.

## Motivation

Research into autoresearch (Karpathy, GEPA, Darwin Derby, autoresearch-skill, autocontext) revealed a clear gap: brainmaxxing captures *what happened* and consolidates *patterns*, but never measures whether its edits actually improved outcomes. Meditate can silently degrade the brain. Reflect loses signal from narrative stage_messages. No skill is ever tested against alternatives.

The autoresearch insight is simple: put an agent inside a fast feedback loop with a fixed evaluator and let it iteratively search. Brainmaxxing has the loop (reflect → meditate → ruminate) but lacks the evaluator and the search.

## Design Decisions

**Curator gate over unchecked application.** Meditate currently applies all changes directly (step 6). Bad brain edits compound silently because every session reads the brain. A curator step adds explicit accept/reject decisions for high-risk changes while auto-applying safe ones. This follows [[principles/encode-lessons-in-structure]] — structural enforcement of brain quality rather than trusting meditate's judgment wholesale. It also follows [[principles/never-block-on-the-human]] — the curator is thresholded, not universal, so low-risk changes don't stall the loop.

**Skill refinement inside meditate, not a separate skill.** Meditate already reviews skills against principles (current step 5). When it finds issues, it currently proposes text fixes by guessing. The refinement loop replaces guessing with measured experimentation: mutate the skill, test it, keep only improvements. This is a natural extension of meditate's existing skill review — the same step, done rigorously. No new skill to install, discover, or trigger. This follows [[principles/subtract-before-you-add]] — extend what exists instead of adding a new top-level concept.

**Structured findings over narrative messages.** Quality emits flat `{ "message": "...", "blocking": bool }`. Reflect parses this narrative and often loses signal. Structured findings (typed check results with file paths, evidence, principle refs) let reflect route precisely. This follows [[principles/boundary-discipline]] — validate and structure at the boundary (quality output), trust the types internally (reflect consumption). Findings travel via conversation context using a tagged envelope (`## Structured Findings` heading + fenced JSON block) — reflect only matches this tagged form (ignoring pasted examples or discussion) and consumes only the last such block in the conversation. Recurrence detection stays with reflect (which has the brain), not with upstream producers.

**Typed edit proposals over prose pipelines.** Meditate's agents return prose reports that step 6 re-interprets to decide edits. This is fragile — identical verdicts can produce different edits across runs. Instead, auditor/reviewer reports should contain typed edit proposals (file path, action, content, base_hash) that the curator operates on directly. Risk classification (auto-apply vs needs-curation) is computed centrally by meditate from action + confidence, not emitted by agents. This follows [[principles/foundational-thinking]] — get the data structure right before writing logic — and [[principles/boundary-discipline]] — policy lives in the shell, not in agent output.

### Alternatives considered

**A. Use GEPA directly via `optimize_anything()`.** Write a 20-line Python evaluator that shells out to `claude` CLI. Rejected: adds Python dependency, GEPA needs 100-500 evals per run (expensive), and brainmaxxing's value prop is zero-dependency skill pack.

**B. Separate optimize skill.** An earlier draft added a standalone `optimize` skill. Rejected: meditate already reviews skills — the refinement loop is a more rigorous version of the same step. A separate skill adds a new top-level concept to discover and trigger for no structural reason.

**C. Sidecar file protocol for structured findings.** An earlier draft proposed `<id>-findings.json` sidecar files. Rejected after adversarial review: brainmaxxing has no stage-output convention, non-Noodle users get zero benefit, and stale sidecars create false-positive routing.

**D. Full autocontext-style multi-agent teams.** Autocontext uses Competitor, Analyst, Coach, Curator, Skeptic agents. Rejected: too heavy for brainmaxxing's scope.

## Scope

**In scope:**
- Curator quality gate + skill refinement loop in meditate (SKILL.md + agents.md + references rewrite)
- Structured findings schema definition + reflect consumption

**Out of scope:**
- Noodle stage_message schema changes (separate PR, noted as dependency)
- Quality skill structured output (noodle-specific, not brainmaxxing)
- Pareto scheduling (premature — no telemetry data)
- gskill / git history mining (speculative)
- Config evolution
- Dead-end tracking (future work, lightweight addition to reflect)

## Scope limitation

Skill refinement is specifically *skill prompt optimization* — not brain-quality measurement or full learning-loop evaluation. Measuring whether brain edits improve downstream outcomes requires longitudinal tracking and is future work. Skill refinement is a stepping stone: it proves the hill-climbing loop works on a narrower, measurable target.

## Verification

- Curator gate: run meditate on a test brain vault with known issues. Verify auto-apply items get applied, needs-curation items produce typed edit proposals with confidence. Verify curator-rejected items are not applied.
- Structured findings: manually include structured findings in a conversation, run reflect, verify it produces more precise brain notes than narrative-only baseline.
- Skill refinement: run meditate with refinement enabled on a skill with known issues and eval criteria, verify score improves over iterations, changelog records mutations, original skill untouched until promotion.

## Phases

1. [[plans/119-brainmaxxing-autoresearch/phase-1-curator-gate]] — Curator quality gate + skill refinement loop in meditate
2. [[plans/119-brainmaxxing-autoresearch/phase-2-structured-findings]] — Structured findings schema and reflect consumption

## Applicable Principles

- [[principles/encode-lessons-in-structure]]
- [[principles/prove-it-works]]
- [[principles/boundary-discipline]]
- [[principles/never-block-on-the-human]]
- [[principles/subtract-before-you-add]]
- [[principles/guard-the-context-window]]
- [[principles/serialize-shared-state-mutations]]
- [[principles/foundational-thinking]]
