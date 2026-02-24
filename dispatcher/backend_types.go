package dispatcher

import (
	"io"
	"time"
)

// StreamStartConfig contains backend-agnostic inputs for starting a stream session.
type StreamStartConfig struct {
	SessionID   string
	Command     string
	Env         []string
	WorkingDir  string
	Provider    string
	RuntimeKind string
}

// PollLaunchConfig contains backend-agnostic inputs for launching a poll session.
type PollLaunchConfig struct {
	Prompt     string
	Repository string
	Model      string
	Branch     string
}

// StreamHandle identifies one active streaming backend process.
type StreamHandle struct {
	Stdout   io.Reader
	ID       string
	Provider string
}

// RemoteStatus is a backend-neutral execution state for polling backends.
type RemoteStatus string

const (
	RemoteStatusRunning   RemoteStatus = "running"
	RemoteStatusCompleted RemoteStatus = "completed"
	RemoteStatusFailed    RemoteStatus = "failed"
	RemoteStatusExpired   RemoteStatus = "expired"
	RemoteStatusUnknown   RemoteStatus = "unknown"
)

// ConversationMessage represents one provider message in a polling transcript.
type ConversationMessage struct {
	Role      string
	Text      string
	Timestamp time.Time
}
