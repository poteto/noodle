package mode

import "github.com/poteto/noodle/internal/state"

// Action names used by BlockedReason and the gate matrix.
const (
	ActionSchedule  = "schedule"
	ActionDispatch  = "dispatch"
	ActionRetry     = "retry"
	ActionAutoMerge = "auto_merge"
)

// ModeGate determines what actions are allowed in each run mode.
type ModeGate struct{}

// CanSchedule reports whether scheduling is allowed in the given mode.
// auto and supervised allow scheduling; manual does not.
func (ModeGate) CanSchedule(m state.RunMode) bool {
	return m != state.RunModeManual
}

// CanDispatch reports whether dispatch is allowed in the given mode.
// auto and supervised allow dispatch; manual does not.
func (ModeGate) CanDispatch(m state.RunMode) bool {
	return m != state.RunModeManual
}

// CanRetry reports whether automatic retry is allowed in the given mode.
// Only auto mode allows automatic retry.
func (ModeGate) CanRetry(m state.RunMode) bool {
	return m == state.RunModeAuto
}

// CanAutoMerge reports whether automatic merge is allowed in the given mode.
// Only auto mode allows automatic merge (per-skill overrides are the caller's
// responsibility).
func (ModeGate) CanAutoMerge(m state.RunMode) bool {
	return m == state.RunModeAuto
}

// BlockedReason returns a human-readable reason code explaining why the given
// action is blocked in the given mode. Returns an empty string if the action
// is not blocked.
func (g ModeGate) BlockedReason(m state.RunMode, action string) string {
	switch action {
	case ActionSchedule:
		if !g.CanSchedule(m) {
			return "manual mode requires explicit scheduling"
		}
	case ActionDispatch:
		if !g.CanDispatch(m) {
			return "manual mode requires explicit dispatch"
		}
	case ActionRetry:
		if !g.CanRetry(m) {
			switch m {
			case state.RunModeSupervised:
				return "supervised mode requires manual retry"
			case state.RunModeManual:
				return "manual mode requires manual retry"
			}
		}
	case ActionAutoMerge:
		if !g.CanAutoMerge(m) {
			switch m {
			case state.RunModeSupervised:
				return "supervised mode requires merge approval"
			case state.RunModeManual:
				return "manual mode requires merge approval"
			}
		}
	}
	return ""
}
