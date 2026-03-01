package loop

import (
	"errors"
	"fmt"
	"strings"

	"github.com/poteto/noodle/internal/failure"
	loopruntime "github.com/poteto/noodle/runtime"
)

// AgentStartFailureClass classifies session start attempts by retry behavior.
type AgentStartFailureClass string

const (
	AgentStartFailureClassRetryable     AgentStartFailureClass = "retryable"
	AgentStartFailureClassFallback      AgentStartFailureClass = "fallback"
	AgentStartFailureClassUnrecoverable AgentStartFailureClass = "unrecoverable"
)

// DispatchFailureEnvelope wraps a start failure with explicit recoverability.
type DispatchFailureEnvelope struct {
	Class          AgentStartFailureClass
	FailureClass   failure.FailureClass
	Recoverability failure.FailureRecoverability
	Runtime        string
	Message        string
	Cause          error
}

func (e DispatchFailureEnvelope) Error() string {
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = "agent session start failed"
	}
	if e.Cause == nil {
		return message
	}
	return message + ": " + e.Cause.Error()
}

func (e DispatchFailureEnvelope) Unwrap() error {
	return e.Cause
}

// RuntimeFallbackOutcome records a successful runtime fallback dispatch path.
type RuntimeFallbackOutcome struct {
	Class            AgentStartFailureClass
	FailureClass     failure.FailureClass
	Recoverability   failure.FailureRecoverability
	RequestedRuntime string
	SelectedRuntime  string
	Message          string
	Cause            error
}

type runtimeNotConfiguredError struct {
	Runtime string
}

func (e runtimeNotConfiguredError) Error() string {
	return fmt.Sprintf("runtime %q not configured", strings.TrimSpace(e.Runtime))
}

func newRuntimeNotConfiguredError(runtimeName string) runtimeNotConfiguredError {
	return runtimeNotConfiguredError{Runtime: strings.ToLower(strings.TrimSpace(runtimeName))}
}

func classifyAgentStartFailure(runtimeName string, cause error) DispatchFailureEnvelope {
	normalizedRuntime := strings.ToLower(strings.TrimSpace(runtimeName))
	class := AgentStartFailureClassRetryable
	failureClass := failure.FailureClassAgentStartRetryable
	if normalizedRuntime == "process" || isRuntimeMisconfiguration(cause) || loopruntime.IsProcessStartFailure(cause) {
		class = AgentStartFailureClassUnrecoverable
		failureClass = failure.FailureClassAgentStartUnrecoverable
	}
	return DispatchFailureEnvelope{
		Class:          class,
		FailureClass:   failureClass,
		Recoverability: failure.RecoverabilityForClass(failureClass),
		Runtime:        normalizedRuntime,
		Message:        "agent session start failed",
		Cause:          cause,
	}
}

func newRuntimeFallbackOutcome(
	requestedRuntime string,
	selectedRuntime string,
	message string,
	cause error,
) RuntimeFallbackOutcome {
	failureClass := failure.FailureClassAgentStartRetryable
	return RuntimeFallbackOutcome{
		Class:            AgentStartFailureClassFallback,
		FailureClass:     failureClass,
		Recoverability:   failure.RecoverabilityForClass(failureClass),
		RequestedRuntime: strings.ToLower(strings.TrimSpace(requestedRuntime)),
		SelectedRuntime:  strings.ToLower(strings.TrimSpace(selectedRuntime)),
		Message:          strings.TrimSpace(message),
		Cause:            cause,
	}
}

func isRuntimeMisconfiguration(err error) bool {
	var typed runtimeNotConfiguredError
	return errors.As(err, &typed)
}

func asDispatchFailureEnvelope(err error) (DispatchFailureEnvelope, bool) {
	var envelope DispatchFailureEnvelope
	if !errors.As(err, &envelope) {
		return DispatchFailureEnvelope{}, false
	}
	return envelope, true
}
