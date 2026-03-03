package loop

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/schemadoc"
	loopruntime "github.com/poteto/noodle/runtime"
)

const scheduleOrderID = "schedule"
const scheduleBootstrapOrderID = "oops-bootstrap-schedule"
const scheduleSkillExamplePath = "https://github.com/poteto/noodle/tree/main/.agents/skills/schedule/"
const scheduleSkillExampleHint = "github.com/poteto/noodle/.agents/skills/schedule/"

func isScheduleStage(stage Stage) bool {
	return strings.EqualFold(strings.TrimSpace(stage.TaskKey), scheduleOrderID)
}

func isScheduleOrder(order Order) bool {
	return strings.EqualFold(strings.TrimSpace(order.ID), scheduleOrderID)
}

func isScheduleBootstrapOrder(order Order) bool {
	return strings.EqualFold(strings.TrimSpace(order.ID), scheduleBootstrapOrderID)
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

func hasScheduleBootstrapOrder(orders OrdersFile) bool {
	for _, order := range orders.Orders {
		if isScheduleBootstrapOrder(order) {
			return true
		}
	}
	return false
}

func (l *Loop) hasTaskType(taskKey string) bool {
	_, ok := l.registry.ByKey(strings.TrimSpace(taskKey))
	return ok
}

func prependOrder(orders *OrdersFile, order Order) {
	orders.Orders = append([]Order{order}, orders.Orders...)
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

func scheduleBootstrapOopsOrder(cfg config.Config) Order {
	return Order{
		ID:        scheduleBootstrapOrderID,
		Title:     "bootstrapping missing schedule skill",
		Rationale: "Schedule task type is unavailable; bootstrap scheduler skill first.",
		Status:    OrderStatusActive,
		Stages: []Stage{
			{
				TaskKey:  "oops",
				Skill:    "oops",
				Provider: strings.TrimSpace(cfg.Routing.Defaults.Provider),
				Model:    strings.TrimSpace(cfg.Routing.Defaults.Model),
				Status:   StageStatusPending,
				Prompt: strings.Join([]string{
					"Schedule skill is missing. Bootstrap it now so Noodle can recover scheduler dispatch.",
					"Create `.agents/skills/schedule/SKILL.md` with proper frontmatter including `schedule:`.",
					"Use `" + scheduleSkillExamplePath + "` as the primary example (also referenced as `" + scheduleSkillExampleHint + "`).",
					"Adapt the scheduling rules to this project's backlog and workflow.",
					"Commit the skill creation in your worktree.",
				}, "\n"),
			},
		},
	}
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
	failures := l.reconciledFailures
	req := loopruntime.DispatchRequest{
		Name:                 name,
		Prompt:               buildSchedulePrompt(skillName, taskTypesPrompt, order, resumePrompt, l.runtimeDir, promotionError, failures),
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
	session, _, err := l.dispatchSession(ctx, req)
	if err != nil {
		return l.handleCookDispatchFailure(dispatchCandidate{
			OrderID:    order.ID,
			StageIndex: stageIndex,
			Stage:      stage,
		}, stage, "", false, err)
	}
	l.reconciledFailures = nil // clear only after successful dispatch
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
		failureMetadata := eventFailureMetadataForLoop(CycleFailureClassDegradeContinue, "", nil)
		l.logger.Warn("bootstrap exhausted — create .agents/skills/schedule/SKILL.md manually or check bootstrap skill output for errors",
			"attempts", l.bootstrapAttempts)
		_ = l.events.Emit(LoopEventBootstrapExhausted, BootstrapExhaustedPayload{
			Reason:  fmt.Sprintf("bootstrap exhausted after %d attempts — create .agents/skills/schedule/SKILL.md manually or check bootstrap skill output for errors", l.bootstrapAttempts),
			Failure: &failureMetadata,
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
	session, _, err := l.dispatchSession(ctx, req)
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

func buildSchedulePrompt(skillName, taskTypesPrompt string, order Order, resumePrompt string, runtimeDir string, lastPromotionError string, failures []reconciledFailure) string {
	miseFile := filepath.Join(runtimeDir, "mise.json")
	ordersNextFile := filepath.Join(runtimeDir, "orders-next.json")
	parts := []string{
		"Use Skill(" + skillName + ") to refresh the schedule from " + miseFile + ".",
		"Write to `" + ordersNextFile + "` (not orders.json). The loop promotes it atomically.",
		"Do not modify " + miseFile + ".",
		"Operate fully autonomously. Only ask the user a question when backlog is empty and no actionable work exists; ask whether to schedule an order that creates a backlog adapter.",
		"You may synthesize orders for task types that don't require backlog items, based on workflow rules in the skill and the task types list below.",
		"Each order is a pipeline of stages. Group related stages into one order.",
		"Failed orders are archived on startup and their details are included below (if any). Use control commands (advance, add-stage, park-review) to manage recovery.",
		ordersSchemaPrompt(),
		taskTypesPrompt,
	}
	if errMsg := strings.TrimSpace(lastPromotionError); errMsg != "" {
		parts = append(parts, "PREVIOUS ORDERS ISSUE: The loop rejected or repaired your recent schedule output. Fix the following issue in your next orders-next.json output:\n"+errMsg)
	}
	if rationale := strings.TrimSpace(order.Rationale); rationale != "" {
		parts = append(parts, "Chef guidance: "+rationale)
	}
	if len(failures) > 0 {
		var fb strings.Builder
		fb.WriteString("Orders failed in a previous session and were archived on startup. Decide whether to re-queue from backlog:")
		for _, f := range failures {
			fb.WriteString("\n- order " + fmt.Sprintf("%q", f.OrderID))
			if f.Title != "" {
				fb.WriteString(" (" + f.Title + ")")
			}
			if f.TaskKey != "" {
				fb.WriteString(": stage " + f.TaskKey + " failed")
			}
			if f.Reason != "" {
				fb.WriteString(" — " + fmt.Sprintf("%q", f.Reason))
			}
		}
		parts = append(parts, fb.String())
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
	next := OrdersFile{
		Orders: []Order{
			scheduleOrder(l.config, prompt),
		},
	}
	return l.mutateOrdersState(func(orders *OrdersFile) (bool, error) {
		if reflect.DeepEqual(*orders, next) {
			return false, nil
		}
		*orders = next
		return true, nil
	})
}

func ordersSchemaPrompt() string {
	prompt, err := schemadoc.RenderPromptJSON("orders")
	if err != nil {
		return "orders.json schema (JSON):\n{}\n\nSchema generation error: " + err.Error()
	}
	return prompt
}
