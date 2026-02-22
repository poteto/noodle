# Foundational Thinking

**Structural decisions** (data models, phase ordering, infrastructure) optimize for option value. **Code-level decisions** (helpers, abstractions, patterns) optimize for simplicity.

"Over-engineering" means making premature decisions that **close doors** — unnecessary abstractions, speculative features, indirection layers that lock you into one approach. Choosing the right foundational data structure (e.g. Map over Array when every operation is ID-based) **opens doors** — it doesn't reduce option value, it preserves it.

- Bad complexity: helper classes for one-time operations, feature flags for hypothetical needs, abstraction layers "in case we swap X"
- Good complexity: choosing the data structure that matches the actual access patterns, even if a simpler one works for now — because migrating data structures later costs more than the marginal upfront complexity

## Data Structures First

Get the data structures right before writing logic. Every downstream layer — rendering, mutations, undo, serialization, AI prompts — is shaped by data flow. The right structure makes downstream code obvious; the wrong one fights you at every turn.

- Define core types early and let them drive the architecture
- Trace every access pattern through a proposed structure: reads, writes, iterations, lookups, serialization
- Choose structures that match the dominant access pattern (for example, ID-based operations usually want keyed lookups)
- A data structure change late in the project is a rewrite; early is a one-line diff

At the code level, simplicity preserves options:
- **DRY at the structural level** (types, data models) — repetition there creates multiple change points that close doors. But three similar lines of code is better than a premature abstraction.
- **Explicit over clever** — cleverness obscures intent, making future changes harder = closing doors
- **No placeholder source files** — don't create empty files "for later". Create source files only when there's real code to put in them.
- **Well-tested** — test behavior and edge cases, not line coverage. See [[principles/verify-runtime]].

Ask: "does this decision reduce my future options, or preserve them?"

**Concurrency corollary:** Before sharing state between actors, ask: "What happens if another actor modifies this concurrently?" If the answer isn't "nothing", isolate.

See also [[principles/scaffold-first]], [[principles/redesign-from-first-principles]], [[principles/verify-runtime]]
