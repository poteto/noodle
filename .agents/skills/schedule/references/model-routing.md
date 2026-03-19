# Model Routing

| Task type | Provider | Model |
|-----------|----------|-------|
| Tiny/small tasks (no deep thinking needed) | codex | gpt-5.3-codex-spark |
| Tiny/small tasks (no deep thinking needed) | claude | claude-sonnet-4-6 |
| Implementation, execution, coding | codex | gpt-5.4 |
| Judgment, strategy, planning, review | claude | claude-opus-4-6 |

Use spark or sonnet for small, mechanical tasks (simple renames, one-liner fixes, straightforward additions). Use full codex for anything requiring multi-step reasoning or cross-file coordination. When uncertain, codex for implementation, opus for judgment.
