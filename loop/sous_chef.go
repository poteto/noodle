package loop

import (
	"context"
	"strings"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/recover"
	"github.com/poteto/noodle/spawner"
)

const sousChefQueueID = taskKeySousChef

const queueSchemaPrompt = `queue.json schema (JSON):
{
  "generated_at": "RFC3339 timestamp",
  "items": [
    {
      "id": "string",
      "task_key": "string (optional)",
      "title": "string (optional)",
      "provider": "string",
      "model": "string",
      "skill": "string (optional)",
      "review": "boolean (optional)",
      "rationale": "string (optional)"
    }
  ]
}`

func isSousChefItem(item QueueItem) bool {
	return strings.EqualFold(strings.TrimSpace(item.ID), sousChefQueueID)
}

func bootstrapSousChefQueue(cfg config.Config, prompt string, generatedAt time.Time) Queue {
	return Queue{
		GeneratedAt: generatedAt,
		Items:       []QueueItem{sousChefQueueItem(cfg, prompt)},
	}
}

func sousChefQueueItem(cfg config.Config, prompt string) QueueItem {
	item := QueueItem{
		ID:       sousChefQueueID,
		Title:    "Refresh prioritized queue from mise",
		Provider: strings.TrimSpace(cfg.Routing.Defaults.Provider),
		Model:    strings.TrimSpace(cfg.Routing.Defaults.Model),
		Skill:    sousChefSkill(cfg),
	}
	prompt = strings.TrimSpace(prompt)
	if prompt != "" {
		item.Rationale = "Chef steer: " + prompt
	}
	return item
}

func (l *Loop) spawnSousChef(ctx context.Context, item QueueItem, attempt int, resumePrompt string) error {
	name := sousChefQueueID
	if attempt > 0 {
		nextName, err := recover.NextRecoveryName(name, attempt, l.config.Recovery.RetrySuffixPattern)
		if err != nil {
			return err
		}
		name = nextName
	}

	skillName := nonEmpty(item.Skill, sousChefSkill(l.config))
	taskTypesPrompt := buildQueueTaskTypesPrompt(configuredTaskTypes(l.config))
	req := spawner.SpawnRequest{
		Name:                 name,
		Prompt:               buildSousChefPrompt(skillName, taskTypesPrompt, item, resumePrompt),
		Provider:             nonEmpty(item.Provider, l.config.Routing.Defaults.Provider),
		Model:                nonEmpty(item.Model, l.config.Routing.Defaults.Model),
		Skill:                skillName,
		WorktreePath:         l.projectDir,
		AllowPrimaryCheckout: true,
	}
	session, err := l.deps.Spawner.Spawn(ctx, req)
	if err != nil {
		return err
	}
	cook := &activeCook{
		queueItem:     item,
		session:       session,
		worktreeName:  "",
		worktreePath:  l.projectDir,
		attempt:       attempt,
		reviewEnabled: false,
	}
	l.activeByTarget[item.ID] = cook
	l.activeByID[session.ID()] = cook
	return nil
}

func buildSousChefPrompt(skillName, taskTypesPrompt string, item QueueItem, resumePrompt string) string {
	parts := []string{
		"Use Skill(" + skillName + ") to refresh .noodle/queue.json from .noodle/mise.json.",
		"Do not modify .noodle/mise.json.",
		"Operate fully autonomously. Never ask the user questions.",
		"You may synthesize new queue items that are not present in mise.json when enforcing stage transitions (for example, Plan -> Review, Execute -> Verify, Verify -> Reflect).",
		queueSchemaPrompt,
		taskTypesPrompt,
	}
	if rationale := strings.TrimSpace(item.Rationale); rationale != "" {
		parts = append(parts, "Chef guidance: "+rationale)
	}
	if resume := strings.TrimSpace(resumePrompt); resume != "" {
		parts = append(parts, resume)
	}
	return strings.Join(parts, "\n\n")
}

func buildQueueTaskTypesPrompt(taskTypes []TaskType) string {
	var b strings.Builder
	b.WriteString("Task types you may schedule (from loop/task_types.go):")
	if len(taskTypes) == 0 {
		b.WriteString("\n- (none configured)")
		return b.String()
	}
	for _, taskType := range taskTypes {
		line := "- " + strings.TrimSpace(taskType.Type)
		line += " | key: " + strings.TrimSpace(taskType.Key)
		if cfg := strings.TrimSpace(taskType.ConfigPath); cfg != "" {
			line += " | config: " + cfg
		}
		if skillName := strings.TrimSpace(taskType.Skill); skillName != "" {
			line += " | skill: " + skillName
		}
		line += " | blocking: "
		if taskType.Blocking {
			line += "true"
		} else {
			line += "false"
		}
		line += " | synthetic: "
		if taskType.Synthetic {
			line += "true"
		} else {
			line += "false"
		}
		if len(taskType.Aliases) > 0 {
			line += " | aliases: "
			line += strings.Join(taskType.Aliases, ", ")
		}
		if purpose := strings.TrimSpace(taskType.Purpose); purpose != "" {
			line += " | purpose: " + purpose
		}
		b.WriteString("\n")
		b.WriteString(line)
	}
	return b.String()
}

func sousChefSkill(cfg config.Config) string {
	return sousChefTaskSkill(cfg)
}

func (l *Loop) reprioritizeForChefPrompt(prompt string) error {
	queue := bootstrapSousChefQueue(l.config, prompt, l.deps.Now().UTC())
	return writeQueueAtomic(l.deps.QueueFile, queue)
}
