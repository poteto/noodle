# Foundational Thinking

**Structural decisions** (data models, phase ordering, infrastructure) optimize for option value. **Code-level decisions** (helpers, abstractions, patterns) optimize for simplicity.

"Over-engineering" means making premature decisions that **close doors** — unnecessary abstractions, speculative features, indirection layers. Choosing the right foundational data structure **opens doors** — it preserves option value.

- Bad complexity: helper classes for one-time operations, feature flags for hypothetical needs, abstraction layers "in case we swap X"
- Good complexity: choosing the data structure that matches actual access patterns, even if a simpler one works for now

## Data Structures First

Get the data structures right before writing logic. The right structure makes downstream code obvious; the wrong one fights you at every turn.

- Define core types early and let them drive the architecture
- Trace every access pattern through a proposed structure: reads, writes, iterations, lookups, serialization
- Choose structures that match the dominant access pattern
- A data structure change late in the project is a rewrite; early is a one-line diff

At the code level, simplicity preserves options:
- **DRY at the structural level** (types, data models) — but three similar lines of code is better than a premature abstraction
- **Explicit over clever** — cleverness obscures intent = closing doors
- **No placeholder source files** — create files only when there's real code
- **Well-tested** — test behavior and edge cases, not line coverage. See [[principles/prove-it-works]].

**Concurrency corollary:** Before sharing state between actors, ask: "What happens if another actor modifies this concurrently?" If the answer isn't "nothing", isolate.

## Scaffold First

If something benefits all future work, do it first. Ask: "does every subsequent phase benefit from this existing?" If yes, it's scaffold — Phase 1, not the end.

- CI, linting, testing infra → scaffold
- Shared types, project config → scaffold

Applies to commits too — sequence for maximum option value:
- Infra/setup before features, tests before fixes
- Keep commits small and single-purpose
- Prefer commits that are easy to review, revert, and cherry-pick independently

Subtraction ([[principles/subtract-before-you-add]]) comes before scaffolding — remove dead weight first, then lay foundations.

Ask: "does this decision reduce my future options, or preserve them?"

See also [[principles/redesign-from-first-principles]], [[principles/prove-it-works]]
