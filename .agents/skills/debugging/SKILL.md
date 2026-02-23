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

3. **Suspect state before code.** Before debugging code, check persistent state — especially on restart bugs or environment drift. See [[principles/suspect-state-before-code]].
   - **Noodle state**: `.noodle/queue.json` (stale items?), `.noodle/sessions/` (orphaned?), `noodle.toml` (valid?)
   - **Environment**: tmux sessions (`tmux ls`), lock files, cached artifacts
   - **Config**: `noodle.toml` validation (`noodle start` reports diagnostics), skill frontmatter parse errors
   - **Persistent files**: brain/ notes, plan files, todos — check for corruption or stale references

4. **Isolate.** Narrow the scope. Which file? Which function? Which line? Use binary search: comment out half, see if it still fails, repeat.

5. **Find root cause.** The first "fix" that comes to mind is usually a symptom fix. Ask "why?" until you hit the actual cause:
   - Test fails → mock is wrong → interface changed → type doesn't match runtime shape → **fix the type**
   - Build fails → import error → circular dependency → **restructure the modules**
   - Runtime crash → undefined value → missing null check → data source returns null on empty → **handle empty case at the source**

6. **Fix and verify.** Fix the root cause, not the symptom. Then verify: run the test, run the build, exercise the feature path. "It compiles" is not verification.

7. **Check for the pattern.** If the bug existed in one place, grep for the same pattern elsewhere. Fix all instances, or make it structurally impossible.

## Noodle-Specific Diagnostics

When debugging Noodle infrastructure failures:

| Symptom | Check first |
|---------|-------------|
| Cook won't start | `tmux ls`, `.noodle/queue.json` validity, skill resolver paths |
| Queue stuck | `.noodle/queue.json` for stale items, `noodle.toml` adapter config |
| Missing skill | `noodle.toml` `[skills]` paths, `.agents/skills/<name>/SKILL.md` exists |
| Stale mise | `.noodle/mise.json` `generated_at` timestamp, backlog sync script |
| Session orphaned | `.noodle/sessions/<id>/meta.json` status, `tmux ls` for zombie sessions |

## When Stuck

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

If this bug could recur or affected your understanding, capture the lesson. Write a brain note, update a skill, or add a todo. The system should get smarter from every bug. See [[principles/encode-lessons-in-structure]].
