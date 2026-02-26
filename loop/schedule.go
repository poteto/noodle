package loop

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/internal/schemadoc"
)

const scheduleQueueID = "schedule"

func isScheduleItem(item QueueItem) bool {
	return strings.EqualFold(strings.TrimSpace(item.ID), scheduleQueueID)
}

func isScheduleStage(stage Stage) bool {
	return strings.EqualFold(strings.TrimSpace(stage.TaskKey), scheduleQueueID)
}

func isScheduleOrder(order Order) bool {
	if len(order.Stages) == 0 {
		return false
	}
	return isScheduleStage(order.Stages[0])
}

func hasNonScheduleItems(queue Queue) bool {
	for _, item := range queue.Items {
		if !isScheduleItem(item) {
			return true
		}
	}
	return false
}

func filterStaleScheduleItems(queue Queue) Queue {
	if len(queue.Items) == 0 {
		return queue
	}
	filtered := queue
	filtered.Items = make([]QueueItem, 0, len(queue.Items))
	for _, item := range queue.Items {
		if isScheduleItem(item) {
			continue
		}
		filtered.Items = append(filtered.Items, item)
	}
	return filtered
}

func bootstrapScheduleQueue(cfg config.Config, prompt string, generatedAt time.Time) Queue {
	return Queue{
		GeneratedAt: generatedAt,
		Items:       []QueueItem{scheduleQueueItem(cfg, prompt)},
	}
}

func bootstrapScheduleOrders(cfg config.Config, generatedAt time.Time) OrdersFile {
	return OrdersFile{
		GeneratedAt: generatedAt,
		Orders: []Order{
			{
				ID:     scheduleQueueID,
				Title:  "scheduling tasks based on your backlog",
				Status: OrderStatusActive,
				Stages: []Stage{
					{
						TaskKey:  scheduleQueueID,
						Skill:    "schedule",
						Provider: strings.TrimSpace(cfg.Routing.Defaults.Provider),
						Model:    strings.TrimSpace(cfg.Routing.Defaults.Model),
						Status:   StageStatusPending,
					},
				},
			},
		},
	}
}

func scheduleQueueItem(cfg config.Config, prompt string) QueueItem {
	item := QueueItem{
		ID:       scheduleQueueID,
		Title:    "scheduling tasks based on your backlog",
		Provider: strings.TrimSpace(cfg.Routing.Defaults.Provider),
		Model:    strings.TrimSpace(cfg.Routing.Defaults.Model),
		Skill:    "schedule",
	}
	prompt = strings.TrimSpace(prompt)
	if prompt != "" {
		item.Rationale = "Chef steer: " + prompt
	}
	return item
}

func (l *Loop) spawnSchedule(ctx context.Context, item QueueItem, attempt int, resumePrompt string) error {
	name := scheduleQueueID

	skillName := nonEmpty(item.Skill, "schedule")
	// Belt-and-suspenders: ensure the schedule skill is fresh before dispatch.
	if !l.ensureSkillFresh(skillName) {
		return l.spawnBootstrapIfNeeded(ctx, item)
	}

	taskTypesPrompt := buildQueueTaskTypesPrompt(l.registry.All())
	req := dispatcher.DispatchRequest{
		Name:                 name,
		Prompt:               buildSchedulePrompt(skillName, taskTypesPrompt, item, resumePrompt),
		Provider:             nonEmpty(item.Provider, l.config.Routing.Defaults.Provider),
		Model:                nonEmpty(item.Model, l.config.Routing.Defaults.Model),
		Skill:                skillName,
		WorktreePath:         l.projectDir,
		AllowPrimaryCheckout: true,
		Title:                item.Title,
		RetryCount:           attempt,
	}
	session, err := l.deps.Dispatcher.Dispatch(ctx, req)
	if err != nil {
		return err
	}
	cook := &activeCook{
		orderID: item.ID,
		stage: Stage{
			TaskKey: scheduleQueueID,
			Skill:   skillName,
		},
		session:      session,
		worktreeName: "",
		worktreePath: l.projectDir,
		attempt:      attempt,
	}
	l.activeByTarget[item.ID] = cook
	l.activeByID[session.ID()] = cook
	l.logger.Info("schedule dispatched", "session", session.ID(), "attempt", attempt)
	return nil
}

// spawnBootstrapIfNeeded dispatches a bootstrap agent to create the
// schedule skill. Returns nil in all cases — the loop continues
// regardless of bootstrap status.
func (l *Loop) spawnBootstrapIfNeeded(ctx context.Context, item QueueItem) error {
	if l.bootstrapExhausted {
		l.logger.Warn("bootstrap exhausted — create .agents/skills/schedule/SKILL.md manually or check bootstrap skill output for errors",
			"attempts", l.bootstrapAttempts)
		eventsPath := filepath.Join(l.runtimeDir, "queue-events.ndjson")
		appendQueueEvent(eventsPath, QueueAuditEvent{
			At:     l.deps.Now().UTC(),
			Type:   "bootstrap_exhausted",
			Reason: fmt.Sprintf("bootstrap exhausted after %d attempts — create .agents/skills/schedule/SKILL.md manually or check bootstrap skill output for errors", l.bootstrapAttempts),
		})
		return nil
	}
	if l.bootstrapInFlight != nil {
		return nil
	}

	provider := nonEmpty(item.Provider, l.config.Routing.Defaults.Provider)
	model := nonEmpty(item.Model, l.config.Routing.Defaults.Model)

	prompt, err := buildBootstrapPrompt(provider, l.config.Skills.Paths)
	if err != nil {
		l.logger.Error("bootstrap skill resolution failed", "error", err)
		return nil
	}

	name := bootstrapSessionPrefix + scheduleQueueID
	req := dispatcher.DispatchRequest{
		Name:                 name,
		Prompt:               "Create a schedule skill for this project. Follow the system prompt instructions exactly.",
		Provider:             provider,
		Model:                model,
		SystemPrompt:         prompt,
		WorktreePath:         l.projectDir,
		AllowPrimaryCheckout: true,
		Title:                "bootstrapping schedule skill",
	}
	session, err := l.deps.Dispatcher.Dispatch(ctx, req)
	if err != nil {
		l.logger.Warn("bootstrap dispatch failed", "error", err, "attempt", l.bootstrapAttempts+1)
		l.bootstrapAttempts++
		if l.bootstrapAttempts >= 3 {
			l.bootstrapExhausted = true
		}
		return nil
	}
	l.bootstrapInFlight = &activeCook{
		orderID: item.ID,
		stage: Stage{
			TaskKey: scheduleQueueID,
			Skill:   "bootstrap",
		},
		session:      session,
		worktreeName: name,
		worktreePath: l.projectDir,
	}
	return nil
}

func buildSchedulePrompt(skillName, taskTypesPrompt string, item QueueItem, resumePrompt string) string {
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

func (l *Loop) rescheduleForChefPrompt(prompt string) error {
	queue := bootstrapScheduleQueue(l.config, prompt, l.deps.Now().UTC())
	return writeQueueAtomic(l.deps.QueueFile, queue)
}

func queueSchemaPrompt() string {
	prompt, err := schemadoc.RenderPromptJSON("queue")
	if err != nil {
		return "queue.json schema (JSON):\n{}\n\nSchema generation error: " + err.Error()
	}
	return prompt
}
