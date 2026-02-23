Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 1: Dynamic Task Registry — Frontmatter-Based Discovery

## Goal

Replace the hardcoded Go task type registry with frontmatter-based discovery. Any skill with a `noodle:` block in its SKILL.md frontmatter becomes a task type. Built-in and user-defined task types follow the same convention.

## Current State

- `internal/taskreg/registry.go` — hardcoded constants (`TaskKeyPlan`, `TaskKeyExecute`, `TaskKeyVerify`, etc.) with a static map of ~12 `TaskType` entries
- `configuredTaskTypes()` merges base types with config skill overrides from adapter paths
- Adding a new task type requires Go code changes
- No user extensibility
- Skill resolver (`skill/resolver.go`) does NOT parse frontmatter — only checks SKILL.md existence
- Skill bundle (`spawner/skill_bundle.go`) does NOT strip frontmatter — agents see raw YAML in context

## Changes

### Frontmatter parser — create `skill/frontmatter.go`

Parse YAML frontmatter from SKILL.md using `gopkg.in/yaml.v3` (already in go.mod). A reference for frontmatter splitting exists at `internal/testutil/fixturedir/fixturedir.go:406-424` but that uses a hand-rolled key:value parser — use proper YAML.

```go
// Frontmatter is the parsed YAML header from a SKILL.md file.
type Frontmatter struct {
    Name        string      `yaml:"name"`
    Description string      `yaml:"description"`
    Model       string      `yaml:"model,omitempty"`
    Noodle      *NoodleMeta `yaml:"noodle,omitempty"` // nil = utility skill, non-nil = task type
}

// IsTaskType returns true if this skill has noodle: frontmatter.
func (f Frontmatter) IsTaskType() bool { return f.Noodle != nil }

// NoodleMeta is the noodle-specific scheduling metadata nested under `noodle:`.
// Two required fields: blocking and schedule. Runtime is optional.
type NoodleMeta struct {
    Blocking bool   `yaml:"blocking"`
    Schedule string `yaml:"schedule"`            // one-line guidance for the prioritize skill
    Runtime  string `yaml:"runtime,omitempty"`   // command template, empty = project default
}

// ParseFrontmatter extracts YAML frontmatter from markdown content.
// Returns parsed metadata, the body (content after closing ---), and any error.
// If no frontmatter is found, returns zero Frontmatter and full content as body.
func ParseFrontmatter(content []byte) (Frontmatter, []byte, error)

// StripFrontmatter removes the YAML frontmatter block, returning only the body.
func StripFrontmatter(content []byte) []byte
```

`Noodle` is a pointer — `nil` means no `noodle:` block (utility skill), non-nil means task type. This is how the resolver distinguishes task types from utility skills: `frontmatter.IsTaskType()`.

Tests (`skill/frontmatter_test.go`):
- Parse complete frontmatter with all `noodle:` fields → `IsTaskType() == true`
- Parse frontmatter without `noodle:` block → `IsTaskType() == false`, `Noodle == nil`
- Parse with partial `noodle:` (only `schedule`) → `Blocking` defaults false
- No frontmatter (no `---`) returns zero struct and full body
- Invalid YAML returns error
- `StripFrontmatter` preserves body, strips header

### Extended resolver — modify `skill/resolver.go`

Add `SkillMeta` and new methods. Existing `Resolve()` and `List()` stay for backward compat.

```go
// SkillMeta is the full resolved metadata for a skill, including parsed frontmatter.
type SkillMeta struct {
    Name        string
    Path        string      // absolute path to skill directory
    SourcePath  string      // search path that matched
    Frontmatter Frontmatter // parsed from SKILL.md
}

// ResolveWithMeta resolves a skill by name and parses its frontmatter.
func (r Resolver) ResolveWithMeta(name string) (SkillMeta, error)

// ListWithMeta returns all skills with parsed frontmatter.
func (r Resolver) ListWithMeta() ([]SkillMeta, error)

// DiscoverTaskTypes returns only skills with noodle: frontmatter (i.e., IsTaskType() == true).
func (r Resolver) DiscoverTaskTypes() ([]SkillMeta, error)
```

`ResolveWithMeta` calls `Resolve()` then `os.ReadFile` + `ParseFrontmatter`. `ListWithMeta` calls `List()` then parses each. `DiscoverTaskTypes` filters `ListWithMeta()` by `Frontmatter.IsTaskType()`.

Tests: add cases to `skill/resolver_test.go` using temp directories with SKILL.md files. Test that skills with `noodle:` are returned by `DiscoverTaskTypes` and those without are excluded.

### Rewrite registry — modify `internal/taskreg/registry.go`

**Delete:**
- All `TaskKey*` constants (lines 37-49)
- `baseTaskTypes` slice (lines 52-167)
- `configuredTaskTypes()` function (lines 265-283)
- `adapterConfiguredSkill()` function (lines 285-291)
- Old `New(config.Config) Registry` constructor and its `bySkill`, `byAlias` maps

**Replace with:**

```go
// TaskType is one schedulable task kind, discovered from skill frontmatter.
type TaskType struct {
    Key       string // skill name (e.g., "prioritize", "execute", "deploy")
    Blocking  bool
    Schedule  string // one-line guidance for prioritize skill
    Runtime   string // command template, empty = project default
    SkillPath string // absolute path to skill directory
}

type Registry struct {
    types []TaskType
    byKey map[string]TaskType
}

// NewFromSkills builds a registry from discovered skill metadata.
// Only skills with noodle: frontmatter are included.
func NewFromSkills(skills []skill.SkillMeta) Registry

func (r Registry) All() []TaskType
func (r Registry) ByKey(key string) (TaskType, bool)

// ResolveQueueItem resolves a queue item to its task type.
// Resolution order: task_key → skill name → id → id prefix before "-".
func (r Registry) ResolveQueueItem(item QueueItemInput) (TaskType, bool)
```

Resolution is simpler — no alias lookup. The skill name IS the key.

Tests: rewrite `internal/taskreg/registry_test.go` to test `NewFromSkills()` with mock `SkillMeta` slices. Test `ResolveQueueItem` with task_key, skill, id, and prefix resolution.

### Add `RuntimeConfig` — modify `config/config.go`

```go
type RuntimeConfig struct {
    Default string `toml:"default"` // command template, empty = built-in tmux
}
```

Add `Runtime RuntimeConfig` to `Config` struct. Default: empty string (built-in tmux logic).

### Update mise brief — modify `mise/types.go` + `mise/builder.go`

New type in `mise/types.go`:

```go
type TaskTypeSummary struct {
    Key      string `json:"key"`
    Blocking bool   `json:"blocking"`
    Schedule string `json:"schedule"`
}
```

Add `TaskTypes []TaskTypeSummary` field to `Brief`.

The `Builder` struct gains a `resolver skill.Resolver` field. `NewBuilder` signature changes to accept the resolver. In `Build()`, call `resolver.DiscoverTaskTypes()` and populate `brief.TaskTypes`.

### Add `noodle:` frontmatter to built-in skills

Add frontmatter to each built-in task type's SKILL.md (all in `.agents/skills/`):

| Skill | blocking | schedule |
|-------|----------|----------|
| prioritize | true | "When the queue is empty, after backlog changes, or when session history suggests re-evaluation" |
| execute | false | "When a planned backlog item is ready for implementation" |
| reflect | false | "After a cook session completes" |
| meditate | false | "Periodically after several reflect cycles have accumulated" |

Skills that don't exist yet (quality, oops, debate) get frontmatter when created in their respective phases.

All built-in tasks omit `runtime` (inherit project default). Quality review sequencing is handled by the prioritize skill — the quality skill's own `schedule` field says "After a cook session completes, before reflecting" and the prioritize skill decides when to run it.

### Update loop — modify `loop/types.go` + `loop/loop.go`

`Loop` struct gains `registry taskreg.Registry` field, built at `New()` time from `resolver.DiscoverTaskTypes()`. Helper functions in `loop/task_types.go` change from `configuredTaskTypes()` to `l.registry` lookups.

## Data Structures

- `skill.Frontmatter` — parsed YAML header from SKILL.md
- `skill.NoodleMeta` — nested `noodle:` block with `blocking`, `schedule`, `runtime`
- `skill.SkillMeta` — skill path + parsed frontmatter
- `taskreg.TaskType` — discovered task type (key, blocking, schedule, runtime, skill_path)
- `taskreg.Registry` — indexed lookup by key
- `mise.TaskTypeSummary` — task type entry in the mise brief
- `config.RuntimeConfig` — project-level runtime default

## Verification

- `make ci` passes
- `go test ./skill/...` — frontmatter parsing, stripping, resolver discovery
- `go test ./internal/taskreg/...` — registry from skills, queue item resolution
- `go test ./mise/...` — brief includes discovered task types
- Skills with `noodle:` frontmatter discovered; skills without it excluded
- No hardcoded `TaskKey*` constants remain
- User-defined task type (test skill with `noodle:` frontmatter) discovered alongside built-ins
