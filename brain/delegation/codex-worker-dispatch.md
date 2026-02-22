# Codex Worker Dispatch

Operational procedures for [[principles/cost-aware-delegation]] applied to Codex worker dispatch. The `codex-worker` agent is a thin dispatcher, not a planner. Its job is to fire `codex exec` fast with a concise prompt (10-20 lines).

## Key Lessons

- **Model: Sonnet, not Opus.** Opus overthinks the prompt — spends 5+ turns sketching pseudocode, reading files, composing a detailed plan. Codex is a capable coding agent; it doesn't need that level of hand-holding.
- **Codex model routing:** Keep `gpt-5.3-codex` as default. Use `gpt-5.3-codex-spark` only for speed-first, interactive loops (quick targeted edits, rapid back-and-forth). Prefer default model for long autonomous, broad, or test-heavy work.
- **Speed target:** First `codex exec` call within 1-2 turns.
- **Prompt content:** Task description + file paths + constraints. No pseudocode, no inline code snippets, no architecture analysis.
- **Failure mode:** If the worker hasn't called `codex exec` by turn 3, the prompt is too complex or the worker is overthinking.

See also [[delegation]], [[delegation/share-what-you-know]], [[principles/cost-aware-delegation]], [[principles/guard-the-context-window]]
