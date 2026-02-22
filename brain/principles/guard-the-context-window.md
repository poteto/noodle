# Guard the Context Window

**Principle:** The context window is finite and non-renewable within a session. Every token that enters should earn its place.

## Why

Context overflow degrades reasoning quality, causes compression artifacts, and halts progress. Unlike compute or time, context consumed within a session cannot be reclaimed.

## Pattern

- **Isolate large payloads.** Route DOM snapshots, screenshots, and verbose tool outputs to subagents. The main context gets summaries, not raw data.
- **Don't read what you won't use.** In CI workers, don't read brain files — plans already encode the principles. In interactive sessions, read selectively based on relevance, not exhaustively.
- **Keep frequently-used content inline.** Skill templates used on every invocation belong in SKILL.md, not in references/. Every file read costs tokens — only split to references when content is truly conditional.
- **Size phases and cap scope.** See [[principles/cost-aware-delegation]] for specific numbers (2-3 files, turn budgets, mechanism costs).

## Relationship to Other Principles

[[principles/cost-aware-delegation]] treats delegation as economics. This principle treats the context window itself as the scarce resource — it applies to both delegated and self-directed work.

See also [[principles/cost-aware-delegation]]
