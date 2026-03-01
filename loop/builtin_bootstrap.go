package loop

import (
	"strings"

	"github.com/poteto/noodle/internal/stringx"
)

const bootstrapSessionPrefix = "bootstrap-"

const bootstrapPrompt = `# Bootstrap: Create Schedule Skill

You are a bootstrap agent. Your only job is to create a schedule skill for this Noodle project, then exit.

Use Skill(skill-creator) to help write the skill file.

## Steps

1. Check if the file ` + "`.agents/skills/schedule/SKILL.md`" + ` already exists. If it does, exit immediately — nothing to do.

2. Inspect the project to understand its shape:
   - Read ` + "`brain/todos.md`" + ` if it exists (backlog items)
   - Read conversation history from {{history_dirs}} to understand what the project does
   - Read ` + "`.noodle.toml`" + ` for project configuration
   - Skim the top-level directory structure

3. Create ` + "`.agents/skills/schedule/SKILL.md`" + ` with appropriate content. Use the example below as a starting template, but customize the scheduling rules and scheduling criteria to match this specific project's backlog shape and conventions you discovered.

4. Run ` + "`git add .agents/skills/schedule/SKILL.md`" + ` and commit with message ` + "`feat: bootstrap schedule skill`" + `.

5. Exit immediately. Do NOT perform scheduling — that happens in the next cycle.

## Example Schedule Skill

` + "```" + `markdown
---
name: schedule
description: Work order scheduler. Reads .noodle/mise.json, writes .noodle/orders-next.json.
schedule: "When no active orders exist, after backlog changes, or when session history suggests re-evaluation"
---

# Schedule

Read ` + "`.noodle/mise.json`" + `, write ` + "`.noodle/orders-next.json`" + `.
The loop atomically promotes orders-next.json into orders.json — never write orders.json directly.
Use ` + "`noodle schema mise`" + ` and ` + "`noodle schema orders`" + ` as the schema source of truth.

Operate fully autonomously. Never ask the user to choose or pause for confirmation.

## Task Types

Read task_types from mise to discover every schedulable task type and its schedule hint.
Each order contains sequential stages — use task_key on each stage to bind it to a task type.

### Orders

An order groups related stages into a pipeline. Execute → quality → reflect should be stages within ONE order, not separate orders.
Use the plan ID (as a string) as the order id.

### Synthesizing Follow-Up Stages

After scheduling an execute stage, add follow-up stages to the same order:
- Quality after Execute — review the cook's work
- Reflect after Quality — capture learnings
` + "```" + `

## Constraints

- Do NOT modify any existing files except to create the new skill
- Do NOT run ` + "`noodle`" + ` commands
- Do NOT perform scheduling — just create the skill file
- Keep the skill concise but complete enough to schedule work effectively`

// buildBootstrapPrompt returns the bootstrap prompt with {{history_dirs}}
// substituted based on provider.
func buildBootstrapPrompt(provider string) string {
	provider = stringx.Normalize(provider)
	var historyDirs string
	switch provider {
	case "codex":
		historyDirs = "`.codex/` (and `.claude/` if it exists)"
	case "claude":
		historyDirs = "`.claude/` (and `.codex/` if it exists)"
	default:
		historyDirs = "`.claude/` and/or `.codex/` if they exist"
	}

	return strings.ReplaceAll(bootstrapPrompt, "{{history_dirs}}", historyDirs)
}

func isBootstrapSession(name string) bool {
	return strings.HasPrefix(name, bootstrapSessionPrefix)
}
