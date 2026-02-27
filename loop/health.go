package loop

import "strings"

func (l *Loop) drainRuntimeHealth() {
	// Terminal events first (HealthDead) — reliable, must not be dropped.
	for _, runtime := range l.deps.Runtimes {
		if runtime == nil {
			continue
		}
		ch := runtime.TerminalHealth()
		if ch == nil {
			continue
		}
		for {
			select {
			case event, ok := <-ch:
				if !ok {
					goto nextTerminal
				}
				sessionID := strings.TrimSpace(event.SessionID)
				if sessionID == "" {
					continue
				}
				latest, exists := l.sessionHealth[sessionID]
				if exists && event.Seq < latest.Seq {
					continue
				}
				l.sessionHealth[sessionID] = event
			default:
				goto nextTerminal
			}
		}
	nextTerminal:
	}

	// Info events (HealthStuck/HealthIdle/HealthHealthy) — non-blocking, last-writer-wins.
	for _, runtime := range l.deps.Runtimes {
		if runtime == nil {
			continue
		}
		ch := runtime.InfoHealth()
		if ch == nil {
			continue
		}
		for {
			select {
			case event, ok := <-ch:
				if !ok {
					goto nextInfo
				}
				sessionID := strings.TrimSpace(event.SessionID)
				if sessionID == "" {
					continue
				}
				latest, exists := l.sessionHealth[sessionID]
				if exists && event.Seq < latest.Seq {
					continue
				}
				l.sessionHealth[sessionID] = event
			default:
				goto nextInfo
			}
		}
	nextInfo:
	}
}
