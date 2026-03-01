package loop

import (
	"errors"
	"fmt"
	"strings"

	"github.com/poteto/noodle/internal/failure"
)

// CycleFailureClass classifies loop-cycle failure behavior.
type CycleFailureClass string

const (
	CycleFailureClassSystemHard      CycleFailureClass = "system-hard"
	CycleFailureClassOrderHard       CycleFailureClass = "order-hard"
	CycleFailureClassDegradeContinue CycleFailureClass = "degrade-continue"
)

// OrderFailureClass classifies terminal order-level outcomes.
type OrderFailureClass string

const (
	OrderFailureClassStageTerminal OrderFailureClass = "stage-terminal"
	OrderFailureClassOrderTerminal OrderFailureClass = "order-terminal"
)

var recoverabilityByCycleFailureClass = map[CycleFailureClass]failure.FailureRecoverability{
	CycleFailureClassSystemHard:      failure.FailureRecoverabilityHard,
	CycleFailureClassOrderHard:       failure.FailureRecoverabilityRecoverable,
	CycleFailureClassDegradeContinue: failure.FailureRecoverabilityDegrade,
}

// LoopFailureEnvelope wraps loop failures with explicit cycle classification.
type LoopFailureEnvelope struct {
	Path           string
	Class          CycleFailureClass
	Recoverability failure.FailureRecoverability
	OrderClass     OrderFailureClass
	AgentMistake   *AgentMistakeEnvelope
	OrderID        string
	StageIndex     int
	Message        string
	Cause          error
}

func (e LoopFailureEnvelope) Error() string {
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = "loop failed"
	}
	if e.Cause == nil {
		return message
	}
	return message + ": " + e.Cause.Error()
}

func (e LoopFailureEnvelope) Unwrap() error {
	return e.Cause
}

func recoverabilityForCycleFailureClass(class CycleFailureClass) failure.FailureRecoverability {
	recoverability, ok := recoverabilityByCycleFailureClass[class]
	if !ok {
		return failure.FailureRecoverabilityHard
	}
	return recoverability
}

func newSystemHardLoopFailure(path string, message string, cause error) LoopFailureEnvelope {
	return LoopFailureEnvelope{
		Path:           path,
		Class:          CycleFailureClassSystemHard,
		Recoverability: recoverabilityForCycleFailureClass(CycleFailureClassSystemHard),
		StageIndex:     -1,
		Message:        message,
		Cause:          cause,
	}
}

func newOrderHardLoopFailure(
	path string,
	orderClass OrderFailureClass,
	orderID string,
	stageIndex int,
	message string,
	cause error,
) LoopFailureEnvelope {
	return LoopFailureEnvelope{
		Path:           path,
		Class:          CycleFailureClassOrderHard,
		Recoverability: recoverabilityForCycleFailureClass(CycleFailureClassOrderHard),
		OrderClass:     orderClass,
		OrderID:        strings.TrimSpace(orderID),
		StageIndex:     stageIndex,
		Message:        message,
		Cause:          cause,
	}
}

func newDegradeLoopFailure(path string, message string, cause error) LoopFailureEnvelope {
	return LoopFailureEnvelope{
		Path:           path,
		Class:          CycleFailureClassDegradeContinue,
		Recoverability: recoverabilityForCycleFailureClass(CycleFailureClassDegradeContinue),
		StageIndex:     -1,
		Message:        message,
		Cause:          cause,
	}
}

func (l *Loop) recordLoopFailure(envelope LoopFailureEnvelope) {
	env := envelope
	env.AgentMistake = cloneAgentMistakeEnvelope(envelope.AgentMistake)
	l.lastLoopFailure = &env
	if l.logger == nil {
		return
	}

	attrs := []any{
		"path", env.Path,
		"class", string(env.Class),
		"recoverability", string(env.Recoverability),
	}
	if env.OrderID != "" {
		attrs = append(attrs, "order_id", env.OrderID)
	}
	if env.StageIndex >= 0 {
		attrs = append(attrs, "stage_index", env.StageIndex)
	}
	if env.OrderClass != "" {
		attrs = append(attrs, "order_class", string(env.OrderClass))
	}
	if env.AgentMistake != nil {
		attrs = append(
			attrs,
			"failure_owner", string(env.AgentMistake.Owner),
			"failure_scope", string(env.AgentMistake.Scope),
			"failure_class", string(env.AgentMistake.FailureClass),
		)
		if reason := agentMistakeReason(env.AgentMistake); reason != "" {
			attrs = append(attrs, "agent_mistake_reason", reason)
		}
	}
	if env.Cause != nil {
		attrs = append(attrs, "cause", env.Cause)
	}
	l.logger.Warn(strings.TrimSpace(env.Message), attrs...)
}

func (l *Loop) classifySystemHard(path string, message string, cause error) error {
	if cause == nil {
		return nil
	}
	var existing LoopFailureEnvelope
	if errors.As(cause, &existing) {
		l.recordLoopFailure(existing)
		return cause
	}
	envelope := newSystemHardLoopFailure(path, message, cause)
	l.recordLoopFailure(envelope)
	return envelope
}

func (l *Loop) classifyOrderHard(
	path string,
	orderClass OrderFailureClass,
	orderID string,
	stageIndex int,
	message string,
	cause error,
) {
	envelope := newOrderHardLoopFailure(path, orderClass, orderID, stageIndex, message, cause)
	l.recordLoopFailure(envelope)
}

func (l *Loop) classifyDegrade(path string, message string, cause error) {
	envelope := newDegradeLoopFailure(path, message, cause)
	l.recordLoopFailure(envelope)
}

func (l *Loop) classifySchedulerMistake(path string, message string, cause error, reason SchedulerMistakeReason) {
	envelope := newDegradeLoopFailure(path, message, cause)
	mistake := newSchedulerMistakeEnvelope(reason)
	envelope.AgentMistake = &mistake
	l.recordLoopFailure(envelope)
}

func (l *Loop) classifyCookMistake(
	path string,
	orderClass OrderFailureClass,
	orderID string,
	stageIndex int,
	message string,
	reason CookMistakeReason,
) {
	envelope := newOrderHardLoopFailure(path, orderClass, orderID, stageIndex, message, nil)
	mistake := newCookMistakeEnvelope(reason, orderID, stageIndex)
	envelope.AgentMistake = &mistake
	l.recordLoopFailure(envelope)
}

func formatCycleFailureMessage(path string, action string) string {
	trimmedAction := strings.TrimSpace(action)
	if trimmedAction == "" {
		trimmedAction = "operation"
	}
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return fmt.Sprintf("cycle %s failed", trimmedAction)
	}
	return fmt.Sprintf("cycle %s failed at %s", trimmedAction, trimmedPath)
}
