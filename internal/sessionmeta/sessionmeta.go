package sessionmeta

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Meta is the canonical runtime session metadata contract read from meta.json.
type Meta struct {
	SessionID             string    `json:"session_id"`
	Status                string    `json:"status"`
	Runtime               string    `json:"runtime,omitempty"`
	Provider              string    `json:"provider,omitempty"`
	Model                 string    `json:"model,omitempty"`
	TotalCostUSD          float64   `json:"total_cost_usd"`
	DurationSeconds       int64     `json:"duration_seconds"`
	LastActivity          time.Time `json:"last_activity"`
	CurrentAction         string    `json:"current_action,omitempty"`
	LoopState             string    `json:"loop_state,omitempty"`
	Health                string    `json:"health"`
	ContextWindowUsagePct float64   `json:"context_window_usage_pct"`
	RetryCount            int       `json:"retry_count"`
	Alive                 bool      `json:"alive"`
	Stuck                 bool      `json:"stuck"`
	LogSize               int64     `json:"log_size"`
	UpdatedAt             time.Time `json:"updated_at"`
	IdleSeconds           int64     `json:"idle_seconds"`
	StuckThresholdSeconds int64     `json:"stuck_threshold_seconds"`
}

func Read(path string) (Meta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Meta{}, err
	}
	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		return Meta{}, fmt.Errorf("parse session meta: %w", err)
	}
	meta.SessionID = strings.TrimSpace(meta.SessionID)
	meta.Status = strings.ToLower(strings.TrimSpace(meta.Status))
	meta.Runtime = strings.TrimSpace(meta.Runtime)
	meta.Provider = strings.TrimSpace(meta.Provider)
	meta.Model = strings.TrimSpace(meta.Model)
	meta.CurrentAction = strings.TrimSpace(meta.CurrentAction)
	meta.LoopState = strings.TrimSpace(meta.LoopState)
	meta.Health = strings.ToLower(strings.TrimSpace(meta.Health))
	return meta, nil
}

func ReadSession(runtimeDir, sessionID string) (Meta, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return Meta{}, fmt.Errorf("session ID is required")
	}
	path := filepath.Join(runtimeDir, "sessions", sessionID, "meta.json")
	meta, err := Read(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Meta{}, os.ErrNotExist
		}
		return Meta{}, err
	}
	if meta.SessionID == "" {
		meta.SessionID = sessionID
	}
	return meta, nil
}

func ReadAll(runtimeDir string) ([]Meta, error) {
	dir := filepath.Join(runtimeDir, "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Meta{}, nil
		}
		return nil, fmt.Errorf("read sessions directory: %w", err)
	}
	out := make([]Meta, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sessionID := strings.TrimSpace(entry.Name())
		if sessionID == "" {
			continue
		}
		meta, err := ReadSession(runtimeDir, sessionID)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		out = append(out, meta)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].SessionID < out[j].SessionID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out, nil
}
