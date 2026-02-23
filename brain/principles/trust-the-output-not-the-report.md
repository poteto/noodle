# Trust the Output, Not the Report

When verifying work done by another agent, tool, or system, inspect the actual output artifact (diff, file, runtime behavior) rather than trusting the agent's self-reported summary of what it did.

## The Rule

Always verify delegated work via artifacts — `git diff --stat`, file contents, runtime behavior — never via the delegate's summary of what they claim to have done.

## Why

Agents (human and AI) have blind spots. They report what they intended, not always what happened. Scope violations, unintended side effects, and silent failures are invisible in self-reports but obvious in artifacts.

## Distinction from Existing Principles

[[principles/observe-directly]] is about *system observation* — read actual state rather than inferring from proxies. This is about *delegation verification* — the specific pattern where the delegate's self-report is the proxy you must not trust. [[principles/verify-runtime]] says verify your own work at runtime; this says verify someone else's work via artifacts.

See also [[principles/observe-directly]], [[principles/verify-runtime]], [[delegation/codex-scope-violations]]
