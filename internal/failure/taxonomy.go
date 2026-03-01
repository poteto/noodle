// Package failure defines the canonical backend failure taxonomy.
package failure

// FailureOwner identifies who owns remediation for a failure.
type FailureOwner string

const (
	FailureOwnerBackend        FailureOwner = "backend"
	FailureOwnerSchedulerAgent FailureOwner = "scheduler_agent"
	FailureOwnerCookAgent      FailureOwner = "cook_agent"
	FailureOwnerRuntime        FailureOwner = "runtime"
)

// FailureScope identifies the blast radius of a failure.
type FailureScope string

const (
	FailureScopeSystem  FailureScope = "system"
	FailureScopeOrder   FailureScope = "order"
	FailureScopeStage   FailureScope = "stage"
	FailureScopeSession FailureScope = "session"
)

// FailureRecoverability identifies how execution should proceed after failure.
type FailureRecoverability string

const (
	FailureRecoverabilityHard        FailureRecoverability = "hard"
	FailureRecoverabilityRecoverable FailureRecoverability = "recoverable"
	FailureRecoverabilityDegrade     FailureRecoverability = "degrade"
)

// FailureClass identifies a canonical failure category at subsystem boundaries.
type FailureClass string

const (
	FailureClassBackendInvariant        FailureClass = "backend_invariant"
	FailureClassBackendRecoverable      FailureClass = "backend_recoverable"
	FailureClassAgentMistake            FailureClass = "agent_mistake"
	FailureClassAgentStartUnrecoverable FailureClass = "agent_start_unrecoverable"
	FailureClassAgentStartRetryable     FailureClass = "agent_start_retryable"
	FailureClassWarningOnly             FailureClass = "warning_only"
)

// UnknownClassRecoverabilityFallback is the explicit default used when class
// lookup misses. It keeps behavior deterministic for unmigrated or new classes.
const UnknownClassRecoverabilityFallback = FailureRecoverabilityHard

var knownFailureClasses = []FailureClass{
	FailureClassBackendInvariant,
	FailureClassBackendRecoverable,
	FailureClassAgentMistake,
	FailureClassAgentStartUnrecoverable,
	FailureClassAgentStartRetryable,
	FailureClassWarningOnly,
}

var recoverabilityByClass = map[FailureClass]FailureRecoverability{
	FailureClassBackendInvariant:        FailureRecoverabilityHard,
	FailureClassBackendRecoverable:      FailureRecoverabilityRecoverable,
	FailureClassAgentMistake:            FailureRecoverabilityRecoverable,
	FailureClassAgentStartUnrecoverable: FailureRecoverabilityHard,
	FailureClassAgentStartRetryable:     FailureRecoverabilityRecoverable,
	FailureClassWarningOnly:             FailureRecoverabilityDegrade,
}

// KnownFailureClasses returns the canonical classes in stable declaration order.
func KnownFailureClasses() []FailureClass {
	classes := make([]FailureClass, len(knownFailureClasses))
	copy(classes, knownFailureClasses)
	return classes
}

// IsKnownClass reports whether class is a canonical taxonomy class.
func IsKnownClass(class FailureClass) bool {
	_, ok := recoverabilityByClass[class]
	return ok
}

// RecoverabilityForClass maps a class to recoverability with explicit fallback
// behavior for unknown classes.
func RecoverabilityForClass(class FailureClass) FailureRecoverability {
	recoverability, ok := recoverabilityByClass[class]
	if !ok {
		return UnknownClassRecoverabilityFallback
	}
	return recoverability
}
