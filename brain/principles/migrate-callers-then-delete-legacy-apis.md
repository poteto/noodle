# Migrate Callers, Then Delete Legacy APIs

**Principle:** When we decide a new API is the right design, migrate callers and remove the old API in the same refactor wave instead of preserving compatibility layers.

## Rule

- Do not keep legacy API paths alive just because internal callers still exist.
- Inventory callers, migrate them, and delete the old API immediately.
- Treat temporary adapters as exceptional and time-boxed, not default architecture.
- Update tests to assert the new contract, and delete tests that only protect pre-refactor implementation details.

## When This Applies

- No external users depend on backward compatibility.
- The project can absorb coordinated breaking changes.
- The new API is part of a simplification/refactor initiative.

## Why

Keeping both old and new APIs creates dual-path complexity, slows cleanup, and makes the codebase feel append-only. Migrating callers and deleting legacy APIs preserves elegance and reinforces subtractive refactoring.

See also [[principles/subtract-before-you-add]] and [[principles/redesign-from-first-principles]].
