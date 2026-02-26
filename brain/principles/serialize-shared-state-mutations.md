# Serialize Shared-State Mutations

**Principle:** When concurrent actors share mutable state, enforce serialization structurally — lockfiles, sequential phases, exclusive ownership. Instructions and conventions are insufficient for concurrency safety.

## Why

Concurrent writes to shared state (files, branches, APIs) produce race conditions that are intermittent, hard to reproduce, and expensive to debug. Telling agents or goroutines to "take turns" does not work — they have no coordination mechanism beyond the instruction itself.

## Pattern

Before allowing any parallel execution (agents, goroutines, sessions):

1. **Identify shared mutable state.** Files both read and write, branches both push to, APIs both define and consume.
2. **If shared state exists, serialize access.** Lockfiles, sequential phases, or exclusive ownership.
3. **If serialization is impractical, eliminate the sharing.** Give each actor its own copy (worktrees, separate files, isolated state directories).

## Evidence

- Concurrent worktree merges caused double-work → fixed with `.worktrees/.merge-lock` lockfile
- Parallel managers sharing an API contract invented conflicting versions → fix is sequential phases (producer then consumer)
- Non-blocking send to a full channel lost shutdown summary → shared channel without serialization
- Worktree merge lockfile uses PID-based stale lock detection → structural enforcement of exclusion

## Relationship to Other Principles

[[principles/make-operations-idempotent]] makes reruns safe; this principle prevents concurrent runs. They are complementary, not redundant.

[[principles/encode-lessons-in-structure]] is the meta-principle — this is what to encode when the lesson is about concurrency.

See also [[codebase/worktree-gotchas]], [[delegation/respect-api-contracts]]
