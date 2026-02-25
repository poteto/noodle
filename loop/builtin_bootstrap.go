package loop

import (
	"strings"
)

const bootstrapSessionPrefix = "bootstrap-"

const bootstrapPromptTemplate = `# Bootstrap: Create Schedule Skill

You are a bootstrap agent. Your only job is to create a schedule skill for this Noodle project, then exit.

## Steps

1. Check if the file ` + "`" + `.agents/skills/schedule/SKILL.md` + "`" + ` already exists. If it does, exit immediately — nothing to do.

2. Inspect the project to understand its shape:
   - Read ` + "`" + `brain/todos.md` + "`" + ` if it exists (backlog items)
   - Read files in ` + "`" + `brain/plans/` + "`" + ` if the directory exists (implementation plans)
   - Read conversation history from {{history_dirs}} to understand what the project does
   - Read ` + "`" + `.noodle.toml` + "`" + ` for project configuration
   - Skim the top-level directory structure

3. Create ` + "`" + `.agents/skills/schedule/SKILL.md` + "`" + ` with appropriate content. Use the example below as a starting template, but customize the scheduling rules and scheduling criteria to match this specific project's backlog shape, plan structure, and conventions you discovered.

4. Run ` + "`" + `git add .agents/skills/schedule/SKILL.md` + "`" + ` and commit with message ` + "`" + `feat: bootstrap schedule skill` + "`" + `.

5. Exit immediately. Do NOT perform scheduling — that happens in the next cycle.

## Example Schedule Skill

` + "```" + `markdown
---
name: schedule
description: Queue scheduler. Reads .noodle/mise.json, writes .noodle/queue-next.json.
noodle:
  schedule: "When the queue is empty, after backlog changes, or when session history suggests re-evaluation"
---

# Schedule

Read ` + "`.noodle/mise.json`" + `, write ` + "`.noodle/queue-next.json`" + `.
The loop atomically promotes queue-next.json to queue.json — never write queue.json directly.
Use ` + "`noodle schema mise`" + ` and ` + "`noodle schema queue`" + ` as the schema source of truth.

Operate fully autonomously. Never ask the user to choose or pause for confirmation.

## Task Types

Read task_types from mise to discover every schedulable task type and its schedule hint.
Use task_key on each queue item to bind it to a task type.

### Execute Tasks

Only schedule execute tasks for plan IDs listed in needs_scheduling.
Use the plan ID (as a string) as the queue item id.

### Synthesizing Follow-Up Tasks

After scheduling a task, consider what naturally follows:
- Quality after Execute — review the cook's work
- Reflect after Quality — capture learnings
- Meditate after several Reflects — audit the brain vault
` + "```" + `

## Constraints

- Do NOT modify any existing files except to create the new skill
- Do NOT run ` + "`" + `noodle` + "`" + ` commands
- Do NOT perform scheduling — just create the skill file
- Keep the skill concise but complete enough to schedule work effectively
`

func buildBootstrapPrompt(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))

	var historyDirs string
	switch provider {
	case "codex":
		historyDirs = "`.codex/` (and `.claude/` if it exists)"
	case "claude":
		historyDirs = "`.claude/` (and `.codex/` if it exists)"
	default:
		historyDirs = "`.claude/` and/or `.codex/` if they exist"
	}

	return strings.ReplaceAll(bootstrapPromptTemplate, "{{history_dirs}}", historyDirs)
}

func isBootstrapSession(name string) bool {
	return strings.HasPrefix(name, bootstrapSessionPrefix)
}
