---
name: go-best-practices
description: >
  Production Go patterns — lifecycle, concurrency, config, testing, CI. Use when
  writing, reviewing, or refactoring Go code. Triggers on Go code review, Go refactoring,
  or 'go best practices'.
---

# Go Best Practices

| Pattern | When to apply |
|---------|---------------|
| Minimal main | New binaries |
| Single bootstrap | App wiring across modes |
| Ordered shutdown | Long-running processes |
| Non-blocking fanout | Channel-based event systems |
| Concurrency testing | Any goroutine code |
| Layered config | Multi-source configuration |
| Cross-platform paths | File/config path resolution |
| Secure debug logging | HTTP client instrumentation |
| Golden test matrices | Rendering / output components |
| Focused linters | CI pipeline setup |

Read [references/patterns.md](references/patterns.md) for code examples.

## Rules

1. **main.go is a stub.** Call `cmd.Execute()` and nothing else. Gate
   diagnostics (pprof) behind env vars.

2. **One bootstrap path.** All modes (interactive, headless, test) share the
   same initialization function. Two paths will drift.

3. **Ordered shutdown.** Cancel dependents before their dependencies, then run
   independent cleanup in parallel under a timeout context. WaitGroup the
   parallel phase.

4. **Never block the sender.** Non-blocking channel sends with an explicit
   drop policy. Log drops at debug level. Document the policy.

5. **Test concurrency mechanically.** `testing/synctest` for deterministic
   timing, `goleak.VerifyNone` for leak detection, dedicated regression tests
   for timer/channel deadlocks.

6. **Explicit config precedence.** Global → project → flags. Walk up from CWD
   to discover project configs, reverse so closest wins, deep-merge.

7. **Centralize platform paths.** One function per concern. Resolution order:
   env override → XDG → platform default → fallback.

8. **Redact secrets in debug logs.** Wrap `http.RoundTripper`. Filter headers
   matching authorization, api-key, token, secret. Gate on debug level to
   skip allocation in prod.

9. **Golden tests over matrices.** Cross dimensions (layout × theme × size)
   as parallel subtests. Sweep continuous ranges to catch off-by-one bugs.

10. **Focused linters, not all linters.** Enable what catches real bugs
    (bodyclose, noctx, tparallel). Disable noisy defaults. Add project-specific
    checks as scripts. Always `-race` in tests.
