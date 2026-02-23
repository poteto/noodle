# Prove It Works

**Principle:** Every task output must be verified by checking the real thing directly — not by inferring from proxies, self-reports, or "it compiles."

## Why

Unverified work has unknown correctness. Indirect verification (file mtimes, output freshness, agent self-reports, cached screenshots) feels cheaper than direct observation, but acting on a wrong inference costs far more than checking the source.

## Pattern

After completing any task, ask: **"How do I prove this actually works?"**

### Check the real thing, not a proxy
- **Check process liveness directly** (PID, process table), not indirectly (tmux pane status, file mtime).
- **Read the actual value**, not a cached or derived representation.
- **When verification fails, suspect the observation method** before suspecting the system.

### Code / Features
1. Build it (necessary but not sufficient)
2. Run it and exercise the actual feature path
3. Check the full chain: does data flow from input to output?
4. For integrations (IPC, sockets), test the full communication path end-to-end

### Skills / Scripts / Tools
1. Spawn a subagent to actually use the skill end-to-end
2. Run scripts with real inputs and verify output
3. Test error paths: bad input, missing files, timeouts

### General
- If you can run it, run it
- Prefer automated verification over manual inspection

## Relationship to Other Principles

[[principles/trust-the-output-not-the-report]] applies this to delegation — verify someone else's work via artifacts, not their self-report.

[[principles/fix-root-causes]] extends this to debugging — check the real cause, not the proxied symptom.

See also [[principles/foundational-thinking]], [[delegation/specify-verification-boundary]]
