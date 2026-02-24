package dispatcher

import "context"

// StreamingBackend starts and controls sessions that emit live output streams.
type StreamingBackend interface {
	Start(ctx context.Context, config StreamStartConfig) (StreamHandle, error)
	IsAlive(ctx context.Context, handle StreamHandle) (bool, error)
	Kill(ctx context.Context, handle StreamHandle) error
}

type SyncResult struct {
	Type   string `json:"type,omitempty"`
	Branch string `json:"branch,omitempty"`
}

const (
	SyncResultTypeNone   = "none"
	SyncResultTypeBranch = "branch"
)

// StreamingSyncBacker is implemented by streaming backends that can sync
// remote changes back to the local repository after a session completes.
type StreamingSyncBacker interface {
	SyncBack(ctx context.Context, sessionID string) (SyncResult, error)
}

// PollingBackend starts and controls sessions that expose state via polling.
type PollingBackend interface {
	Launch(ctx context.Context, config PollLaunchConfig) (string, error)
	PollStatus(ctx context.Context, remoteID string) (RemoteStatus, error)
	GetConversation(ctx context.Context, remoteID string) ([]ConversationMessage, error)
	Stop(ctx context.Context, remoteID string) error
	Delete(ctx context.Context, remoteID string) error
}
