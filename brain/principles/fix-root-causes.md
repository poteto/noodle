# Fix Root Causes

**Principle:** When debugging, never paper over symptoms. Trace every problem to its root cause and fix it there.

## Why

Symptom fixes accumulate: each workaround makes the system harder to reason about, and the real bug remains. Root-cause fixes are slower upfront but reduce total debugging time across the project's lifetime.

## Pattern

- **Reproduce first.** If you can't reproduce it, you can't verify your fix.
- **Ask "why" until you hit bedrock.** The first explanation is usually a symptom. The test fails → the mock is wrong → the interface changed → the type doesn't match the runtime shape. Fix the type, not the mock.
- **Resist the urge to add guards.** Adding a nil check to silence a crash is a symptom fix. Why is it nil? Fix that.
- **Check for the pattern, not just the instance.** If one file has a bug, grep for the same pattern. Fix all instances, or make it structurally impossible.
- **When stuck, instrument — don't guess.** Add logging, read the actual error. Hypothesizing without data leads to whack-a-mole fixes.

## Restart Bugs: Suspect State Before Code

Code doesn't change between runs. State does. When the symptom is "fails after restart," the cause is almost always stale persistent state.

- **Inventory persistent artifacts first.** Before reading code, list everything that survives a restart: config files, caches, lock files, serialized state. These are your suspects.
- **"Reset unwedged it" is a strong signal.** If clearing a state file restores behavior, prioritize state validation as the fix path.
- **Check for staleness.** Does the loaded state assume a context (queue composition, active sessions) that no longer holds?
- **Test by clearing.** Clear the suspect state file and restart. If the system works, you found it.
- **Don't trace code when tests pass.** If the code is correct in tests but broken in production, the difference is the data — not the code.

## Anti-Patterns

- **Bypass flags.** `--no-verify`, `--force`, `SKIP_CHECK=true` — these silence symptoms. Use only as temporary diagnostics.
- **Retry loops.** "If it fails, try again" hides intermittent bugs behind probability.
- **Shotgun debugging.** Changing three things at once makes it impossible to know what fixed the problem.

## Relationship to Other Principles

[[principles/prove-it-works]] says to check real state, not proxies. This extends that to debugging: check the real cause, not the proxied symptom.

[[principles/encode-lessons-in-structure]] provides the encoding mechanism once a root cause is understood — make it structurally impossible to recur.

See also [[principles/prove-it-works]], [[principles/encode-lessons-in-structure]]
