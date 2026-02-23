# Trust the Output, Not the Report

When verifying work done by another agent, tool, or system, inspect the actual output artifact (diff, file, runtime behavior) rather than trusting the agent's self-reported summary of what it did.

## The Rule

Always verify delegated work via artifacts — `git diff --stat`, file contents, runtime behavior — never via the delegate's summary of what they claim to have done.

## Why

Agents (human and AI) have blind spots. They report what they intended, not always what happened. Scope violations, unintended side effects, and silent failures are invisible in self-reports but obvious in artifacts.

## Distinction from Existing Principles

[[principles/prove-it-works]] says to check real state, not proxies. This is the delegation-specific application — the delegate's self-report is the proxy you must not trust.

See also [[principles/prove-it-works]], [[delegation/codex-scope-violations]]
