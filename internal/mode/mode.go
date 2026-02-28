// Package mode provides unified mode gates and epoch-stamped effect validation.
//
// A single global RunMode (auto, supervised, manual) deterministically gates
// schedule, dispatch, retry, and merge behavior. Each mode transition increments
// a monotonic ModeEpoch. Effects are stamped with their creation epoch; before
// an effect executor commits a result, it validates the epoch — stale effects
// are cancelled rather than applied.
package mode

import (
	"time"

	"github.com/poteto/noodle/internal/state"
)

// ModeEpoch is a monotonic counter incremented on every mode transition.
// Strictly ordered, cheaply comparable, and persisted in canonical state.
type ModeEpoch uint64

// EpochResult is the outcome of validating an effect's epoch against the
// current epoch.
type EpochResult string

const (
	EpochValid    EpochResult = "valid"
	EpochStale    EpochResult = "stale"
	EpochDeferred EpochResult = "deferred"
)

// ModeTransitionRecord tracks a single mode change for audit purposes.
type ModeTransitionRecord struct {
	FromMode    state.RunMode `json:"from_mode"`
	ToMode      state.RunMode `json:"to_mode"`
	Epoch       ModeEpoch     `json:"epoch"`
	RequestedBy string        `json:"requested_by"`
	Reason      string        `json:"reason"`
	AppliedAt   time.Time     `json:"applied_at"`
}

// ModeState is the canonical mode tracking state, intended for embedding
// in the top-level canonical state.
type ModeState struct {
	RequestedMode state.RunMode          `json:"requested_mode"`
	EffectiveMode state.RunMode          `json:"effective_mode"`
	Epoch         ModeEpoch              `json:"epoch"`
	Transitions   []ModeTransitionRecord `json:"transitions"`
}

// maxTransitionHistory is the number of recent transitions retained for audit.
const maxTransitionHistory = 50

// StampedEffect pairs an effect ID with the epoch at which it was created.
type StampedEffect struct {
	EffectID string    `json:"effect_id"`
	Epoch    ModeEpoch `json:"epoch"`
}

// NewModeState creates an initial ModeState with the given mode and epoch 0.
func NewModeState(initial state.RunMode) ModeState {
	return ModeState{
		RequestedMode: initial,
		EffectiveMode: initial,
		Epoch:         0,
		Transitions:   []ModeTransitionRecord{},
	}
}

// TransitionMode returns a new ModeState reflecting a mode change. The epoch
// is incremented even if newMode equals the current mode (the transition is
// still recorded). Transition history is capped at maxTransitionHistory.
func TransitionMode(current ModeState, newMode state.RunMode, requestedBy, reason string, now time.Time) ModeState {
	next := current
	next.Epoch++
	next.RequestedMode = newMode
	next.EffectiveMode = newMode

	record := ModeTransitionRecord{
		FromMode:    current.EffectiveMode,
		ToMode:      newMode,
		Epoch:       next.Epoch,
		RequestedBy: requestedBy,
		Reason:      reason,
		AppliedAt:   now,
	}

	// Copy transitions slice to avoid aliasing the original.
	transitions := make([]ModeTransitionRecord, len(current.Transitions), len(current.Transitions)+1)
	copy(transitions, current.Transitions)
	transitions = append(transitions, record)

	// Cap history.
	if len(transitions) > maxTransitionHistory {
		transitions = transitions[len(transitions)-maxTransitionHistory:]
	}
	next.Transitions = transitions

	return next
}

// StampEffect stamps an effect with the given epoch.
func StampEffect(epoch ModeEpoch, effectID string) StampedEffect {
	return StampedEffect{
		EffectID: effectID,
		Epoch:    epoch,
	}
}

// ValidateEpoch checks whether an effect created at effectEpoch should proceed
// given the currentEpoch. Same epoch means valid; older means stale.
func ValidateEpoch(effectEpoch, currentEpoch ModeEpoch) EpochResult {
	if effectEpoch == currentEpoch {
		return EpochValid
	}
	if effectEpoch < currentEpoch {
		return EpochStale
	}
	// Future epoch — should not happen in practice, but treat as deferred.
	return EpochDeferred
}
