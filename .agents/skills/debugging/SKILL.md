---
name: debugging
description: >
  Systematic debugging methodology grounded in root-cause analysis. Invoke this skill whenever
  you encounter a technical problem: build failures, test failures, runtime errors, unexpected
  behavior, type errors, performance issues, flaky tests, integration failures, configuration
  problems, or any situation where something isn't working as expected. Also use when the user
  reports a bug, asks "why is this broken", or when you need to investigate an error you
  produced. Triggers: any error, failure, crash, unexpected output, "debug", "fix this",
  "why doesn't this work", "investigate", "what's wrong", or when stuck on a problem.
---

# Debugging

Read [[principles/fix-root-causes]] before starting. Every debugging session follows that principle: trace to root cause, never paper over symptoms.

## Process

1. **Reproduce.** Get the exact error. Run the failing command, read the full output. If you can't reproduce, you can't verify your fix.

2. **Read the error.** The error message, stack trace, and line numbers are data. Read all of it before forming hypotheses. Most bugs tell you exactly where they are.

3. **Isolate.** Narrow the scope. Which file? Which function? Which line? Use binary search: comment out half, see if it still fails, repeat.

4. **Find root cause.** The first "fix" that comes to mind is usually a symptom fix. Ask "why?" until you hit the actual cause:
   - Test fails → mock is wrong → interface changed → type doesn't match runtime shape → **fix the type**
   - Build fails → import error → circular dependency → **restructure the modules**
   - Runtime crash → undefined value → missing null check → data source returns null on empty → **handle empty case at the source**

5. **Fix and verify.** Fix the root cause, not the symptom. Then verify: run the test, run the build, exercise the feature path. "It compiles" is not verification.

6. **Check for the pattern.** If the bug existed in one place, grep for the same pattern elsewhere. Fix all instances, or make it structurally impossible.

## When Stuck

- **Suspect state before code on restart bugs.** If the system works in tests but fails after restart, the bug is almost certainly in persistent state (config, caches, serialized overrides, lock files), not in the code. Inventory what persists, check for staleness, test by clearing. See [[principles/suspect-state-before-code]].
- **Instrument, don't guess.** Add logging at key points to see actual values. Read actual state ([[principles/observe-directly]]).
- **Diff what changed.** `git diff`, `git log`, `git bisect`. Most bugs are recent.
- **Simplify.** Strip the failing case to minimum reproduction. Remove everything unrelated.
- **Change one thing at a time.** Multiple changes at once make it impossible to know what fixed (or broke) it.
- **Check assumptions.** The bug is often in the part you're sure is correct. Verify everything.

## Anti-Patterns

- **Bypass flags** (`--no-verify`, `--force`, `SKIP_CHECK=true`) silence symptoms without fixing causes.
- **Retry loops** hide intermittent bugs behind probability.
- **Shotgun debugging** — changing three things to "see if it helps" — obscures causality.
- **Adding guards** (`if (x !== undefined)`) without asking why x is undefined.

## After Fixing

Per [[principles/recursive-self-healing]]: if this bug could recur or affected your understanding of the codebase, capture the lesson. Write a brain note, update a skill, or add a todo. The system should get smarter from every bug.
