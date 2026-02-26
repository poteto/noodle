Back to [[plans/59-subtract-go-logic-and-resilience/overview]]

# Phase 2: Wire domain_skill through dispatch

Covers: #64 (wiring + migration)

## Goal

Propagate `domain_skill` from frontmatter through the task registry to dispatch. Replace all hardcoded `taskType.Key == "execute"` checks with registry-driven domain skill lookups. Update the execute skill frontmatter to declare `domain_skill: backlog`.

The `domain_skill` value is a **skill name** (e.g., `"backlog"`), not an adapter key. The loop sets `req.DomainSkill` to this skill name; the dispatcher resolves it to a path via the skill resolver (existing `loadSkillBundle` path).

## Changes

- `internal/taskreg/registry.go` — Propagate `DomainSkill` from parsed frontmatter into the task type entry. When building the registry from discovered skills, copy `fm.Noodle.DomainSkill`.
- `loop/cook.go` (~line 102-106) — Replace `if taskType.Key == "execute" { ... adapter.Skill }` with `if taskType.DomainSkill != "" { req.DomainSkill = taskType.DomainSkill }`.
- `loop/control.go` (~line 309-313) — Same replacement in `controlRequestChanges()`.
- `internal/queuex/queue.go` (~line 193) — Third `taskType.Key == "execute"` hit. Replace with equivalent check using registry DomainSkill or the queue item's task key context.
- `dispatcher/tmux_dispatcher.go` (~line 130) and `dispatcher/sprites_dispatcher.go` (~line 102) — Both check `req.TaskKey == "execute"` to decide whether to load domain skill. Change to `req.DomainSkill != ""` so domain skill injection works for any task type that declares one.
- `.agents/skills/execute/SKILL.md` — Add `domain_skill: backlog` under the `noodle:` frontmatter block.

## Data structures

Task registry entry gains `DomainSkill string`, populated from `NoodleMeta.DomainSkill`.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical wiring with clear spec |

## Verification

### Static
- `go test ./internal/taskreg/... ./loop/... ./dispatcher/...` passes
- `go vet ./...` clean
- New test: registry built from skill with `domain_skill: backlog` → entry has DomainSkill set
- New test: cook dispatch with DomainSkill → `req.DomainSkill` set correctly
- New test: cook dispatch without DomainSkill → `req.DomainSkill` empty
- Grep for `taskType.Key == "execute"` and `req.TaskKey == "execute"` — zero hits remaining in domain-skill contexts

### Runtime
- `noodle start --once` → verify domain skill injected into cook session via execute skill frontmatter
