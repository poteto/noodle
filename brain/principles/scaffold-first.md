# Scaffold First

If something benefits all future work, do it first. Ask: "does every subsequent phase benefit from this existing?" If yes, it's scaffold — put it in Phase 1, don't append it at the end.

- CI, linting, testing infra → scaffold
- Shared types, project config → scaffold
- Only things that depend on prior phases being complete should come later

Applies to commits too — order to maximize ability to ship quickly. Infra/setup before features, tests before fixes:
- One commit for a failing test, then another to fix the bug
- `.gitignore` before code changes
- Brain vault before skills that reference it
- Keep commits small and single-purpose: one logical change per commit
- Split unrelated changes (for example, plan updates vs principle updates) into separate commits
- Prefer commits that are easy to review, revert, and cherry-pick independently

Subtraction ([[principles/subtract-before-you-add]]) comes before scaffolding — remove dead weight first, then lay foundations.

A specific application of [[principles/foundational-thinking]] — sequencing for maximum option value.
