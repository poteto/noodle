# Fix Root Causes

**Principle:** When debugging, never paper over symptoms. Trace every problem to its root cause and fix it there.

## Why

Symptom fixes accumulate: each workaround makes the system harder to reason about, and the real bug remains. Root-cause fixes are slower upfront but reduce total debugging time across the project's lifetime.

## Pattern

- **Reproduce first.** If you can't reproduce it, you can't verify your fix.
- **Ask "why" until you hit bedrock.** The test fails → the mock is wrong → the interface changed → the type doesn't match the runtime shape. Fix the type, not the mock.
- **Resist the urge to add guards.** Adding a nil check to silence a crash is a symptom fix. Why is it nil? Fix that.
- **Check for the pattern, not just the instance.** If one file has a bug, grep for the same pattern. Fix all instances, or make it structurally impossible.
- **When stuck, instrument — don't guess.** Add logging, read the actual error.

## Restart Bugs: Suspect State Before Code

Code doesn't change between runs. State does. When "fails after restart," suspect stale persistent state first — config files, caches, lock files, serialized state. If clearing a state file restores behavior, prioritize state validation as the fix.

## Relationship to Other Principles

[[principles/prove-it-works]] says to check real state, not proxies. This extends that to debugging: check the real cause, not the proxied symptom.

[[principles/encode-lessons-in-structure]] provides the encoding mechanism once a root cause is understood — make it structurally impossible to recur.

See also [[principles/prove-it-works]], [[principles/encode-lessons-in-structure]]
