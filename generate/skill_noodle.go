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
	"prioritize.skill":              "Skill name loaded for scheduling sessions",
	"prioritize.run":                "When to run scheduling: after-each, after-n, or manual",
	"prioritize.model":              "Model used for scheduling sessions",
	"adapters":                      "Adapter configs keyed by adapter name (e.g. backlog)",
	"recovery.max_retries":          "Maximum retry attempts for failed cooks",
	"recovery.retry_suffix_pattern": "Naming pattern for retry sessions (must include %d)",
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
	"phases":                        "Map of phase names to skill names for lifecycle hooks",
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

Noodle is an open-source AI coding framework. Skills are the only extension point. An LLM schedules work, Go code executes it mechanically. Everything is a file — queue.json, mise.json, verdicts, control.ndjson. No hidden state.

Kitchen brigade model: the human is the Chef (strategy and judgment), Prioritize is the Sous Chef (scheduling), Cook does the work, Quality reviews it.

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

### Adapter config

Adapters bridge your backlog or plan system to Noodle. Each adapter has a skill (teaches agents the semantics) and scripts (deterministic commands for CRUD actions).

` + "```" + `toml
[adapters.backlog]
skill = "backlog"

[adapters.backlog.scripts]
sync = ".noodle/adapters/backlog-sync"
add = ".noodle/adapters/backlog-add"
done = ".noodle/adapters/backlog-done"
edit = ".noodle/adapters/backlog-edit"
` + "```" + `

Scripts can be any executable — shell scripts, binaries, or inline commands like ` + "`" + `gh issue close` + "`" + `. Noodle calls them mechanically; the skill teaches agents when and why to use them.

### Routing tags

Override the default model for specific task categories:

` + "```" + `toml
[routing.tags.frontend]
provider = "claude"
model = "claude-opus-4-6"

[routing.tags.backend]
provider = "codex"
model = "gpt-5.3-codex"
` + "```" + `

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

## Adapter Setup

Adapters are optional. If you omit ` + "`" + `[adapters.plans]` + "`" + ` from config, Noodle runs with backlog only. If both adapters are omitted, the mise contains only internal state.

### Writing adapter scripts

Each adapter action (sync, add, done, edit) maps to a script path in config. Scripts receive arguments via environment variables and must produce NDJSON output for sync actions.

1. **Sync** — reads all items from your system, writes NDJSON to stdout. Each line is a ` + "`" + `BacklogItem` + "`" + ` or ` + "`" + `PlanItem` + "`" + `.
2. **Add** — creates a new item. Receives ` + "`" + `NOODLE_TITLE` + "`" + ` and ` + "`" + `NOODLE_BODY` + "`" + ` env vars.
3. **Done** — marks an item complete. Receives ` + "`" + `NOODLE_ID` + "`" + `.
4. **Edit** — updates an item. Receives ` + "`" + `NOODLE_ID` + "`" + `, ` + "`" + `NOODLE_FIELD` + "`" + `, ` + "`" + `NOODLE_VALUE` + "`" + `.

### Markdown backlog (default)

The default adapter reads ` + "`" + `brain/todos.md` + "`" + ` — a markdown file with numbered items. Scripts live at ` + "`" + `.noodle/adapters/backlog-*` + "`" + `.

### GitHub Issues

` + "```" + `toml
[adapters.backlog]
skill = "backlog"

[adapters.backlog.scripts]
sync = "gh issue list --json number,title,body,labels,state"
add = "gh issue create"
done = "gh issue close"
edit = "gh issue edit"
` + "```" + `

### Linear

Use the Linear CLI or API. The adapter pattern is the same — write scripts that call the Linear API and output NDJSON.

## Hook Installation

### Brain injection hook

Injects brain vault content into the agent's context at session start. Add to ` + "`" + `.claude/settings.json` + "`" + `:

` + "```" + `json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "matcher": "SessionStart",
        "hooks": [
          {
            "type": "command",
            "command": "noodle worktree hook"
          }
        ]
      }
    ]
  }
}
` + "```" + `

## Skill Management

Skills live in ` + "`" + `.agents/skills/` + "`" + ` by default. Each skill is a directory with a ` + "`" + `SKILL.md` + "`" + ` file and optional ` + "`" + `references/` + "`" + ` subdirectory.

### Search path precedence

Paths in ` + "`" + `skills.paths` + "`" + ` are searched in order. First match wins. Project skills override user-level skills of the same name.

### Installing a skill

Copy the skill directory to your first skill path:

` + "```" + `sh
cp -r /path/to/skill .agents/skills/my-skill
` + "```" + `

### Task-type skills

Skills with a ` + "`" + `task_type` + "`" + ` in their frontmatter are discovered as task types by the scheduling loop. These are loaded automatically for their respective session types (prioritize, review, etc.).

## Troubleshooting

### ` + "`" + `noodle debug` + "`" + `

Dumps the full runtime state: config validation, active sessions, queue, mise, and diagnostics. Run this first when something is wrong.

### Common issues

1. **"tmux is not available on PATH"** — Install tmux. Noodle uses tmux to spawn and manage cook sessions.
2. **"fatal config diagnostics prevent start"** — Run ` + "`" + `noodle debug` + "`" + ` to see which config fields are invalid. Fix them in ` + "`" + `.noodle.toml` + "`" + `.
3. **Missing adapter scripts** — ` + "`" + `noodle start` + "`" + ` reports missing script paths as repairable diagnostics. Create the scripts or update the paths in config.
4. **Stale worktrees** — Run ` + "`" + `noodle worktree list` + "`" + ` to check status, then ` + "`" + `noodle worktree prune` + "`" + ` to clean up merged branches.

### Config validation

` + "`" + `noodle start` + "`" + ` validates config on every run. Diagnostics are classified as:
- **Fatal** — blocks startup (missing tmux, invalid routing defaults)
- **Repairable** — warns but allows startup (missing adapter scripts)

On interactive terminals, ` + "`" + `noodle start` + "`" + ` offers to spawn a repair session for repairable issues.
`
