# Subtract Before You Add

**Principle:** When evolving a system, remove complexity first, then build. Deletion creates a simpler substrate that makes subsequent additions cleaner, smaller, and less error-prone.

## Why

Adding to a complex system compounds complexity. Removing first reduces the surface area, reveals the essential structure, and makes the addition's design more obvious. The default action should be subtraction.

## The Pattern

- **Sequence removal before construction.** In plans, schedule deletion phases before build phases. Each piece of dead code, unused feature, or unnecessary abstraction removed makes the subsequent work simpler.
- **Cut before you polish.** When scoping a feature set, cut ruthlessly to the minimum before investing in quality. Half-finished features are worse than missing ones.
- **Design for observed usage, not speculative edge cases.** If the real workflow is single-process or low-concurrency, prefer simpler designs that fit that reality. Add edge-case machinery only after usage data says it's needed.
- **Simplify prompts.** Remove redundant instructions, excessive templates, and sub-band examples. One-line instructions often work as well as 15-line templates.
- **When a reference has no novel content, delete it** rather than leaving a stub.

## Evidence

- UI simplification plan: "removal before construction. Deleting code first (phases 1-4) reduces the surface area and makes later merge/consolidation work (phases 5-10) simpler."
- Product direction: "Say no to 1,000 things. Every item must earn its place. When in doubt, cut."
- CI prompt design: stripped sub-band examples, output format templates, and redundant instructions — each removal improved clarity without reducing quality.
- Skill authoring: "If a reference file has no novel content, delete it entirely rather than leaving a stub."
- Noodle planning: deferred locking and multi-process complexity when the expected usage is one active noodle process, focusing effort on command consistency and actual operator pain.

## Relationship to Other Principles

This is not about *what* to build ([[principles/foundational-thinking]]) or *how* to redesign it ([[principles/redesign-from-first-principles]]). It's about *when* to act — an ordering principle that says subtraction comes before addition.

See also [[todos]]
