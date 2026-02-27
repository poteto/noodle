---
id: 38
created: 2026-02-24
status: done
---

# Resilient Skill Resolution

## Context

Noodle currently treats missing skills as fatal errors. If the prioritize skill is missing, the loop can't bootstrap. Any dispatch-time skill resolution failure kills the session. The runtime repair system was supposed to self-heal infrastructure failures but has never succeeded — all repair sessions fail with duration 0, and the repair state machine creates a double-spawn race condition where queue items fall out of all tracking maps.

The design goal: `noodle start` should work no matter what state the project is in. Transient failures recover via retry. Fatal failures bubble up to the human. No intermediate repair layer.

## Scope

**In:**
- Remove dead skill-name config fields from `.noodle.toml` (`phases`, `prioritize.skill`)
- Non-fatal skill resolution — dispatch proceeds without methodology, warns
- Remove runtime repair entirely — delete `runtime_repair.go`, fix the `pendingRetry` double-spawn race
- Prioritize bootstrap agent — inspects `.claude/`/`.codex/` history, creates prioritize skill
- fsnotify hot-reload of skill registry with debounce
- Queue audit after registry rebuild — drop stale items, warn via TUI feed
- Dispatch-time re-scan fallback (belt-and-suspenders for fsnotify)
- TUI feed events for skill lifecycle (dropped items, registry changes)

**Out:**
- Install-time prioritize bootstrapping (deferred — first `noodle start` is the natural moment)
- Rewriting the prioritize skill content itself
- Changes to skill discovery format (frontmatter stays the same)
- TUI notification system redesign (reuse existing feed infrastructure)
- Removing `adapters[*].skill` — this is domain context injection for execute tasks, not task-type→skill mapping. Different concept, separate concern.
- Removing `skills.paths` — still needed for non-standard skill locations

**Subsumes:** Todo #27 (hot-reload skill registry via fsnotify)

## Constraints

- fsnotify v1.9.0 is already a dependency; no built-in debouncing — manual implementation required
- Existing 200ms debounce pattern in `monitor/` can be reused
- `loop/loop.go` already uses fsnotify for queue/control file watches — extend, don't duplicate
- TUI feed system (`FeedEvent` with categories) is the notification mechanism — no new UI patterns
- Execute domain skill failures are already non-fatal (`skill_bundle.go:105-112`) — use as the model

## Alternatives considered

1. **Improve runtime repair** (original plan 38 phase 3) — make repair always available via hardcoded oops prompt. Rejected: repair has 0% success rate, adds state machine complexity, and causes double-spawn race. Subtract before you add.
2. **Config-driven fallbacks** — keep `prioritize.skill` config but add a fallback chain. Rejected: adds complexity to config when discovery already handles skill lookup.
3. **Virtual skills in resolver** — register built-in skills as in-memory entries. Rejected: conflates "user-provided skill files" with "hardcoded system prompts."
4. **Chosen: delete repair, caller-level fallback for bootstrap only** — remove runtime repair entirely. Bootstrap agent uses `SystemPrompt` field to dispatch without a skill file. All other error paths use retry or fatal exit.

## Applicable skills

- `go-best-practices` — goroutine lifecycle for watcher, channel patterns, shutdown
- `bubbletea-tui` — feed event rendering for skill lifecycle notifications

## Phases

- [[archive/plans/38-resilient-skill-resolution/phase-01-remove-dead-skill-config]]
- [[archive/plans/38-resilient-skill-resolution/phase-02-non-fatal-skill-resolution]]
- [[archive/plans/38-resilient-skill-resolution/phase-03-remove-runtime-repair]]
- [[archive/plans/38-resilient-skill-resolution/phase-04-fsnotify-skill-watcher]]
- [[archive/plans/38-resilient-skill-resolution/phase-05-registry-rebuild-queue-audit]]
- [[archive/plans/38-resilient-skill-resolution/phase-06-dispatch-rescan-fallback]]
- [[archive/plans/38-resilient-skill-resolution/phase-07-prioritize-bootstrap-agent]]
- [[archive/plans/38-resilient-skill-resolution/phase-08-tui-skill-lifecycle-events]]

## Verification

### Static

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

### Runtime — sandbox scenarios

Use `scripts/sandbox.sh` to test each resilience path in an isolated temp project.

**Bare repo (zero skills, zero config):**
```bash
cd "$(scripts/sandbox.sh bare)"
noodle start
```
Verify: loop doesn't crash. Bootstrap agent spawns, inspects `.claude/`/`.codex/`, creates prioritize skill, commits it. Next cycle runs real prioritization.

**Initialized project (scaffolding, no skills):**
```bash
cd "$(scripts/sandbox.sh init)"
noodle start
```
Verify: same bootstrap flow as bare, but `.noodle.toml` and `brain/` already exist. No duplicate scaffolding.

**WIP project (skills present, then deleted):**
```bash
cd "$(scripts/sandbox.sh wip)"
noodle start &
# Wait for first cycle, then delete a skill
rm -rf .agents/skills/quality
```
Verify: fsnotify triggers registry rebuild. Queue items referencing quality are dropped. TUI feed shows drop notification. Loop continues without crashing.

**WIP project (skill added mid-run):**
```bash
cd "$(scripts/sandbox.sh wip)"
noodle start &
# Add a new skill while running
mkdir -p .agents/skills/deploy && cat > .agents/skills/deploy/SKILL.md <<'EOF'
---
name: deploy
noodle:
  schedule: "Deploy changes to staging"
---
# Deploy
EOF
```
Verify: fsnotify picks up new skill. Next prioritize cycle can schedule deploy tasks.

**Spawn failure resilience:**
```bash
cd "$(scripts/sandbox.sh full)"
# Simulate spawn failures by misconfiguring provider
noodle start
```
Verify: failed spawns retry via `retryCook`. After `maxRetries`, item marked failed and skipped. Loop continues to next item. No repair sessions spawned. No double-spawn.
