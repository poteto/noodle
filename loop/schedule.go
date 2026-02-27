package loop

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/internal/schemadoc"
)

const scheduleOrderID = "schedule"

func isScheduleStage(stage Stage) bool {
	return strings.EqualFold(strings.TrimSpace(stage.TaskKey), scheduleOrderID)
}

func isScheduleOrder(order Order) bool {
	return strings.EqualFold(strings.TrimSpace(order.ID), scheduleOrderID)
}

func hasNonScheduleOrders(orders OrdersFile) bool {
	for _, order := range orders.Orders {
		if !isScheduleOrder(order) {
			return true
		}
	}
	return false
}

func bootstrapScheduleOrder(cfg config.Config) OrdersFile {
	return OrdersFile{
		Orders: []Order{
			scheduleOrder(cfg, ""),
		},
	}
}

func scheduleOrder(cfg config.Config, prompt string) Order {
	order := Order{
		ID:     scheduleOrderID,
		Title:  "scheduling tasks based on your backlog",
		Status: OrderStatusActive,
		Stages: []Stage{
			{
				TaskKey:  scheduleOrderID,
				Skill:    "schedule",
				Provider: strings.TrimSpace(cfg.Routing.Defaults.Provider),
				Model:    strings.TrimSpace(cfg.Routing.Defaults.Model),
				Status:   StageStatusPending,
			},
		},
	}
	prompt = strings.TrimSpace(prompt)
	if prompt != "" {
		order.Rationale = "Chef steer: " + prompt
	}
	return order
}

func (l *Loop) spawnSchedule(ctx context.Context, order Order, attempt int, resumePrompt string) error {
	name := scheduleOrderID
	stage := order.Stages[0]

	skillName := nonEmpty(stage.Skill, "schedule")
	// Belt-and-suspenders: ensure the schedule skill is fresh before dispatch.
	if !l.ensureSkillFresh(skillName) {
		return l.spawnBootstrapIfNeeded(ctx, order)
	}

	taskTypesPrompt := buildOrderTaskTypesPrompt(l.registry.All())
	req := dispatcher.DispatchRequest{
		Name:                 name,
		Prompt:               buildSchedulePrompt(skillName, taskTypesPrompt, order, resumePrompt),
		Provider:             nonEmpty(stage.Provider, l.config.Routing.Defaults.Provider),
		Model:                nonEmpty(stage.Model, l.config.Routing.Defaults.Model),
		Skill:                skillName,
		WorktreePath:         l.projectDir,
		AllowPrimaryCheckout: true,
		Title:                order.Title,
		RetryCount:           attempt,
	}
	// Persist active status BEFORE spawning — crash safety.
	orders, err := readOrders(l.deps.OrdersFile)
	if err != nil {
		return err
	}
	for i := range orders.Orders {
		if orders.Orders[i].ID != order.ID {
			continue
		}
		if len(orders.Orders[i].Stages) > 0 {
			orders.Orders[i].Stages[0].Status = StageStatusActive
		}
		break
	}
	if err := writeOrdersAtomic(l.deps.OrdersFile, orders); err != nil {
		return err
	}

	session, err := l.deps.Dispatcher.Dispatch(ctx, req)
	if err != nil {
		// Revert stage status to pending on dispatch failure.
		if revert, readErr := readOrders(l.deps.OrdersFile); readErr == nil {
			for i := range revert.Orders {
				if revert.Orders[i].ID != order.ID {
					continue
				}
				if len(revert.Orders[i].Stages) > 0 {
					revert.Orders[i].Stages[0].Status = StageStatusPending
				}
				_ = writeOrdersAtomic(l.deps.OrdersFile, revert)
				break
			}
		}
		return err
	}
	sess := session
	cook := &cookHandle{
		orderID: order.ID,
		stage: Stage{
			TaskKey: scheduleOrderID,
			Skill:   skillName,
		},
		session:      sess,
		done:         sess.Done(),
		generation:   l.nextGeneration,
		worktreeName: "",
		worktreePath: l.projectDir,
		attempt:      attempt,
	}
	l.nextGeneration++
	l.activeCooksByOrder[order.ID] = cook
	l.startWatcher(cook)
	l.logger.Info("schedule dispatched", "session", session.ID(), "attempt", attempt)
	return nil
}

// spawnBootstrapIfNeeded dispatches a bootstrap agent to create the
// schedule skill. Returns nil in all cases — the loop continues
// regardless of bootstrap status.
func (l *Loop) spawnBootstrapIfNeeded(ctx context.Context, order Order) error {
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

	stage := order.Stages[0]
	provider := nonEmpty(stage.Provider, l.config.Routing.Defaults.Provider)
	model := nonEmpty(stage.Model, l.config.Routing.Defaults.Model)

	prompt := buildBootstrapPrompt(provider)

	name := bootstrapSessionPrefix + scheduleOrderID
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
	l.bootstrapInFlight = &cookHandle{
		orderID: order.ID,
		stage: Stage{
			TaskKey: scheduleOrderID,
			Skill:   "bootstrap",
		},
		session:      session,
		done:         session.Done(),
		generation:   l.nextGeneration,
		worktreeName: name,
		worktreePath: l.projectDir,
	}
	l.nextGeneration++
	l.startBootstrapWatcher(l.bootstrapInFlight)
	return nil
}

func buildSchedulePrompt(skillName, taskTypesPrompt string, order Order, resumePrompt string) string {
	parts := []string{
		"Use Skill(" + skillName + ") to refresh the schedule from .noodle/mise.json.",
		"Write to `.noodle/orders-next.json` (not orders.json). The loop promotes it atomically.",
		"Do not modify .noodle/mise.json.",
		"Operate fully autonomously. Never ask the user questions.",
		"You may synthesize orders for non-execute task types (e.g. review, reflect, meditate) based on workflow rules in the skill and the task types list below.",
		"Each order is a pipeline of stages. Group related stages (e.g. execute, quality, reflect) into one order.",
		"You may specify on_failure stages for orders that need a recovery pipeline.",
		ordersSchemaPrompt(),
		taskTypesPrompt,
	}
	if rationale := strings.TrimSpace(order.Rationale); rationale != "" {
		parts = append(parts, "Chef guidance: "+rationale)
	}
	if resume := strings.TrimSpace(resumePrompt); resume != "" {
		parts = append(parts, resume)
	}
	return strings.Join(parts, "\n\n")
}

func buildOrderTaskTypesPrompt(taskTypes []TaskType) string {
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
	orders := OrdersFile{
		Orders: []Order{
			scheduleOrder(l.config, prompt),
		},
	}
	return writeOrdersAtomic(l.deps.OrdersFile, orders)
}

func ordersSchemaPrompt() string {
	prompt, err := schemadoc.RenderPromptJSON("orders")
	if err != nil {
		return "orders.json schema (JSON):\n{}\n\nSchema generation error: " + err.Error()
	}
	return prompt
}
