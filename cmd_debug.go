package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/poteto/noodle/cmdmeta"
	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/internal/stringx"
	"github.com/spf13/cobra"
)

type debugDump struct {
	SchemaVersion int             `json:"schema_version"`
	Config        debugConfigDump `json:"config"`
	Runtime       debugRuntime    `json:"runtime"`
}

type debugConfigDump struct {
	RoutingDefaults debugRoutingDefaults `json:"routing_defaults"`
	Phases          map[string]string    `json:"phases,omitempty"`
}

type debugRoutingDefaults struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type debugRuntime struct {
	LoopState       string            `json:"loop_state"`
	Queue           debugQueue        `json:"queue"`
	Sessions        []debugSession    `json:"sessions"`
	FailedTargets   map[string]string `json:"failed_targets,omitempty"`
	ControlAckCount int               `json:"control_ack_count"`
}

type debugQueue struct {
	Items []debugQueueItem `json:"items"`
}

type debugQueueItem struct {
	ID       string `json:"id"`
	Title    string `json:"title,omitempty"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	Skill    string `json:"skill,omitempty"`
}

type debugSession struct {
	ID           string  `json:"id"`
	Status       string  `json:"status,omitempty"`
	LoopState    string  `json:"loop_state,omitempty"`
	Target       string  `json:"target,omitempty"`
	Provider     string  `json:"provider,omitempty"`
	Model        string  `json:"model,omitempty"`
	TotalCostUSD float64 `json:"total_cost_usd,omitempty"`
}

func newDebugCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "debug",
		Short: cmdmeta.Short("debug"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDebug(app)
		},
	}
}

func runDebug(app *App) error {
	runtimeDir, err := app.RuntimeDir()
	if err != nil {
		return err
	}

	cfg := config.DefaultConfig()
	if app != nil {
		cfg = app.Config
	}
	dump, err := buildDebugDump(cfg, runtimeDir)
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(dump, "", "  ")
	if err != nil {
		return fmt.Errorf("encode debug dump: %w", err)
	}
	fmt.Fprintln(os.Stdout, string(encoded))
	return nil
}

func buildDebugDump(cfg config.Config, runtimeDir string) (debugDump, error) {
	queue, err := readDebugQueue(filepath.Join(runtimeDir, "queue.json"))
	if err != nil {
		return debugDump{}, err
	}
	sessions, loopState, err := readDebugSessions(filepath.Join(runtimeDir, "sessions"))
	if err != nil {
		return debugDump{}, err
	}
	failedTargets, err := readFailedTargets(filepath.Join(runtimeDir, "failed.json"))
	if err != nil {
		return debugDump{}, err
	}
	controlAcks, err := countAckLines(filepath.Join(runtimeDir, "control-ack.ndjson"))
	if err != nil {
		return debugDump{}, err
	}

	phases := map[string]string{}
	for key, value := range cfg.Phases {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		phases[key] = strings.TrimSpace(value)
	}

	return debugDump{
		SchemaVersion: 1,
		Config: debugConfigDump{
			RoutingDefaults: debugRoutingDefaults{
				Provider: strings.TrimSpace(cfg.Routing.Defaults.Provider),
				Model:    strings.TrimSpace(cfg.Routing.Defaults.Model),
			},
			Phases: phases,
		},
		Runtime: debugRuntime{
			LoopState:       stringx.FirstNonEmpty(loopState, "running"),
			Queue:           queue,
			Sessions:        sessions,
			FailedTargets:   failedTargets,
			ControlAckCount: controlAcks,
		},
	}, nil
}

func readDebugQueue(path string) (debugQueue, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return debugQueue{}, nil
		}
		return debugQueue{}, fmt.Errorf("read queue: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return debugQueue{}, nil
	}

	var wrapped struct {
		Items []debugQueueItem `json:"items"`
	}
	if err := json.Unmarshal(data, &wrapped); err == nil {
		return debugQueue{Items: wrapped.Items}, nil
	}
	var bare []debugQueueItem
	if err := json.Unmarshal(data, &bare); err == nil {
		return debugQueue{Items: bare}, nil
	}
	return debugQueue{}, fmt.Errorf("parse queue.json")
}

func readDebugSessions(sessionsDir string) ([]debugSession, string, error) {
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "running", nil
		}
		return nil, "", fmt.Errorf("read sessions directory: %w", err)
	}

	sessions := make([]debugSession, 0, len(entries))
	loopState := "running"
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sessionID := strings.TrimSpace(entry.Name())
		if sessionID == "" {
			continue
		}
		sessionPath := filepath.Join(sessionsDir, sessionID)

		current := debugSession{ID: sessionID}
		metaPath := filepath.Join(sessionPath, "meta.json")
		metaData, metaErr := os.ReadFile(metaPath)
		if metaErr == nil {
			var meta struct {
				Status       string  `json:"status"`
				LoopState    string  `json:"loop_state"`
				Target       string  `json:"target"`
				TotalCostUSD float64 `json:"total_cost_usd"`
			}
			if err := json.Unmarshal(metaData, &meta); err != nil {
				return nil, "", fmt.Errorf("parse session meta %s: %w", sessionID, err)
			}
			current.Status = strings.ToLower(strings.TrimSpace(meta.Status))
			current.LoopState = strings.ToLower(strings.TrimSpace(meta.LoopState))
			current.Target = strings.TrimSpace(meta.Target)
			current.TotalCostUSD = meta.TotalCostUSD
			loopState = pickLoopState(loopState, current.LoopState)
			loopState = pickLoopState(loopState, current.Status)
		} else if !os.IsNotExist(metaErr) {
			return nil, "", fmt.Errorf("read session meta %s: %w", sessionID, metaErr)
		}

		spawnPath := filepath.Join(sessionPath, "spawn.json")
		spawnData, spawnErr := os.ReadFile(spawnPath)
		if spawnErr == nil {
			var spawn struct {
				Provider string `json:"provider"`
				Model    string `json:"model"`
			}
			if err := json.Unmarshal(spawnData, &spawn); err != nil {
				return nil, "", fmt.Errorf("parse spawn metadata %s: %w", sessionID, err)
			}
			current.Provider = strings.TrimSpace(spawn.Provider)
			current.Model = strings.TrimSpace(spawn.Model)
		} else if !os.IsNotExist(spawnErr) {
			return nil, "", fmt.Errorf("read spawn metadata %s: %w", sessionID, spawnErr)
		}

		sessions = append(sessions, current)
	}

	sort.Slice(sessions, func(i, j int) bool { return sessions[i].ID < sessions[j].ID })
	return sessions, loopState, nil
}

func readFailedTargets(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("read failed targets: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return map[string]string{}, nil
	}
	var failed map[string]string
	if err := json.Unmarshal(data, &failed); err != nil {
		return nil, fmt.Errorf("parse failed targets: %w", err)
	}
	if failed == nil {
		return map[string]string{}, nil
	}
	return failed, nil
}

func countAckLines(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read control ack log: %w", err)
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count, nil
}
