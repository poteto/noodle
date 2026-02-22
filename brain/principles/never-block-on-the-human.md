# Never Block on the Human

**Principle:** The human supervises asynchronously. Agents must stay unblocked — make reasonable decisions, proceed, and let the human course-correct after the fact. Code is cheap; waiting is expensive.

## Why

Every time an agent pauses to ask for permission, the entire pipeline stalls. The human becomes the bottleneck in a system designed to multiply their output. Since code changes are reversible and reviewable, the cost of a wrong decision is almost always lower than the cost of blocking.

## Pattern

- **Proceed, then present.** Do the work, show the result. Don't ask "should I do X?" — do X, explain why, and let the human redirect if needed.
- **Reserve questions for genuine ambiguity.** Ask only when you truly cannot infer intent from context — not for low-stakes decisions like which skill to install or how to split tasks.
- **Make the system self-healing.** When you notice a problem — in code, in the brain, in a skill — log it as a todo and fix it in the next round. The system improves continuously without requiring the human to triage every issue.
- **Supervision is async.** The human reviews plans, diffs, and brain changes on their own schedule. Design workflows for review-after-the-fact, not approval-before-the-fact.
- **Code is cheap, attention is scarce.** A wrong implementation costs minutes to fix. A blocked agent costs the human's attention to unblock. Bias toward action.

## Boundaries

This does not mean agents should ignore the human. It means:
- **Irreversible actions** (force-push, delete production data, send external messages) still require confirmation.
- **Reversible actions** (write code, edit brain notes, install skills, split tasks) should proceed without blocking.
- **Product direction** comes from the human; *execution* should not block on the human. "Bias toward action" applies to implementation decisions (how to code something), not product decisions (what to build).
- **The human can always interrupt.** Workflows must be designed so the human can redirect mid-stream without losing work.

## Relationship to Other Principles

[[principles/cost-aware-delegation]] treats delegation as an economic problem. This principle extends that thinking to the human-agent boundary: the human's attention is the scarcest resource in the system, so minimize demands on it.

See also [[principles/encode-lessons-in-structure]], [[principles/cost-aware-delegation]]
