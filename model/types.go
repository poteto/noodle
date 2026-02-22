package model

// AgentID is a stable identifier for an agent session.
type AgentID string

// CookStatus is the lifecycle status for a cook session.
type CookStatus string

const (
	CookStatusSpawning  CookStatus = "spawning"
	CookStatusRunning   CookStatus = "running"
	CookStatusCompleted CookStatus = "completed"
	CookStatusFailed    CookStatus = "failed"
	CookStatusKilled    CookStatus = "killed"
)

// Provider identifies the execution provider for a cook.
type Provider string

const (
	ProviderClaude Provider = "claude"
	ProviderCodex  Provider = "codex"
)

// ModelPolicy captures provider and model selection policy.
type ModelPolicy struct {
	Provider       Provider `json:"provider"`
	Model          string   `json:"model"`
	ReasoningLevel string   `json:"reasoning_level,omitempty"`
}

// Cook is the core runtime actor record used across the kitchen brigade.
type Cook struct {
	ID       AgentID     `json:"id"`
	Provider Provider    `json:"provider"`
	Model    string      `json:"model"`
	Status   CookStatus  `json:"status"`
	Parent   *AgentID    `json:"parent,omitempty"`
	Policy   ModelPolicy `json:"policy"`
}
