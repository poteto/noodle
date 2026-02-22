# Always Verify Your Work

**Principle:** Every task output must be verified before it's considered done. "It compiles," "it looks right," or "the file was created" is not verification.

## Why

Unverified work has unknown correctness. This applies to everything:
- Code that compiles but doesn't run correctly
- Skills that are syntactically valid but fail when actually used
- Scripts that look right but break on edge cases
- UI changes that render differently than expected

## Pattern

After completing any task, ask: **"How do I prove this actually works?"**

### Code / Features
1. Build it (necessary but not sufficient)
2. Run it and exercise the actual feature path
3. Write a bash script in `/tmp/` for complex multi-step test sequences
4. Check the full chain: does data flow from input to output?
5. For integrations (plugins, IPC, sockets), test the full communication path end-to-end
6. For UI changes, take a screenshot and verify visually (local only — CI runners are headless)

### Skills / Scripts / Tools
1. Spawn a subagent or agent team to actually use the skill end-to-end
2. Run the scripts with real (small) inputs and verify output
3. Test error paths: bad input, missing files, timeouts

### General
- If you can run it, run it
- If you can screenshot it, screenshot it
- If you can spawn a test agent, spawn one
- Prefer automated verification over manual inspection

See also [[principles/foundational-thinking]], [[principles/redesign-from-first-principles]], [[delegation/specify-verification-boundary]]

