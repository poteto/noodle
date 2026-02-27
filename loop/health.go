package loop

import (
	nrt "github.com/poteto/noodle/runtime"
)

// drainHealthEvents reads pending health events from all registered runtimes
// and updates the per-session health map. Stale events (lower sequence number
// than the last seen for that session) are discarded.
func (l *Loop) drainHealthEvents() {
	rtMap, ok := l.deps.Dispatcher.(*nrt.RuntimeMap)
	if !ok {
		return
	}
	for _, ch := range rtMap.HealthChannels() {
		for {
			select {
			case ev := <-ch:
				l.applyHealthEvent(ev)
			default:
				goto nextChannel
			}
		}
	nextChannel:
	}
}

func (l *Loop) applyHealthEvent(ev nrt.HealthEvent) {
	if l.sessionHealth == nil {
		l.sessionHealth = map[string]nrt.HealthEvent{}
	}
	existing, exists := l.sessionHealth[ev.SessionID]
	if exists && ev.Seq <= existing.Seq {
		return
	}
	l.sessionHealth[ev.SessionID] = ev
}
