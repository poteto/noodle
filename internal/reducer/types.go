package reducer

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/poteto/noodle/internal/ingest"
	"github.com/poteto/noodle/internal/state"
)

// Reducer is a pure state-transition function.
//
// It must never perform side effects. It only transforms state and emits
// declarative effects for external executors.
type Reducer func(state.State, ingest.StateEvent) (state.State, []Effect)

// EffectType identifies the side effect to execute after a state transition.
type EffectType string

const (
	// EffectDispatch dispatches/launches a session.
	// STUB: concrete launch semantics depend on Phase 4 two-phase launch.
	EffectDispatch EffectType = "dispatch"

	// EffectMerge merges a completed stage worktree.
	EffectMerge EffectType = "merge"

	// EffectWriteProjection writes projection files from canonical state.
	EffectWriteProjection EffectType = "write_projection"

	// EffectAck emits acknowledgements for committed outcomes.
	EffectAck EffectType = "ack"

	// EffectCleanup cleans up worktrees/resources.
	EffectCleanup EffectType = "cleanup"
)

// Effect is a side effect planned by the reducer.
type Effect struct {
	EffectID  string          `json:"effect_id"`
	Type      EffectType      `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

// EffectResultStatus is the terminal outcome of effect execution.
type EffectResultStatus string

const (
	EffectResultCompleted EffectResultStatus = "completed"
	EffectResultFailed    EffectResultStatus = "failed"
	EffectResultCancelled EffectResultStatus = "cancelled"
	EffectResultDeferred  EffectResultStatus = "deferred"
)

// EffectResult records one execution outcome for an effect.
type EffectResult struct {
	EffectID  string             `json:"effect_id"`
	Status    EffectResultStatus `json:"status"`
	Error     string             `json:"error"`
	Timestamp time.Time          `json:"timestamp"`
}

// EffectLedgerStatus tracks effect lifecycle in durable state.
type EffectLedgerStatus string

const (
	EffectLedgerPending   EffectLedgerStatus = "pending"
	EffectLedgerRunning   EffectLedgerStatus = "running"
	EffectLedgerDone      EffectLedgerStatus = "done"
	EffectLedgerFailed    EffectLedgerStatus = "failed"
	EffectLedgerCancelled EffectLedgerStatus = "cancelled"
	EffectLedgerDeferred  EffectLedgerStatus = "deferred"
)

// EffectLedgerRecord stores the lifecycle state for one effect.
type EffectLedgerRecord struct {
	EffectID      string             `json:"effect_id"`
	Effect        Effect             `json:"effect"`
	Status        EffectLedgerStatus `json:"status"`
	Attempts      int                `json:"attempts"`
	LastAttemptAt time.Time          `json:"last_attempt_at"`
	Result        *EffectResult      `json:"result"`
}

func effectIDForEvent(eventID ingest.EventID, effectIndex int) string {
	return fmt.Sprintf("event-%d-effect-%d", eventID, effectIndex)
}

func makeEffect(event ingest.StateEvent, effectIndex int, effectType EffectType, payload any) Effect {
	return Effect{
		EffectID:  effectIDForEvent(event.ID, effectIndex),
		Type:      effectType,
		Payload:   mustRawJSON(payload),
		CreatedAt: event.Timestamp,
	}
}

func mustRawJSON(payload any) json.RawMessage {
	data, err := json.Marshal(payload)
	if err != nil {
		// Payloads are reducer-owned structs/maps and must always encode.
		panic(fmt.Sprintf("effect payload encoding failed: %v", err))
	}
	return json.RawMessage(data)
}
