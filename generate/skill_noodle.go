// Package generate produces auto-generated files from source metadata.
package generate

//go:generate go run ./cmd/gen-skill

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"text/template"

	"github.com/poteto/noodle/cmdmeta"
	"github.com/poteto/noodle/config"
)

// ConfigField describes a single config field for documentation.
type ConfigField struct {
	TOMLKey     string
	Type        string
	Default     string
	Description string
	Section     string
}

// SkillData holds all template data for the noodle skill.
type SkillData struct {
	ConfigFields []ConfigField
	Commands     []cmdmeta.Command
}

// fieldDescriptions maps TOML paths to human-readable descriptions.
var fieldDescriptions = map[string]string{
	"mode":                        "Run mode governing schedule/dispatch/merge gates: auto (full automation), supervised (human approves merges), or manual (human triggers scheduling/dispatch/merge)",
	"routing.defaults.provider":   "Default LLM provider for cook sessions (claude or codex)",
	"routing.defaults.model":      "Default model name for cook sessions",
	"skills.paths":                "Ordered search paths for skill resolution",
	"adapters":                    "Adapter configs keyed by adapter name (e.g. backlog)",
	"concurrency.max_concurrency": "Maximum concurrent cook sessions",
	"agents.claude.path":          "Custom path to Claude Code binary",
	"agents.claude.args":          "Extra CLI arguments for Claude Code",
	"agents.codex.path":           "Custom path to Codex CLI binary",
	"agents.codex.args":           "Extra CLI arguments for Codex CLI",
	"runtime.default":             "Default runtime kind for spawning cooks (process, sprites, cursor)",
}

// GenerateSkillContent produces the full SKILL.md content.
func GenerateSkillContent() (string, error) {
	data := SkillData{
		ConfigFields: extractConfigFields(),
		Commands:     cmdmeta.Commands(),
	}

	tmpl, err := template.New("skill").Parse(skillTemplate)
	if err != nil {
		return "", fmt.Errorf("parse skill template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute skill template: %w", err)
	}

	return buf.String(), nil
}

func extractConfigFields() []ConfigField {
	defaults := config.DefaultConfig()
	defaultsVal := reflect.ValueOf(defaults)
	defaultsType := defaultsVal.Type()

	var fields []ConfigField
	for i := range defaultsType.NumField() {
		sf := defaultsType.Field(i)
		tomlTag := sf.Tag.Get("toml")
		if tomlTag == "" || tomlTag == "-" {
			continue
		}

		fv := defaultsVal.Field(i)
		fields = append(fields, flattenField("", tomlTag, sf.Type, fv)...)
	}

	sort.Slice(fields, func(i, j int) bool {
		return fields[i].TOMLKey < fields[j].TOMLKey
	})

	return fields
}

func flattenField(prefix, key string, t reflect.Type, v reflect.Value) []ConfigField {
	fullKey := key
	if prefix != "" {
		fullKey = prefix + "." + key
	}

	switch t.Kind() {
	case reflect.Struct:
		var fields []ConfigField
		for i := range t.NumField() {
			sf := t.Field(i)
			tag := sf.Tag.Get("toml")
			if tag == "" || tag == "-" {
				continue
			}
			fields = append(fields, flattenField(fullKey, tag, sf.Type, v.Field(i))...)
		}
		return fields

	case reflect.Map:
		desc := fieldDescriptions[fullKey]
		return []ConfigField{{
			TOMLKey:     fullKey,
			Type:        formatType(t),
			Default:     formatMapDefault(v),
			Description: desc,
		}}

	case reflect.Slice:
		desc := fieldDescriptions[fullKey]
		return []ConfigField{{
			TOMLKey:     fullKey,
			Type:        formatType(t),
			Default:     formatSliceDefault(v),
			Description: desc,
		}}

	default:
		desc := fieldDescriptions[fullKey]
		return []ConfigField{{
			TOMLKey:     fullKey,
			Type:        formatType(t),
			Default:     formatDefault(v),
			Description: desc,
		}}
	}
}

func formatType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int64:
		return "integer"
	case reflect.Float64:
		return "float"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice:
		return "array"
	case reflect.Map:
		return "table"
	default:
		return t.Kind().String()
	}
}

func formatDefault(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}
	switch v.Kind() {
	case reflect.String:
		s := v.String()
		if s == "" {
			return `""`
		}
		return fmt.Sprintf(`"%s"`, s)
	case reflect.Int, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Float64:
		return fmt.Sprintf("%.1f", v.Float())
	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

func formatSliceDefault(v reflect.Value) string {
	if !v.IsValid() || v.Len() == 0 {
		return "[]"
	}
	parts := make([]string, v.Len())
	for i := range v.Len() {
		parts[i] = fmt.Sprintf(`"%s"`, v.Index(i).String())
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func formatMapDefault(v reflect.Value) string {
	if !v.IsValid() || v.Len() == 0 {
		return "{}"
	}
	keys := make([]string, 0, v.Len())
	for _, k := range v.MapKeys() {
		keys = append(keys, k.String())
	}
	sort.Strings(keys)
	return fmt.Sprintf("{%s}", strings.Join(keys, ", "))
}

var skillTemplate = `---
name: noodle
description: >-
  Operate the Noodle CLI — explain commands, find flags, create/edit .noodle.toml config.
---

# Noodle

Noodle is an open-source AI coding framework. Skills are the only extension point. An LLM schedules work, Go code executes it mechanically. Everything is a file — orders-next.json, mise.json, control.ndjson. No hidden state.

## Task-Type Skill Frontmatter

Skills with a top-level ` + "`" + `schedule:` + "`" + ` field in their YAML frontmatter are discovered as task types by the scheduling loop. The schedule skill reads ` + "`" + `task_types[].schedule` + "`" + ` from mise to decide when to schedule each type.

` + "```" + `yaml
---
name: my-task-type
description: What this task type does
schedule: "When to schedule this task type"
---
` + "```" + `

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| ` + "`" + `schedule` + "`" + ` | yes | — | Hint for the schedule skill on when to schedule this type |

## Config Reference

Noodle reads ` + "`" + `.noodle.toml` + "`" + ` at project root. If missing, ` + "`" + `noodle start` + "`" + ` scaffolds a minimal one on first run.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
{{- range .ConfigFields}}
| ` + "`" + `{{.TOMLKey}}` + "`" + ` | {{.Type}} | {{.Default}} | {{.Description}} |
{{- end}}

### Minimal config

` + "```" + `toml
mode = "auto"  # run mode: auto | supervised | manual

[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"

[skills]
paths = [".agents/skills"]
` + "```" + `

For adapter config, see [references/adapters.md](references/adapters.md) and [references/config-schema.md](references/config-schema.md).

## Mode Contract

The ` + "`" + `mode` + "`" + ` config field sets the run mode. Three modes are supported:

| Mode | Schedule | Dispatch | Auto-merge |
|------|----------|----------|------------|
| ` + "`" + `auto` + "`" + ` | yes | yes | yes |
| ` + "`" + `supervised` + "`" + ` | yes | yes | no |
| ` + "`" + `manual` + "`" + ` | no | no | no |

Mode transitions are tracked with a monotonic ` + "`" + `mode_epoch` + "`" + `. Each transition increments the epoch. In-flight effects are epoch-stamped at creation; stale effects (created under a previous epoch) are cancelled rather than applied.

Key fields in the canonical state:

| Field | Description |
|-------|-------------|
| ` + "`" + `requested_mode` + "`" + ` | Mode requested by the operator |
| ` + "`" + `effective_mode` + "`" + ` | Mode currently governing behavior |
| ` + "`" + `mode_epoch` + "`" + ` | Monotonic counter; increments on every mode transition |

## Runtime Capabilities

Each runtime declares a capability set that determines polling, steering, sync, and heartbeat behavior:

| Capability | Description |
|------------|-------------|
| ` + "`" + `steerable` + "`" + ` | Session supports live message injection |
| ` + "`" + `polling` + "`" + ` | Session status must be polled (no push completion) |
| ` + "`" + `remote_sync` + "`" + ` | Session runs remotely and needs branch push/pull |
| ` + "`" + `heartbeat` + "`" + ` | Session emits periodic heartbeats for liveness |

Default capability profiles:

| Runtime | steerable | polling | remote_sync | heartbeat |
|---------|-----------|---------|-------------|-----------|
| ` + "`" + `process` + "`" + ` | yes | no | no | no |
| ` + "`" + `sprites` + "`" + ` | yes | no | no | no |
| ` + "`" + `cursor` + "`" + ` | no | yes | yes | no |

## Canonical State Model

The backend maintains canonical state with the following node hierarchy:

` + "```" + `
State
  orders: map[string]OrderNode
    order_id, status, stages[], created_at, updated_at, metadata
      StageNode
        stage_index, status, skill, runtime, attempts[], group
          AttemptNode
            attempt_id, session_id, status, started_at, completed_at, exit_code, worktree_name, error
  mode: RunMode (auto | supervised | manual)
  schema_version: int
  last_event_id: string
` + "```" + `

### Lifecycle enums

**Order:** pending, active, completed, failed, cancelled

**Stage:** pending, dispatching, running, merging, review, completed, failed, skipped, cancelled

**Attempt:** launching, running, completed, failed, cancelled

## Control Commands

| Action | Description |
|--------|-------------|
| ` + "`" + `pause` + "`" + ` | Pause the scheduling loop |
| ` + "`" + `resume` + "`" + ` | Resume a paused loop |
| ` + "`" + `drain` + "`" + ` | Stop scheduling; let active cooks finish |
| ` + "`" + `skip` + "`" + ` | Cancel an order |
| ` + "`" + `kill` + "`" + ` | Kill a running cook session |
| ` + "`" + `stop` + "`" + ` | Gracefully stop a cook (interrupt if steerable, kill if not) |
| ` + "`" + `stop-all` + "`" + ` | Kill all running cooks |
| ` + "`" + `steer` + "`" + ` | Inject a message into a running session (requires ` + "`" + `steerable` + "`" + ` capability) |
| ` + "`" + `merge` + "`" + ` | Approve merge for a pending-review order |
| ` + "`" + `reject` + "`" + ` | Reject a pending-review order |
| ` + "`" + `request-changes` + "`" + ` | Request changes on a pending-review order |
| ` + "`" + `enqueue` + "`" + ` | Add an ad-hoc order to the queue |
| ` + "`" + `requeue` + "`" + ` | Reset a failed order for retry |
| ` + "`" + `reorder` + "`" + ` | Move an order to a new position |
| ` + "`" + `edit-item` + "`" + ` | Edit a pending order's prompt, task key, or model |
| ` + "`" + `set-max-concurrency` + "`" + ` | Override concurrency limit at runtime |
| ` + "`" + `advance` + "`" + ` | Manually advance an order to its next stage |
| ` + "`" + `add-stage` + "`" + ` | Append a new stage to an existing order |
| ` + "`" + `park-review` + "`" + ` | Park an order for human review with a reason |

## Dispatch and Projection

Orders are dispatched through a planner that scans canonical state:

1. **PlanDispatches** — identifies dispatch candidates and blocked reasons (busy, capacity, failed, no pending stage)
2. **Two-phase launch** — launch record persisted as ` + "`" + `launching` + "`" + ` before process start, marked ` + "`" + `launched` + "`" + ` after session ID is known
3. **RouteCompletion** — applies attempt completion, triggers retry or failure routing
4. **AdvanceOrder** — marks post-merge progress and detects order completion

Projection writes external views:

| File | Content |
|------|---------|
| ` + "`" + `orders.json` + "`" + ` | Projected orders with stage status |
| ` + "`" + `state.json` + "`" + ` | Schema version marker |
| Snapshot API | Full projected state with mode, epoch, orders |
| WebSocket | Incremental deltas between projection versions |

## CLI Commands

| Command | Description |
|---------|-------------|
{{- range .Commands}}
{{- $parent := .Name}}
| ` + "`" + `noodle {{.Name}}` + "`" + ` | {{.Short}} |
{{- range .Subcommands}}
| ` + "`" + `noodle {{$parent}} {{.Name}}` + "`" + ` | {{.Short}} |
{{- end}}
{{- end}}

### Flags
{{range .Commands}}
{{- $parent := .Name}}
{{- if .Flags}}
` + "`" + `noodle {{.Name}}` + "`" + `:
{{range .Flags}}- ` + "`" + `--{{.Name}}` + "`" + `{{if .Short}} (` + "`" + `-{{.Short}}` + "`" + `){{end}} ({{.Type}}){{if .Default}}, default ` + "`" + `{{.Default}}` + "`" + `{{end}}: {{.Desc}}
{{end}}
{{- end}}
{{- range .Subcommands}}
{{- if .Flags}}
` + "`" + `noodle {{$parent}} {{.Name}}` + "`" + `:
{{range .Flags}}- ` + "`" + `--{{.Name}}` + "`" + `{{if .Short}} (` + "`" + `-{{.Short}}` + "`" + `){{end}} ({{.Type}}){{if .Default}}, default ` + "`" + `{{.Default}}` + "`" + `{{end}}: {{.Desc}}
{{end}}
{{- end}}
{{- end}}
{{- end}}

## Skill Management

Skills live in ` + "`" + `.agents/skills/` + "`" + ` by default. Paths in ` + "`" + `skills.paths` + "`" + ` are searched in order; first match wins. Install a skill by copying its directory to your skill path.

## Troubleshooting

Run ` + "`" + `noodle debug` + "`" + ` to dump the full runtime state. Common issues:

1. **"fatal config diagnostics prevent start"** — Run ` + "`" + `noodle debug` + "`" + `, fix fields in ` + "`" + `.noodle.toml` + "`" + `.
3. **Missing adapter scripts** — Create scripts or update paths in config.
4. **Stale worktrees** — ` + "`" + `noodle worktree list` + "`" + `, then ` + "`" + `noodle worktree prune` + "`" + `.

## References

- [references/config-schema.md](references/config-schema.md) — config validation
- [references/adapters.md](references/adapters.md) — adapter setup, script writing, provider examples
- [references/hooks.md](references/hooks.md) — brain injection hook, settings.json setup
`
