package loop

import (
	"strings"

	"github.com/poteto/noodle/internal/failure"
)

// FailureProjection is the canonical failure classification payload projected
// into control acknowledgements and loop event payloads.
type FailureProjection struct {
	Class          failure.FailureClass          `json:"class"`
	Recoverability failure.FailureRecoverability `json:"recoverability"`
	Owner          failure.FailureOwner          `json:"owner,omitempty"`
	Scope          failure.FailureScope          `json:"scope,omitempty"`
}

// ControlAckFailure is projected into control acknowledgement rows when a
// command fails.
type ControlAckFailure struct {
	FailureProjection
}

// EventFailureMetadata is projected into loop event payloads for failure
// events. It carries canonical failure class/recoverability plus loop-local
// cycle/order classification.
type EventFailureMetadata struct {
	FailureProjection
	CycleClass CycleFailureClass `json:"cycle_class,omitempty"`
	OrderClass OrderFailureClass `json:"order_class,omitempty"`
}

func newFailureProjection(
	class failure.FailureClass,
	owner failure.FailureOwner,
	scope failure.FailureScope,
) FailureProjection {
	return FailureProjection{
		Class:          class,
		Recoverability: failure.RecoverabilityForClass(class),
		Owner:          owner,
		Scope:          scope,
	}
}

func controlAckFailureForInvalidCommandJSON() ControlAckFailure {
	return ControlAckFailure{
		FailureProjection: newFailureProjection(
			failure.FailureClassBackendRecoverable,
			failure.FailureOwnerBackend,
			failure.FailureScopeSystem,
		),
	}
}

func controlAckFailureForCommand(cmd ControlCommand) ControlAckFailure {
	scope := failure.FailureScopeSystem
	if strings.TrimSpace(cmd.OrderID) != "" {
		scope = failure.FailureScopeOrder
	} else if strings.TrimSpace(cmd.Name) != "" || strings.TrimSpace(cmd.Target) != "" {
		scope = failure.FailureScopeSession
	}
	return ControlAckFailure{
		FailureProjection: newFailureProjection(
			failure.FailureClassBackendRecoverable,
			failure.FailureOwnerBackend,
			scope,
		),
	}
}

func eventFailureMetadataForLoop(
	class CycleFailureClass,
	orderClass OrderFailureClass,
	agentMistake *AgentMistakeEnvelope,
) EventFailureMetadata {
	metadata := EventFailureMetadata{
		CycleClass: class,
		OrderClass: orderClass,
	}
	if agentMistake != nil {
		metadata.FailureProjection = FailureProjection{
			Class:          agentMistake.FailureClass,
			Recoverability: agentMistake.Recoverability,
			Owner:          agentMistake.Owner,
			Scope:          agentMistake.Scope,
		}
		return metadata
	}
	metadata.FailureProjection = defaultFailureProjectionForCycleClass(class)
	return metadata
}

func eventFailureMetadataForDispatch(
	envelope DispatchFailureEnvelope,
	orderClass OrderFailureClass,
) EventFailureMetadata {
	return EventFailureMetadata{
		FailureProjection: FailureProjection{
			Class:          envelope.FailureClass,
			Recoverability: envelope.Recoverability,
			Owner:          failure.FailureOwnerRuntime,
			Scope:          failure.FailureScopeSession,
		},
		CycleClass: CycleFailureClassOrderHard,
		OrderClass: orderClass,
	}
}

func defaultFailureProjectionForCycleClass(class CycleFailureClass) FailureProjection {
	switch class {
	case CycleFailureClassOrderHard:
		return newFailureProjection(
			failure.FailureClassBackendRecoverable,
			failure.FailureOwnerBackend,
			failure.FailureScopeOrder,
		)
	case CycleFailureClassDegradeContinue:
		return newFailureProjection(
			failure.FailureClassWarningOnly,
			failure.FailureOwnerBackend,
			failure.FailureScopeSystem,
		)
	case CycleFailureClassSystemHard:
		fallthrough
	default:
		return newFailureProjection(
			failure.FailureClassBackendInvariant,
			failure.FailureOwnerBackend,
			failure.FailureScopeSystem,
		)
	}
}
