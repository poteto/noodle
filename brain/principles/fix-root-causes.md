# Fix Root Causes

**Principle:** When debugging, never paper over symptoms. Trace every problem to its root cause and fix it there. A symptom fix creates two problems: the original bug still exists, and now there's a workaround obscuring it.

## Why

Symptom fixes are seductive because they're fast. But they accumulate: each workaround makes the system harder to reason about, and the real bug remains, waiting to surface in a different form. Root-cause fixes are slower upfront but reduce total debugging time across the project's lifetime.

## Pattern

- **Reproduce first.** Before fixing, reproduce the failure reliably. If you can't reproduce it, you can't verify your fix.
- **Ask "why" until you hit bedrock.** The first explanation is usually a symptom. The test fails → the mock is wrong → the interface changed → the type doesn't match the runtime shape. Fix the type, not the mock.
- **Resist the urge to add guards.** Adding `if (x !== undefined)` to silence a crash is a symptom fix. Why is `x` undefined? Fix that.
- **Check for the pattern, not just the instance.** If one file has a bug, grep for the same pattern in the codebase. Fix all instances, or better yet, make the bug structurally impossible.
- **When stuck, instrument — don't guess.** Add logging, use the debugger, read the actual error. Hypothesizing without data leads to whack-a-mole fixes.

## Anti-Patterns

- **Bypass flags.** `--no-verify`, `--force`, `SKIP_CHECK=true` — these silence symptoms without fixing causes. Use them only as temporary diagnostics, never as permanent fixes.
- **Retry loops.** "If it fails, try again" hides intermittent bugs behind probability. Find out why it fails sometimes.
- **Shotgun debugging.** Changing three things at once to "see if it helps" makes it impossible to know what actually fixed the problem — or whether it's actually fixed.
- **Cargo-cult solutions.** Copying a fix from Stack Overflow without understanding why it works. If you don't understand the fix, you don't understand the bug.

## Relationship to Other Principles

[[principles/observe-directly]] says to check real state, not proxies. This principle extends that to debugging: check the real cause, not the proxied symptom.

[[principles/encode-lessons-in-structure]] provides the encoding mechanism once a root cause is understood — make it structurally impossible to recur.

See also [[principles/observe-directly]], [[principles/encode-lessons-in-structure]]
