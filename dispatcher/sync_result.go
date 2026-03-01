package dispatcher

type SyncResult struct {
	Type   string `json:"type,omitempty"`
	Branch string `json:"branch,omitempty"`
}

const (
	SyncResultTypeNone   = "none"
	SyncResultTypeBranch = "branch"
)
