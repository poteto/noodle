package loop

import (
	"context"
	"strings"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/internal/schemadoc"
	"github.com/poteto/noodle/recover"
)

const prioritizeQueueID = "prioritize"

func isPrioritizeItem(item QueueItem) bool {
	return strings.EqualFold(strings.TrimSpace(item.ID), prioritizeQueueID)
}

func bootstrapPrioritizeQueue(cfg config.Config, prompt string, generatedAt time.Time) Queue {
	return Queue{
		GeneratedAt: generatedAt,
		Items:       []QueueItem{prioritizeQueueItem(cfg, prompt)},
	}
}

func prioritizeQueueItem(cfg config.Config, prompt string) QueueItem {
	item := QueueItem{
		ID:       prioritizeQueueID,
		Title:    "prioritizing tasks based on your backlog",
		Provider: strings.TrimSpace(cfg.Routing.Defaults.Provider),
		Model:    strings.TrimSpace(cfg.Routing.Defaults.Model),
		Skill:    "prioritize",
	}
	prompt = strings.TrimSpace(prompt)
	if prompt != "" {
		item.Rationale = "Chef steer: " + prompt
	}
	return item
}

func (l *Loop) spawnPrioritize(ctx context.Context, item QueueItem, attempt int, resumePrompt string) error {
	name := prioritizeQueueID
	if attempt > 0 {
		nextName, err := recover.NextRecoveryName(name, attempt, l.config.Recovery.RetrySuffixPattern)
		if err != nil {
			return err
		}
		name = nextName
	}

	skillName := nonEmpty(item.Skill, "prioritize")
	taskTypesPrompt := buildQueueTaskTypesPrompt(l.registry.All())
	req := dispatcher.DispatchRequest{
		Name:                 name,
		Prompt:               buildPrioritizePrompt(skillName, taskTypesPrompt, item, resumePrompt),
		Provider:             nonEmpty(item.Provider, l.config.Routing.Defaults.Provider),
		Model:                nonEmpty(item.Model, l.config.Routing.Defaults.Model),
		Skill:                skillName,
		WorktreePath:         l.projectDir,
		AllowPrimaryCheckout: true,
	}
	session, err := l.deps.Dispatcher.Dispatch(ctx, req)
	if err != nil {
		return err
	}
	cook := &activeCook{
		queueItem:    item,
		session:      session,
		worktreeName: "",
		worktreePath: l.projectDir,
		attempt:      attempt,
	}
	l.activeByTarget[item.ID] = cook
	l.activeByID[session.ID()] = cook
	return nil
}

func buildPrioritizePrompt(skillName, taskTypesPrompt string, item QueueItem, resumePrompt string) string {
	parts := []string{
		"Use Skill(" + skillName + ") to refresh the queue from .noodle/mise.json.",
		"Write to `.noodle/queue-next.json` (not queue.json). The loop promotes it atomically.",
		"Do not modify .noodle/mise.json.",
		"Operate fully autonomously. Never ask the user questions.",
		"You may synthesize queue items for non-execute task types (e.g. review, reflect, meditate) based on workflow rules in the skill and the task types list below.",
		queueSchemaPrompt(),
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
	b.WriteString("Task types you may schedule:")
	if len(taskTypes) == 0 {
		b.WriteString("\n- (none configured)")
		return b.String()
	}
	for _, taskType := range taskTypes {
		key := strings.TrimSpace(taskType.Key)
		if key == "" {
			continue
		}
		schedule := strings.TrimSpace(taskType.Schedule)
		if schedule == "" {
			schedule = key
		}
		b.WriteString("\n- " + key + ": " + schedule)
	}
	return b.String()
}

func (l *Loop) reprioritizeForChefPrompt(prompt string) error {
	queue := bootstrapPrioritizeQueue(l.config, prompt, l.deps.Now().UTC())
	return writeQueueAtomic(l.deps.QueueFile, queue)
}

func queueSchemaPrompt() string {
	prompt, err := schemadoc.RenderPromptJSON("queue")
	if err != nil {
		return "queue.json schema (JSON):\n{}\n\nSchema generation error: " + err.Error()
	}
	return prompt
}
