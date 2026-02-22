---
name: debate
description: Structured multi-round debate protocol for validating a plan, diff, or artifact. Use when a reviewer/responder loop is needed to reach consensus or stop at max rounds.
---

# Debate Protocol

Run debates as alternating rounds between `reviewer` and `responder`.

## Steps

1. Create a debate target (plan, diff, or artifact) and debate directory.
2. Round 1 (odd rounds): reviewer writes a concrete critique.
3. Round 2 (even rounds): responder addresses the critique directly.
4. After each responder round, write `verdict.json`:
   - `consensus: true` when issues are resolved
   - `consensus: false` when more rounds are needed
   - `summary`: one short rationale
5. Repeat reviewer/responder alternation until:
   - `consensus: true`, or
   - max rounds are reached.

## File Contract

- `round-1.md`, `round-2.md`, ... contain round content only.
- `verdict.json` contains:

```json
{"consensus": false, "summary": "Needs one more revision to close two risks."}
```

## Style

- Reviewer: specific, testable critique; no vague feedback.
- Responder: map each response to a critique item.
- Keep rounds concise so the debate converges quickly.
