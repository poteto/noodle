---
name: debate
description: Structured multi-round debate for design decisions. Configures parameters per-task, runs reviewer/responder rounds, writes verdict.
noodle:
  blocking: false
  schedule: "When design decisions need structured evaluation before implementation"
---

# Debate

Structured multi-round review for design decisions. Single agent alternates between reviewer and responder roles.

## State

Per-task state in `.noodle/debates/<task-id>/`. See `references/debate-state-schema.md` for file layout.

## Process

### 1. Configure

Read `.noodle/debates/defaults.json` if it exists (project-wide defaults).

Evaluate the task context and write `.noodle/debates/<task-id>/config.json`:

| Field | Default | Guidance |
|---|---|---|
| `max_rounds` | 3 | Straightforward: 1-2. Standard: 3. Major refactor: 4-5 |
| `convergence` | `"no-high-severity-issues"` | Or `"unanimous"` for critical decisions |
| `focus_areas` | `[]` | e.g. `["performance", "API design", "error handling"]` |

Per-task config merges over project defaults (per-task wins on conflict).

### 2. Run rounds

Alternate reviewer and responder rounds, writing each to `round-NN-role.md`.

**Reviewer role:**
- Critique must be specific and testable — not "consider performance" but "this O(n^2) loop over tasks will block the queue at >1000 items"
- Assign severity: high (blocks ship), medium (should fix), low (nice to have)
- Scope critiques to `focus_areas` when set

**Responder role:**
- Address every critique individually — no batch dismissals
- For each: accept and describe the fix, or reject with specific rationale
- Unaddressed critiques carry forward to next round

### 3. Check convergence

After each responder round, evaluate against `convergence` criteria:
- `"no-high-severity-issues"` — no unresolved high-severity critiques remain
- `"unanimous"` — no unresolved critiques of any severity

If converged, proceed to verdict. If `max_rounds` reached without convergence, proceed to verdict with `consensus: false`.

### 4. Write verdict

Write `.noodle/debates/<task-id>/verdict.json`:

```json
{
  "consensus": true,
  "rounds": 3,
  "summary": "...",
  "open_questions": []
}
```

Non-convergence: set `consensus: false` and populate `open_questions` with unresolved critiques.

## Principles

- [[foundational-thinking]]
- [[subtract-before-you-add]]
- [[prove-it-works]]
