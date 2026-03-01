package monitor

import (
	"math"
	"strings"
	"time"
)

func DeriveSessionMeta(
	sessionID string,
	observation Observation,
	claims SessionClaims,
	previous SessionMeta,
	now time.Time,
	stuckThreshold time.Duration,
) SessionMeta {
	now = now.UTC()
	lastActivity := claims.LastEventAt
	if observation.LogMTime.After(lastActivity) {
		lastActivity = observation.LogMTime
	}
	idleSeconds := int64(0)
	if !lastActivity.IsZero() && now.After(lastActivity) {
		idleSeconds = int64(now.Sub(lastActivity).Seconds())
	}

	stuck := observation.Alive &&
		stuckThreshold > 0 &&
		!lastActivity.IsZero() &&
		now.Sub(lastActivity) > stuckThreshold

	persistentScheduler := isPersistentSchedulerSession(claims)
	terminalByClaims := (claims.Completed || claims.Failed) && !persistentScheduler

	status := SessionStatusExited
	switch {
	case terminalByClaims && claims.Failed:
		status = SessionStatusFailed
	case terminalByClaims:
		status = SessionStatusExited
	case stuck:
		status = SessionStatusStuck
	case observation.Alive:
		status = SessionStatusRunning
	case claims.Failed:
		status = SessionStatusFailed
	case claims.Completed:
		status = SessionStatusExited
	case !observation.Alive && claims.HasEvents:
		status = SessionStatusExited
	default:
		status = SessionStatusFailed
	}

	contextUsagePct := 0.0
	tokens := claims.TokensIn + claims.TokensOut
	if tokens > 0 {
		contextUsagePct = math.Min(100, (float64(tokens)/contextTokenBudget)*100)
	}

	health := HealthGreen
	switch status {
	case SessionStatusFailed, SessionStatusStuck:
		health = HealthRed
	case SessionStatusRunning:
		if contextUsagePct >= 80 {
			health = HealthYellow
		}
		if stuckThreshold > 0 &&
			!lastActivity.IsZero() &&
			now.Sub(lastActivity) > stuckThreshold/2 {
			health = HealthYellow
		}
	default:
		health = HealthYellow
	}

	alive := observation.Alive
	if terminalByClaims {
		alive = false
		stuck = false
	}

	durationSeconds := int64(0)
	if !claims.FirstEventAt.IsZero() {
		end := now
		if !observation.Alive && !claims.LastEventAt.IsZero() {
			end = claims.LastEventAt
		}
		if end.After(claims.FirstEventAt) {
			durationSeconds = int64(end.Sub(claims.FirstEventAt).Seconds())
		}
	}

	provider := strings.TrimSpace(claims.Provider)
	if provider == "" {
		provider = strings.TrimSpace(previous.Provider)
	}
	runtime := strings.TrimSpace(claims.Runtime)
	if runtime == "" {
		runtime = strings.TrimSpace(previous.Runtime)
	}
	model := strings.TrimSpace(claims.Model)
	if model == "" {
		model = strings.TrimSpace(previous.Model)
	}
	currentAction := strings.TrimSpace(claims.LastAction)
	if currentAction == "" {
		currentAction = strings.TrimSpace(previous.CurrentAction)
	}

	return SessionMeta{
		SessionID:               sessionID,
		Status:                  status,
		Runtime:                 runtime,
		Provider:                provider,
		Model:                   model,
		TotalCostUSD:            claims.TotalCostUSD,
		DurationSeconds:         durationSeconds,
		LastActivity:            lastActivity,
		CurrentAction:           currentAction,
		Health:                  health,
		ContextWindowUsagePct:   contextUsagePct,
		RetryCount:              previous.RetryCount,
		Alive:                   alive,
		Stuck:                   stuck,
		LogSize:                 observation.LogSize,
		UpdatedAt:               now,
		IdleSeconds:             idleSeconds,
		StuckThresholdSeconds:   int64(stuckThreshold.Seconds()),
		LastObservedProviderRaw: claims.Provider,
	}
}

func isPersistentSchedulerSession(claims SessionClaims) bool {
	return strings.EqualFold(strings.TrimSpace(claims.Skill), "schedule")
}
