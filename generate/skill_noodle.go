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
	"autonomy":                      "How much human oversight the loop requires: auto or approve",
	"routing.defaults.provider":     "Default LLM provider for cook sessions (claude or codex)",
	"routing.defaults.model":        "Default model name for cook sessions",
	"routing.tags":                  "Per-tag model overrides keyed by tag name",
	"skills.paths":                  "Ordered search paths for skill resolution",
	"schedule.run":                  "When to run scheduling: after-each, after-n, or manual",
	"schedule.model":                "Model used for scheduling sessions",
	"adapters":                      "Adapter configs keyed by adapter name (e.g. backlog)",
	"recovery.max_retries":          "Maximum retry attempts for failed cooks",
	"monitor.stuck_threshold":       "Duration before a cook is considered stuck",
	"monitor.ticket_stale":          "Duration before a ticket is considered stale",
	"monitor.poll_interval":         "How often the monitor checks session status",
	"concurrency.max_cooks":         "Maximum concurrent cook sessions",
	"agents.claude.path":            "Custom path to Claude Code binary",
	"agents.claude.args":            "Extra CLI arguments for Claude Code",
	"agents.codex.path":             "Custom path to Codex CLI binary",
	"agents.codex.args":             "Extra CLI arguments for Codex CLI",
	"runtime.default":               "Default runtime command template for spawning cooks",
	"plans.on_done":                 "What to do with completed plans: keep or remove",
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

Skills with a ` + "`" + `noodle:` + "`" + ` block in their YAML frontmatter are discovered as task types by the scheduling loop. The schedule skill reads ` + "`" + `task_types[].schedule` + "`" + ` from mise to decide when to schedule each type.

` + "```" + `yaml
---
name: my-task-type
description: What this task type does
noodle:
  schedule: "When to schedule this task type"
  permissions:
    merge: true
---
` + "```" + `

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| ` + "`" + `noodle.schedule` + "`" + ` | yes | — | Hint for the schedule skill on when to schedule this type |
| ` + "`" + `noodle.permissions.merge` + "`" + ` | no | ` + "`" + `true` + "`" + ` | Auto-merge worktree on success. Set ` + "`" + `false` + "`" + ` to park for human approval |

When ` + "`" + `permissions.merge` + "`" + ` is ` + "`" + `false` + "`" + `, the loop parks the completed worktree instead of auto-merging. The human reviews and approves parked worktrees before they are merged.

The global ` + "`" + `autonomy` + "`" + ` config (` + "`" + `auto` + "`" + ` or ` + "`" + `approve` + "`" + `) overrides per-skill merge permissions: ` + "`" + `approve` + "`" + ` mode parks all worktrees regardless of the skill's ` + "`" + `permissions.merge` + "`" + ` value.

## Config Reference

Noodle reads ` + "`" + `.noodle.toml` + "`" + ` at project root. If missing, ` + "`" + `noodle start` + "`" + ` scaffolds a minimal one on first run.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
{{- range .ConfigFields}}
| ` + "`" + `{{.TOMLKey}}` + "`" + ` | {{.Type}} | {{.Default}} | {{.Description}} |
{{- end}}

### Minimal config

` + "```" + `toml
autonomy = "auto"

[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"

[skills]
paths = [".agents/skills"]
` + "```" + `

For adapter config and routing tags, see [references/adapters.md](references/adapters.md) and [references/config-schema.md](references/config-schema.md).

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

1. **"tmux is not available on PATH"** — Install tmux.
2. **"fatal config diagnostics prevent start"** — Run ` + "`" + `noodle debug` + "`" + `, fix fields in ` + "`" + `.noodle.toml` + "`" + `.
3. **Missing adapter scripts** — Create scripts or update paths in config.
4. **Stale worktrees** — ` + "`" + `noodle worktree list` + "`" + `, then ` + "`" + `noodle worktree prune` + "`" + `.

## References

- [references/config-schema.md](references/config-schema.md) — routing tags, config validation
- [references/adapters.md](references/adapters.md) — adapter setup, script writing, provider examples
- [references/hooks.md](references/hooks.md) — brain injection hook, settings.json setup
`
