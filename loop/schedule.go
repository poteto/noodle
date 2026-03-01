package loop

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/schemadoc"
	loopruntime "github.com/poteto/noodle/runtime"
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

func hasScheduleOrder(orders OrdersFile) bool {
	for _, order := range orders.Orders {
		if isScheduleOrder(order) {
			return true
		}
	}
	return false
}

func (l *Loop) hasActiveScheduleCook() bool {
	for _, cook := range l.cooks.activeCooksByOrder {
		if isScheduleStage(cook.stage) {
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
	l.schedulePromoted = false // reset: new schedule dispatch invalidates prior promotion
	name := scheduleOrderID
	stageIndex, stagePtr := activeStageForOrder(order)
	if stageIndex < 0 || stagePtr == nil {
		return fmt.Errorf("schedule order has no active or pending stage")
	}
	stage := *stagePtr

	skillName := nonEmpty(stage.Skill, "schedule")
	// Belt-and-suspenders: ensure the schedule skill is fresh before dispatch.
	if !l.ensureSkillFresh(skillName) {
		return l.spawnBootstrapIfNeeded(ctx, order)
	}

	taskTypesPrompt := buildOrderTaskTypesPrompt(l.registry.All())
	promotionError := l.lastPromotionError
	l.lastPromotionError = ""
	req := loopruntime.DispatchRequest{
		Name:                 name,
		Prompt:               buildSchedulePrompt(skillName, taskTypesPrompt, order, resumePrompt, l.runtimeDir, promotionError),
		Provider:             nonEmpty(stage.Provider, l.config.Routing.Defaults.Provider),
		Model:                nonEmpty(stage.Model, l.config.Routing.Defaults.Model),
		Skill:                skillName,
		WorktreePath:         l.projectDir,
		AllowPrimaryCheckout: true,
		Title:                order.Title,
		RetryCount:           attempt,
	}
	if err := l.persistOrderStageStatus(order.ID, stageIndex, StageStatusActive); err != nil {
		return err
	}
	session, err := l.dispatchSession(ctx, req)
	if err != nil {
		_ = l.persistOrderStageStatus(order.ID, stageIndex, StageStatusPending)
		return err
	}
	cook := &cookHandle{
		cookIdentity: cookIdentity{
			orderID:    order.ID,
			stageIndex: stageIndex,
			stage:      stage,
			plan:       order.Plan,
		},
		orderStatus:  order.Status,
		session:      session,
		worktreeName: "",
		worktreePath: l.projectDir,
		attempt:      attempt,
		generation:   l.nextDispatchGeneration(),
	}
	l.trackCookStarted(cook)
	l.cooks.activeCooksByOrder[order.ID] = cook
	l.startSessionWatcher(ctx, cook, false)

	// Emit V2 canonical state events for schedule dispatch.
	l.emitEvent(ingest.EventDispatchRequested, map[string]any{
		"order_id":    order.ID,
		"stage_index": stageIndex,
	})
	l.emitEvent(ingest.EventDispatchCompleted, map[string]any{
		"order_id":    order.ID,
		"stage_index": stageIndex,
		"session_id":  session.ID(),
	})

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
		_ = l.events.Emit(LoopEventBootstrapExhausted, BootstrapExhaustedPayload{
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
	req := loopruntime.DispatchRequest{
		Name:                 name,
		Prompt:               "Create a schedule skill for this project. Follow the system prompt instructions exactly.",
		Provider:             provider,
		Model:                model,
		SystemPrompt:         prompt,
		WorktreePath:         l.projectDir,
		AllowPrimaryCheckout: true,
		Title:                "bootstrapping schedule skill",
	}
	session, err := l.dispatchSession(ctx, req)
	if err != nil {
		l.logger.Warn("bootstrap dispatch failed", "error", err, "attempt", l.bootstrapAttempts+1)
		l.bootstrapAttempts++
		if l.bootstrapAttempts >= 3 {
			l.bootstrapExhausted = true
		}
		return nil
	}
	l.bootstrapInFlight = &cookHandle{
		cookIdentity: cookIdentity{
			orderID: order.ID,
			stage: Stage{
				TaskKey: scheduleOrderID,
				Skill:   "bootstrap",
			},
		},
		session:      session,
		worktreeName: name,
		worktreePath: l.projectDir,
		generation:   l.nextDispatchGeneration(),
	}
	l.trackCookStarted(l.bootstrapInFlight)
	l.startSessionWatcher(ctx, l.bootstrapInFlight, true)
	return nil
}

func buildSchedulePrompt(skillName, taskTypesPrompt string, order Order, resumePrompt string, runtimeDir string, lastPromotionError string) string {
	miseFile := filepath.Join(runtimeDir, "mise.json")
	ordersNextFile := filepath.Join(runtimeDir, "orders-next.json")
	parts := []string{
		"Use Skill(" + skillName + ") to refresh the schedule from " + miseFile + ".",
		"Write to `" + ordersNextFile + "` (not orders.json). The loop promotes it atomically.",
		"Do not modify " + miseFile + ".",
		"Operate fully autonomously. Never ask the user questions.",
		"You may synthesize orders for non-execute task types (e.g. review, reflect, meditate) based on workflow rules in the skill and the task types list below.",
		"Each order is a pipeline of stages. Group related stages (e.g. execute, quality, reflect) into one order.",
		"Failed stages are removed from orders and forwarded to the scheduler. Use control commands (advance, add-stage, park-review) to manage recovery.",
		ordersSchemaPrompt(),
		taskTypesPrompt,
	}
	if errMsg := strings.TrimSpace(lastPromotionError); errMsg != "" {
		parts = append(parts, "PREVIOUS ORDERS REJECTED: Your last orders-next.json was invalid and renamed to .bad. Fix the following error in your next output:\n"+errMsg)
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
	return l.writeOrdersState(orders)
}

func ordersSchemaPrompt() string {
	prompt, err := schemadoc.RenderPromptJSON("orders")
	if err != nil {
		return "orders.json schema (JSON):\n{}\n\nSchema generation error: " + err.Error()
	}
	return prompt
}
