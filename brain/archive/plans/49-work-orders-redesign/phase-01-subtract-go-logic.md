Back to [[archive/plans/49-work-orders-redesign/overview]]

# Phase 1: Subtract Go logic

Covers: #59, #62, #64 (type only)

## Goal

Subtract hardcoded Go logic and add resilience before the orders migration. Three independent changes that simplify the substrate the migration builds on.

1. **Domain skill frontmatter type** — Add `DomainSkill` field to `NoodleMeta` so task types can declare domain context needs. Dispatch wiring happens in phase 5 when cook.go is rewritten.
2. **Bootstrap as skill file** — Extract the hardcoded bootstrap prompt into `.agents/skills/bootstrap/SKILL.md`. Wire dispatch through the skill resolver. Fix silent exhaustion.
3. **Sync script graceful degradation** — Missing sync script degrades to empty backlog with warning instead of crashing the cycle.

## Changes

### 1. Domain skill frontmatter type (#64 foundation)

**`skill/frontmatter.go`** — Add `DomainSkill string` to `NoodleMeta`:
```go
DomainSkill string `yaml:"domain_skill,omitempty"`
```

**`skill/frontmatter_test.go`** — Test parsing `domain_skill` from frontmatter YAML (present and absent cases).

**`internal/taskreg/registry.go`** — Propagate `DomainSkill` from parsed frontmatter into the task type registry entry. When building the registry from discovered skills, copy `fm.Noodle.DomainSkill`.

**`.agents/skills/execute/SKILL.md`** — Add `domain_skill: backlog` under the `noodle:` frontmatter block. This declares the execute skill's dependency on the backlog adapter for domain context.

### 1b. Remove hardcoded "execute" fallback in queuex

**`internal/queuex/queue.go`** (~line 168) — Delete the fallback that silently coerces items with no `task_key` and no `skill` to the `"execute"` task type (`if resolved, found := reg.ByKey("execute"); found { ... }`). This is hardcoded scheduling logic inside the validation layer — items without a valid task key should be rejected, not silently defaulted. This subtraction makes the validation honest before the orders migration rewrites it.

### 2. Bootstrap as skill file (#59)

**`.agents/skills/bootstrap/SKILL.md`** — New file. Extract prompt content from `bootstrapPromptTemplate` in `loop/builtin_bootstrap.go`. Add frontmatter (`name: bootstrap`, `description: ...`). Keep `{{history_dirs}}` template variable — the loop substitutes it before dispatch. Use `skill-creator` skill.

**`loop/builtin_bootstrap.go`** — Delete `bootstrapPromptTemplate` constant and `buildBootstrapPrompt()`. Replace with skill resolver lookup: resolve "bootstrap" skill, read content, strip frontmatter, substitute `{{history_dirs}}`, pass as `req.SystemPrompt`. If skill is missing, log actionable error ("bootstrap skill not found — create .agents/skills/bootstrap/SKILL.md or run noodle init") and return error. **Do NOT update `queue-next.json` → `orders-next.json` path references here** — the orders file format doesn't exist until phase 3 and the loop doesn't consume `orders-next.json` until phase 5. The path update happens in phase 5 alongside the loop migration.

**`loop/schedule.go`** (~lines 109-115) — Replace silent `return nil` on `bootstrapExhausted` with logged warning: "bootstrap exhausted after N attempts — create .agents/skills/schedule/SKILL.md manually or check bootstrap skill output for errors". Emit feed event for web UI visibility.

**`loop/cook.go`** (~lines 192-195) — On bootstrap failure, log the failure reason (not just increment counter).

**`loop/bootstrap_test.go`** — Rewrite tests:
- `TestBootstrapPromptContainsHistoryDirForProvider` — test skill-file-based loading with `{{history_dirs}}` substitution
- `TestBootstrapSessionUsesSystemPromptNotSkill` and `TestSystemPromptFieldOnDispatchRequest` — add bootstrap skill fixture to test temp dirs
- New test: missing bootstrap skill → actionable error
- New test: bootstrap exhaustion → warning logged

### 3. Sync script graceful degradation (#62)

**`loop/loop.go`** (~lines 378-382) — Replace `return Queue{}, false, fmt.Errorf(...)` with: log warning, emit feed event, continue with empty queue.

**`loop/util.go`** (~lines 196-210) — Delete or simplify `shouldRecoverMissingSyncScripts()`. With graceful degradation, recovery concept is no longer needed.

## Routing

| Subsection | Provider | Model | Rationale |
|------------|----------|-------|-----------|
| Domain skill type | `codex` | `gpt-5.4` | Mechanical type addition |
| Bootstrap skill file | `claude` | `claude-opus-4-6` | Skill writing — prompt quality matters |
| Bootstrap wiring | `codex` | `gpt-5.4` | Mechanical wiring with clear spec |
| Sync degradation | `codex` | `gpt-5.4` | Small behavioral change |

## Verification

### Static
- `go test ./skill/... ./internal/taskreg/... ./loop/...` passes
- `go vet ./...` clean
- Grep for `bootstrapPromptTemplate` — zero hits
- Grep for `shouldRecoverMissingSyncScripts` — zero hits (or simplified)
- Grep for `reg.ByKey("execute")` in `internal/queuex/` — zero hits (hardcoded fallback removed)
- Bootstrap skill file exists with valid frontmatter and `{{history_dirs}}` placeholder

### Runtime
- New test: parse frontmatter with `domain_skill: backlog` under `noodle:`, assert field populated
- New test: registry built from skill with `domain_skill: backlog` → entry has DomainSkill set
- New test: bootstrap dispatch loads skill content from resolver
- New test: missing bootstrap skill → actionable error logged
- New test: bootstrap exhaustion → warning logged with next steps
- New test: bootstrap exhaustion → feed event emitted with attempt count and next-steps payload
- New test: mise build with missing sync script → returns brief with empty backlog + warning (not error)
- New test: mise build with present-but-failing sync script (non-zero exit) → returns brief with empty backlog + warning (not error)
- New test: cycle continues after missing sync script (not halted)
- New test: cycle continues after failing sync script (not halted)
