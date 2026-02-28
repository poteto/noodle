// Package rtcap defines runtime capability contracts for dispatch runtimes.
//
// Instead of branching on runtime names, callers query typed capabilities
// to decide polling, steering, sync, and heartbeat behavior.
package rtcap

// Capability is a typed runtime capability.
type Capability string

const (
	// CapSteerable indicates the session supports live message injection.
	CapSteerable Capability = "steerable"

	// CapPolling indicates the session status must be polled (no push completion).
	CapPolling Capability = "polling"

	// CapRemoteSync indicates the session runs remotely and needs branch push/pull.
	CapRemoteSync Capability = "remote_sync"

	// CapHeartbeat indicates the session emits periodic heartbeats for liveness.
	CapHeartbeat Capability = "heartbeat"
)

// RuntimeCapabilities describes the capability set for a runtime.
type RuntimeCapabilities struct {
	Name string              `json:"name"`
	Caps map[Capability]bool `json:"caps"`
}

// Has reports whether the runtime has the given capability.
func (rc RuntimeCapabilities) Has(cap Capability) bool {
	return rc.Caps[cap]
}

// All returns all capabilities this runtime has.
func (rc RuntimeCapabilities) All() []Capability {
	var out []Capability
	for cap, enabled := range rc.Caps {
		if enabled {
			out = append(out, cap)
		}
	}
	return out
}

// NeedsPolling reports whether the runtime requires polling for session status.
func NeedsPolling(caps RuntimeCapabilities) bool {
	return caps.Has(CapPolling)
}

// NeedsRemoteSync reports whether the runtime requires branch push/pull.
func NeedsRemoteSync(caps RuntimeCapabilities) bool {
	return caps.Has(CapRemoteSync)
}

// CanSteer reports whether the runtime supports live message injection.
func CanSteer(caps RuntimeCapabilities) bool {
	return caps.Has(CapSteerable)
}
