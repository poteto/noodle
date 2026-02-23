# Cost-Aware Delegation

**Principle:** Every delegation boundary has a budget. Account for delegation overhead itself, and hard-cap scope to prevent work from expanding to fill available resources.

## Why

Agent turns, CI minutes, and API dollars are finite. Without explicit budgets, work expands to fill the available resources — CI runs time out, agents spend 5 turns rediscovering context you already had, and coordination overhead eats the margin.

## Pattern

- **Budget before delegating.** Count turns per phase: setup (2-3), read context (1-2), implement (2-5), verify + fix (2-4), git ops (2-3). If total > budget - 7, the scope is too large.
- **Front-load context to avoid rediscovery costs.** Every piece of analysis withheld is a turn wasted. The cost of a longer prompt is one read; the cost of rediscovery is 3-5 turns.
- **Hard-cap scope.** Max 2-3 files per CI phase. 1 function/type + tests per phase. Without caps, work expands.
- **Account for coordination overhead.** TeamCreate costs 8-10 turns in coordination. Task subagents return directly — saves 5-7 turns. Choose the cheaper mechanism unless coordination is genuinely needed.
- **Exit smart, not late.** Commit passing work at budget - 7, not budget - 5. Git operations reliably cost 2-3 turns.

## Relationship to Other Principles

This is not about *what* to build ([[principles/foundational-thinking]]) or *how* to structure it ([[principles/boundary-discipline]]). It's about the economics of having someone else build it — treating attention as a finite currency.

See also [[delegation/include-domain-quirks]]
