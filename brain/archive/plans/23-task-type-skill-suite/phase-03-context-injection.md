Back to [[archive/plans/23-task-type-skill-suite/overview]]

# Phase 3: Dispatcher + Context Injection

## Goal

Rename `spawner` → `dispatcher`, add frontmatter stripping, context preamble injection, execute session assembly, and runtime command template support. The runtime template controls what command runs inside a tmux session (launch-only) — the dispatcher still handles tmux lifecycle (monitoring, stopping, events).

## Current State

- `spawner/` package: 13 files — `tmux_spawner.go`, `skill_bundle.go`, `tmux_session.go`, `tmux_command.go`, `spawn_metadata.go`, `types.go`, plus tests
- `spawner.Spawner` interface: `Spawn(ctx, SpawnRequest) (Session, error)`
- `spawner.TmuxSpawner` — concrete implementation, builds provider CLI command piped to `noodle stamp`, launches in tmux
- `spawner.SpawnRequest` — Name, Prompt, Provider, Model, Skill, WorktreePath, etc.
- `spawner.Session` interface — ID, Status, Events, Done, TotalCost, Kill
- `skill_bundle.go:loadSkillBundle()` does NOT strip frontmatter — agents see raw YAML
- `tmux_command.go` builds provider-specific CLI commands (`buildClaudeCommand`, `buildCodexCommand`) piped to `noodle stamp`
- Loop depends on `Spawner` interface in `loop/types.go` for test mocking
- `cmd_spawn.go` exposes CLI `noodle spawn` command
- 10+ files import `spawner` package

## Changes

### Package rename: `spawner/` → `dispatcher/`

Mechanical rename of all files:

| From | To |
|------|-----|
| `spawner/types.go` | `dispatcher/types.go` |
| `spawner/tmux_spawner.go` | `dispatcher/tmux_dispatcher.go` |
| `spawner/skill_bundle.go` | `dispatcher/skill_bundle.go` |
| `spawner/tmux_session.go` | `dispatcher/tmux_session.go` |
| `spawner/tmux_command.go` | `dispatcher/tmux_command.go` |
| `spawner/spawn_metadata.go` | `dispatcher/dispatch_metadata.go` |
| `spawner/fixture_test.go` | `dispatcher/fixture_test.go` |
| All `*_test.go` files | Same pattern |

Type renames within the package:

| Old | New |
|-----|-----|
| `SpawnRequest` | `DispatchRequest` |
| `TmuxSpawner` | `TmuxDispatcher` |
| `TmuxSpawnerConfig` | `TmuxDispatcherConfig` |
| `NewTmuxSpawner` | `NewTmuxDispatcher` |
| `spawnMetadata` | `dispatchMetadata` |
| `writeSpawnMetadata` | `writeDispatchMetadata` |

Import updates — every file importing `"<module>/spawner"` → `"<module>/dispatcher"`:
- `loop/types.go`, `loop/loop.go`, `loop/prioritize.go`, `loop/reconcile.go`, `loop/runtime_repair.go`, `loop/defaults.go`, `loop/fixture_test.go`
- `cmd_spawn.go` → `cmd_dispatch.go` (file rename too)
- `cmd_spawn_test.go` → `cmd_dispatch_test.go`
- `main.go` (if it references spawner)
- `fixtures_immutability_test.go`

Loop interface rename in `loop/types.go`:

```go
// Old:
type Spawner interface {
    Spawn(ctx context.Context, req spawner.SpawnRequest) (spawner.Session, error)
}

// New:
type Dispatcher interface {
    Dispatch(ctx context.Context, req dispatcher.DispatchRequest) (dispatcher.Session, error)
}
```

`Dependencies.Spawner` → `Dependencies.Dispatcher`. All loop call sites change from `l.deps.Spawner.Spawn(...)` to `l.deps.Dispatcher.Dispatch(...)`.

### Frontmatter stripping — modify `dispatcher/skill_bundle.go`

At line ~34 of current `loadSkillBundle`, SKILL.md is read and injected raw:

```go
// Before (current):
sections := []string{..., string(skillMarkdown), ...}

// After:
body := skill.StripFrontmatter(skillMarkdown)
sections := []string{..., string(body), ...}
```

`skill.StripFrontmatter` is from Phase 1's `skill/frontmatter.go`.

### Context preamble — create `dispatcher/preamble.go`

Static markdown template listing `.noodle/` state files and conventions:

```go
const noodlePreamble = `# Noodle Context
You are running inside a Noodle cook session...
## State Files
- .noodle/mise.json — ...
- .noodle/queue.json — ...
- .noodle/tickets.json — ...
- .noodle/quality/ — ...
- brain/plans/ — ...
- brain/todos.md — ...
## Conventions
- Work in your assigned worktree...
- Commit with conventional commit messages...
- Run verification before finishing...
`

func buildSessionPreamble() string { return noodlePreamble }
```

Injected in `Dispatch()` before skill content:

```go
preamble := buildSessionPreamble()
skillBundle, _ := loadSkillBundle(...)
fullSystemPrompt := preamble + "\n\n" + skillBundle.SystemPrompt
systemPrompt, finalPrompt := composePrompts(req.Provider, req.Prompt, fullSystemPrompt)
```

### Runtime command template — modify `dispatcher/tmux_dispatcher.go`

`DispatchRequest` gains new fields:

```go
type DispatchRequest struct {
    // ... existing fields (renamed from SpawnRequest) ...
    TaskKey     string // e.g., "execute", "prioritize" — from resolved task type
    DomainSkill string // for execute: adapter-configured skill (e.g., "todo")
    Runtime     string // command template from frontmatter, empty = project default
}
```

`TmuxDispatcherConfig` gains:

```go
type TmuxDispatcherConfig struct {
    // ... existing fields (renamed from TmuxSpawnerConfig) ...
    RuntimeDefault string // from config.Runtime.Default
}
```

The runtime template is **launch-only** — it controls what command runs inside a tmux session. The dispatcher still creates the tmux session, monitors it, and manages lifecycle.

Resolution in `Dispatch()`:

```go
func (d *TmuxDispatcher) Dispatch(ctx context.Context, req DispatchRequest) (Session, error) {
    // ... validation, session ID, worktree (same as before) ...

    // Load skill bundle (with frontmatter stripping)
    bundle := loadSkillBundle(d.skillResolver, req.Provider, req.Skill)
    // OR for execute: loadExecuteBundle(...)

    // Resolve runtime command
    runtimeCmd := d.resolveRuntime(req)

    if runtimeCmd == "" {
        // Built-in path: existing buildClaudeCommand/buildCodexCommand + noodle stamp pipeline
        cmd := buildProviderCommand(req, bundle, sessionDir)
        pipeline := buildPipelineCommand(cmd, sessionDir)
        // ... launch tmux with pipeline (existing logic)
    } else {
        // Custom runtime path: resolve template, run inside tmux
        vars := map[string]string{
            "session": sessionID,
            "repo":    worktreePath,
            "prompt":  filepath.Join(sessionDir, "prompt.txt"),
            "skill":   req.Skill,
            "brief":   filepath.Join(d.runtimeDir, "mise.json"),
        }
        resolved := resolveTemplateVars(runtimeCmd, vars)
        // Both paths pipe through noodle stamp — the dispatcher always appends it.
        // Runtime templates must NOT include stamp piping.
        pipeline := buildPipelineCommand(resolved, sessionDir)
        // ... launch tmux with pipeline (same tmux logic, different command)
    }
}

func (d *TmuxDispatcher) resolveRuntime(req DispatchRequest) string {
    if req.Runtime != "" { return req.Runtime }
    return d.runtimeDefault
}

// resolveTemplateVars replaces {{key}} placeholders with verbatim values.
// No shell quoting — the template author controls quoting in their template.
// This matches envsubst/docker semantics: placeholders are pure substitution.
// Built-in variables (session, repo, prompt, skill, brief) are paths controlled
// by Noodle, not user input, so injection risk is minimal.
func resolveTemplateVars(template string, vars map[string]string) string {
    result := template
    for key, value := range vars {
        result = strings.ReplaceAll(result, "{{"+key+"}}", value)
    }
    return result
}
```

Template variables:

| Variable | Value |
|----------|-------|
| `{{session}}` | tmux session name (e.g., `noodle-execute-20260222-abc`) |
| `{{repo}}` | worktree path |
| `{{prompt}}` | path to prompt.txt in session dir |
| `{{skill}}` | skill name |
| `{{brief}}` | path to `.noodle/mise.json` |

### Execute session assembly — modify `dispatcher/skill_bundle.go`

New function for loading two skills:

```go
// loadExecuteBundle loads execute methodology + adapter-configured domain skill.
func loadExecuteBundle(
    resolver skill.Resolver,
    provider string,
    methodologySkill string, // "execute"
    domainSkill string,      // e.g., "todo" from config.Adapters["backlog"].Skill
) (loadedSkill, error) {
    methodology, err := loadSkillBundle(resolver, provider, methodologySkill)
    if err != nil {
        return loadedSkill{}, fmt.Errorf("load methodology skill %s: %w", methodologySkill, err)
    }
    if domainSkill == "" || domainSkill == methodologySkill {
        return methodology, nil
    }
    domain, err := loadSkillBundle(resolver, provider, domainSkill)
    if err != nil {
        // Not fatal — execute works standalone
        methodology.Warnings = append(methodology.Warnings,
            fmt.Sprintf("domain skill %q not found: %v", domainSkill, err))
        return methodology, nil
    }
    return loadedSkill{
        SystemPrompt: methodology.SystemPrompt + "\n\n" + domain.SystemPrompt,
        Warnings:     append(methodology.Warnings, domain.Warnings...),
    }, nil
}
```

The dispatcher checks `req.TaskKey` in `Dispatch()`:

```go
var bundle loadedSkill
if req.TaskKey == "execute" && req.DomainSkill != "" {
    bundle, err = loadExecuteBundle(d.skillResolver, req.Provider, req.Skill, req.DomainSkill)
} else {
    bundle, err = loadSkillBundle(d.skillResolver, req.Provider, req.Skill)
}
```

### Loop changes — modify `loop/loop.go`

`spawnCook()` passes the new fields:

```go
func (l *Loop) spawnCook(ctx context.Context, item QueueItem, ...) error {
    taskType, ok := l.registry.ResolveQueueItem(taskreg.QueueItemInput{
        ID: item.ID, TaskKey: item.TaskKey, Skill: item.Skill,
    })

    // ... worktree, name logic (unchanged) ...

    req := dispatcher.DispatchRequest{
        Name:         name,
        Prompt:       prompt,
        Provider:     nonEmpty(item.Provider, l.config.Routing.Defaults.Provider),
        Model:        nonEmpty(item.Model, l.config.Routing.Defaults.Model),
        Skill:        item.Skill,
        WorktreePath: worktreePath,
        TaskKey:      taskType.Key,
        Runtime:      taskType.Runtime,
    }

    // Execute session assembly: load domain skill alongside methodology
    if ok && taskType.Key == "execute" {
        if adapter, exists := l.config.Adapters["backlog"]; exists {
            req.DomainSkill = adapter.Skill
        }
    }

    session, err := l.deps.Dispatcher.Dispatch(ctx, req)
    // ...
}
```

### Skill-specific schema references

Each task-type skill that reads/writes `.noodle/` state includes schema docs in `references/`:

| Skill | References |
|-------|------------|
| prioritize | `references/mise-schema.md`, `references/queue-schema.md` |
| quality | `references/verdict-schema.md` |
| debate | `references/debate-state-schema.md` |

These are markdown files documenting the JSON schemas. Created alongside each skill in Phases 4-10.

## Data Structures

- `dispatcher.DispatchRequest` — renamed `SpawnRequest` + `TaskKey`, `DomainSkill`, `Runtime` fields
- `dispatcher.TmuxDispatcher` — renamed `TmuxSpawner`, resolves runtime template
- `dispatcher.TmuxDispatcherConfig` — renamed, adds `RuntimeDefault`
- `dispatcher.Session` — unchanged interface
- `loop.Dispatcher` — renamed `Spawner` interface
- Preamble: static markdown constant in `dispatcher/preamble.go`
- Runtime template variables: `map[string]string` resolved by `resolveTemplateVars()`

## Verification

- `make ci` passes
- No remaining references to `spawner` or `Spawner` in Go code (grep verification)
- `go test ./dispatcher/...` passes — all existing spawner tests pass after rename
- Frontmatter stripped: test that `loadSkillBundle` output does NOT contain `---` YAML markers
- Preamble present: test that session system prompt starts with "# Noodle Context"
- Execute bundle: test `loadExecuteBundle` produces combined prompt with methodology first, domain second
- Runtime template: test `resolveTemplateVars` substitutes all variables verbatim (no quoting added)
- Custom runtime: test that non-empty `req.Runtime` changes the command inside tmux
- Default runtime: test that empty `req.Runtime` uses built-in provider command logic
- Loop passes `TaskKey`, `Runtime`, `DomainSkill` through `DispatchRequest`
