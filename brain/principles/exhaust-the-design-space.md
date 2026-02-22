# Exhaust the Design Space Before Committing

When facing a novel interaction or architectural decision with no established precedent, explore multiple concrete alternatives before committing to implementation. The cost of building the wrong thing dwarfs the cost of exploring three options.

## The Rule

For decisions where the right answer isn't obvious, build 2-3 competing prototypes or sketches. Compare them side-by-side. Only then commit.

## When It Applies

- Novel UI interactions (no prior art in the codebase)
- Architectural choices with multiple viable approaches
- Product design decisions where user experience depends on feel, not logic

## When It Doesn't

- Mechanical implementation where the pattern is established
- Bug fixes or refactors with a clear target state
- Changes where constraints dictate a single viable approach

## Evidence

- Plan 05 (canvas-native AI) mandates two prototyping phases before production code: three HTML prototypes for interaction models, four mascot layouts with a tuning panel
- The product design audit calls out "needs prototyping and ideation before implementation" for the most important feature
- [[principles/experience-first]] embeds this: "Prototype before committing. Design decisions are cheaper to explore in throwaway HTML than in production React."

## Distinction from Existing Principles

[[principles/redesign-from-first-principles]] governs how to *integrate* a change. This governs how to *discover which change to make* when the answer isn't obvious. [[principles/experience-first]] mentions prototyping as one tactic; this elevates design-space exploration to a decision-making principle.

See also [[principles/experience-first]], [[principles/redesign-from-first-principles]]
