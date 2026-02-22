package loop

import (
	"context"
	"strings"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/recover"
	"github.com/poteto/noodle/spawner"
)

const sousChefQueueID = "sous-chef"

const queueSchemaPrompt = `queue.json schema (JSON):
{
  "generated_at": "RFC3339 timestamp",
  "items": [
    {
      "id": "string",
      "title": "string (optional)",
      "provider": "string",
      "model": "string",
      "skill": "string (optional)",
      "review": "boolean (optional)",
      "rationale": "string (optional)"
    }
  ]
}`

const miseSchemaPrompt = `mise.json schema (JSON):
{
  "generated_at": "RFC3339 timestamp",
  "backlog": [
    {
      "id": "string",
      "title": "string",
      "description": "string (optional)",
      "section": "string (optional)",
      "status": "open|in_progress|done",
      "tags": ["string"],
      "plan": "string (optional)",
      "plan_phase": "string (optional)",
      "estimate": "small|medium|large (optional)"
    }
  ],
  "plans": [
    {
      "id": "string",
      "title": "string",
      "status": "draft|active|done",
      "phases": [{"name": "string", "status": "pending|active|done|skipped"}],
      "tags": ["string"]
    }
  ],
  "active_cooks": [
    {
      "id": "string",
      "target": "string (optional)",
      "provider": "string (optional)",
      "model": "string (optional)",
      "status": "string",
      "cost": "number",
      "duration_s": "integer"
    }
  ],
  "tickets": [
    {
      "target": "string",
      "target_type": "backlog_item|file|plan_phase",
      "cook_id": "string",
      "files": ["string"],
      "claimed_at": "RFC3339 timestamp",
      "last_progress": "RFC3339 timestamp",
      "status": "active|blocked|stale",
      "blocked_by": "string (optional)",
      "reason": "string (optional)"
    }
  ],
  "resources": {"max_cooks": "integer", "active": "integer", "available": "integer"},
  "recent_history": [
    {
      "target": "string (optional)",
      "cook_id": "string",
      "outcome": "string",
      "cost": "number",
      "duration_s": "integer"
    }
  ],
  "routing": {
    "defaults": {"provider": "string", "model": "string"},
    "tags": {"<tag>": {"provider": "string", "model": "string"}}
  },
  "warnings": ["string"]
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
	req := spawner.SpawnRequest{
		Name:                 name,
		Prompt:               buildSousChefPrompt(skillName, item, resumePrompt),
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

func buildSousChefPrompt(skillName string, item QueueItem, resumePrompt string) string {
	parts := []string{
		"Use Skill(" + skillName + ") to refresh .noodle/queue.json from .noodle/mise.json.",
		queueSchemaPrompt,
		miseSchemaPrompt,
	}
	if rationale := strings.TrimSpace(item.Rationale); rationale != "" {
		parts = append(parts, "Chef guidance: "+rationale)
	}
	if resume := strings.TrimSpace(resumePrompt); resume != "" {
		parts = append(parts, resume)
	}
	return strings.Join(parts, "\n\n")
}

func sousChefSkill(cfg config.Config) string {
	if skillName := strings.TrimSpace(cfg.SousChef.Skill); skillName != "" {
		return skillName
	}
	return "sous-chef"
}

func (l *Loop) reprioritizeForChefPrompt(prompt string) error {
	queue := bootstrapSousChefQueue(l.config, prompt, l.deps.Now().UTC())
	return writeQueueAtomic(l.deps.QueueFile, queue)
}
