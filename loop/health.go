package loop

import "strings"

func (l *Loop) drainRuntimeHealth() {
	for _, runtime := range l.deps.Runtimes {
		if runtime == nil {
			continue
		}
		health := runtime.Health()
		if health == nil {
			continue
		}
		for {
			select {
			case event, ok := <-health:
				if !ok {
					goto nextRuntime
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
				goto nextRuntime
			}
		}

	nextRuntime:
	}
}
