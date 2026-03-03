package main

import (
	"strings"

	"github.com/poteto/noodle/internal/failure"
)

// StartBoundaryOutcome classifies startup boundary handling.
type StartBoundaryOutcome string

const (
	StartBoundaryOutcomeAbort        StartBoundaryOutcome = "abort"
	StartBoundaryOutcomePromptRepair StartBoundaryOutcome = "prompt-repair"
	StartBoundaryOutcomeWarningOnly  StartBoundaryOutcome = "warning-only"
)

// RepairPromptKind identifies the repair flow selected at startup.
type RepairPromptKind string

const (
	RepairPromptKindMissingScripts RepairPromptKind = "missing-scripts"
)

// RepairPromptOutcome captures typed repair prompt details for startup.
type RepairPromptOutcome struct {
	Kind         RepairPromptKind
	Provider     string
	SessionID    string
	WorktreePath string
}

// StartFailureEnvelope describes classified startup boundary outcomes.
type StartFailureEnvelope struct {
	Outcome        StartBoundaryOutcome
	Class          failure.FailureClass
	Recoverability failure.FailureRecoverability
	Message        string
	Cause          error
	RepairPrompt   *RepairPromptOutcome
}

func (e StartFailureEnvelope) Error() string {
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = "startup boundary failed"
	}
	if e.Cause == nil {
		return message
	}
	return message + ": " + e.Cause.Error()
}

func (e StartFailureEnvelope) Unwrap() error {
	return e.Cause
}

func newStartAbortEnvelope(
	message string,
	class failure.FailureClass,
	cause error,
) StartFailureEnvelope {
	return StartFailureEnvelope{
		Outcome:        StartBoundaryOutcomeAbort,
		Class:          class,
		Recoverability: failure.RecoverabilityForClass(class),
		Message:        message,
		Cause:          cause,
	}
}

func newStartRepairPromptEnvelope(
	message string,
	repair RepairPromptOutcome,
) StartFailureEnvelope {
	class := failure.FailureClassBackendRecoverable
	return StartFailureEnvelope{
		Outcome:        StartBoundaryOutcomePromptRepair,
		Class:          class,
		Recoverability: failure.RecoverabilityForClass(class),
		Message:        message,
		RepairPrompt:   &repair,
	}
}

func newStartWarningOnlyEnvelope(message string, cause error) StartFailureEnvelope {
	class := failure.FailureClassWarningOnly
	return StartFailureEnvelope{
		Outcome:        StartBoundaryOutcomeWarningOnly,
		Class:          class,
		Recoverability: failure.RecoverabilityForClass(class),
		Message:        message,
		Cause:          cause,
	}
}
