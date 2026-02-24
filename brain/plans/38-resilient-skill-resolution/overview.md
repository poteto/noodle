---
id: 38
created: 2026-02-24
status: active
---

# Resilient Skill Resolution

## Context

Noodle currently treats missing skills as fatal errors. If the prioritize skill is missing, the loop can't bootstrap. If oops/debugging skills are missing, runtime repair is impossible. Any dispatch-time skill resolution failure kills the session. This creates a fragile system that can't self-heal.

The design goal: `noodle start` should work no matter what state the project is in. The only truly fatal error is "cannot spawn an agent at all" (tmux dead, binary missing). Everything else is recoverable by spawning an agent to fix it.

## Scope

**In:**
- Remove dead skill-name config fields from `.noodle.toml` (`phases`, `prioritize.skill`)
- Non-fatal skill resolution — dispatch proceeds without methodology, warns
- Built-in oops fallback prompt (hardcoded, user skill override wins)
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

1. **Config-driven fallbacks** — keep `prioritize.skill` config but add a fallback chain. Rejected: adds complexity to config when discovery already handles skill lookup. Subtract before you add.
2. **Virtual skills in resolver** — register built-in skills as in-memory entries in the resolver. Rejected: conflates "user-provided skill files" with "hardcoded system prompts." Cleaner to have the caller decide.
3. **Chosen: caller-level fallback + prompt injection** — runtime_repair.go tries resolver first, falls back to hardcoded prompt. Dispatcher gets a `SystemPrompt` field to skip resolution when caller provides prompt directly. Clean separation between user skills and built-in behavior.

## Applicable skills

- `go-best-practices` — goroutine lifecycle for watcher, channel patterns, shutdown
- `bubbletea-tui` — feed event rendering for skill lifecycle notifications

## Phases

- [[plans/38-resilient-skill-resolution/phase-01-remove-dead-skill-config]]
- [[plans/38-resilient-skill-resolution/phase-02-non-fatal-skill-resolution]]
- [[plans/38-resilient-skill-resolution/phase-03-builtin-oops-fallback]]
- [[plans/38-resilient-skill-resolution/phase-04-fsnotify-skill-watcher]]
- [[plans/38-resilient-skill-resolution/phase-05-registry-rebuild-queue-audit]]
- [[plans/38-resilient-skill-resolution/phase-06-dispatch-rescan-fallback]]
- [[plans/38-resilient-skill-resolution/phase-07-prioritize-bootstrap-agent]]
- [[plans/38-resilient-skill-resolution/phase-08-tui-skill-lifecycle-events]]

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

**Full project (runtime repair without oops skill):**
```bash
cd "$(scripts/sandbox.sh full)"
rm -rf .agents/skills/quality  # remove a skill referenced in queue
# Corrupt queue.json to trigger runtime repair
echo '[{"target":"bad","task_type":"nonexistent"}]' > .noodle/queue.json
noodle start
```
Verify: built-in oops prompt handles the repair. No user-provided oops skill needed.

**Full project (user oops skill overrides built-in):**
```bash
cd "$(scripts/sandbox.sh full)"
mkdir -p .agents/skills/oops
cat > .agents/skills/oops/SKILL.md <<'EOF'
---
name: oops
noodle:
  schedule: "Fix runtime issues"
---
# Custom Oops
My project-specific repair instructions.
EOF
noodle start &
# Corrupt queue.json to trigger runtime repair
echo '{"items":[{"id":"x","task_key":"nonexistent"}]}' > .noodle/queue.json
```
Verify: user's oops skill is used instead of built-in fallback.
