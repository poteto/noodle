# Codex Review Is Valuable

## Pattern

Using Codex for static analysis review of code changes catches real issues that manual review misses. In one session, Codex review identified:

- **EPERM handling bug** — process liveness check treated EPERM as "dead" instead of "alive"
- **Busy-spin anti-pattern** — polling loop without sleep or backoff
- **Flaky test** — git worktree TempDir cleanup race condition

None of these were caught during manual review of the same code.

## When to Use

Codex review is most valuable for:

- **Systems code** with subtle correctness requirements (concurrency, signals, file locking)
- **Go/Rust code** where edge cases in error handling are easy to miss
- **Infrastructure changes** that affect reliability (lock files, process management, cleanup)

Less valuable for:

- UI/styling changes
- Documentation-only edits
- Simple mechanical refactors

## How

Spawn a Codex worker with a review-focused prompt: "Review this diff for correctness issues, edge cases, and anti-patterns. Focus on error handling, concurrency, and cleanup paths."

See also [[delegation/codex-scope-violations]], [[delegation/specify-verification-boundary]], [[principles/cost-aware-delegation]], [[codebase/unix-process-liveness-eperm]], [[codebase/worktree-gotchas]]
