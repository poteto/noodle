package monitor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/poteto/noodle/dispatcher"
)

type HeartbeatObserver struct {
	runtimeDir string
	now        func() time.Time
}

func NewHeartbeatObserver(runtimeDir string) *HeartbeatObserver {
	return &HeartbeatObserver{
		runtimeDir: strings.TrimSpace(runtimeDir),
		now:        time.Now,
	}
}

func (o *HeartbeatObserver) Observe(sessionID string) (Observation, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return Observation{}, fmt.Errorf("session ID is required")
	}
	if o.runtimeDir == "" {
		return Observation{}, fmt.Errorf("runtime directory is required")
	}

	obs, err := canonicalLogObservation(o.runtimeDir, sessionID)
	if err != nil {
		return Observation{}, err
	}

	data, err := os.ReadFile(filepath.Join(o.runtimeDir, "sessions", sessionID, "heartbeat.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return obs, nil
		}
		return Observation{}, fmt.Errorf("read heartbeat: %w", err)
	}

	var heartbeat struct {
		Timestamp  time.Time `json:"timestamp"`
		TTLSeconds int64     `json:"ttl_seconds"`
	}
	if err := json.Unmarshal(data, &heartbeat); err != nil {
		return Observation{}, fmt.Errorf("parse heartbeat: %w", err)
	}
	if heartbeat.Timestamp.IsZero() || heartbeat.TTLSeconds <= 0 {
		return obs, nil
	}
	maxAge := time.Duration(heartbeat.TTLSeconds*2) * time.Second
	obs.Alive = o.now().UTC().Sub(heartbeat.Timestamp.UTC()) <= maxAge
	return obs, nil
}

type CompositeObserver struct {
	runtimeDir string
	local      Observer
	remote     Observer
}

func NewCompositeObserver(runtimeDir string, local Observer, remote Observer) *CompositeObserver {
	return &CompositeObserver{
		runtimeDir: strings.TrimSpace(runtimeDir),
		local:      local,
		remote:     remote,
	}
}

func (o *CompositeObserver) Observe(sessionID string) (Observation, error) {
	observer, err := o.observerForSession(sessionID)
	if err != nil {
		return Observation{}, err
	}
	return observer.Observe(sessionID)
}

func (o *CompositeObserver) observerForSession(sessionID string) (Observer, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("session ID is required")
	}
	if o.runtimeDir == "" {
		return nil, fmt.Errorf("runtime directory is required")
	}
	if o.local == nil || o.remote == nil {
		return nil, fmt.Errorf("composite observer not configured")
	}

	selected := o.local
	if runtime, err := readSessionRuntime(o.runtimeDir, sessionID); err == nil {
		if runtime != "" && runtime != "process" && runtime != "tmux" {
			selected = o.remote
		}
	}
	return selected, nil
}

func readSessionRuntime(runtimeDir, sessionID string) (string, error) {
	data, err := os.ReadFile(filepath.Join(runtimeDir, "sessions", sessionID, "spawn.json"))
	if err != nil {
		return "", err
	}
	var payload struct {
		Runtime string `json:"runtime"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", err
	}
	return dispatcher.NormalizeRuntime(payload.Runtime), nil
}

func canonicalLogObservation(runtimeDir, sessionID string) (Observation, error) {
	canonicalPath := filepath.Join(runtimeDir, "sessions", sessionID, "canonical.ndjson")
	stat, err := os.Stat(canonicalPath)
	if err != nil && !os.IsNotExist(err) {
		return Observation{}, fmt.Errorf("stat canonical events: %w", err)
	}

	obs := Observation{SessionID: sessionID}
	if stat != nil {
		obs.LogMTime = stat.ModTime().UTC()
		obs.LogSize = stat.Size()
	}
	return obs, nil
}
