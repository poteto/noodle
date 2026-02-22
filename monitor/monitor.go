package monitor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Monitor struct {
	runtimeDir      string
	stuckThreshold  time.Duration
	pollInterval    time.Duration
	debounce        time.Duration
	observer        Observer
	claims          ClaimsReader
	tickets         TicketMaterializer
	now             func() time.Time
	mu              sync.Mutex
	lastRunFinished time.Time
}

func NewMonitor(runtimeDir string) *Monitor {
	runtimeDir = strings.TrimSpace(runtimeDir)
	return &Monitor{
		runtimeDir:     runtimeDir,
		stuckThreshold: defaultStuckThreshold,
		pollInterval:   defaultPollInterval,
		debounce:       defaultDebounce,
		observer:       NewTmuxObserver(runtimeDir),
		claims:         NewCanonicalClaimsReader(runtimeDir),
		tickets:        NewEventTicketMaterializer(runtimeDir),
		now:            time.Now,
	}
}

func (m *Monitor) RunOnce(ctx context.Context) ([]SessionMeta, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(m.runtimeDir) == "" {
		return nil, fmt.Errorf("runtime directory is required")
	}

	sessionIDs, err := listSessionIDs(m.runtimeDir)
	if err != nil {
		return nil, err
	}

	metas := make([]SessionMeta, 0, len(sessionIDs))
	now := m.now().UTC()
	for _, sessionID := range sessionIDs {
		observation, err := m.observer.Observe(sessionID)
		if err != nil {
			return nil, fmt.Errorf("observe session %s: %w", sessionID, err)
		}
		claims, err := m.claims.ReadSession(sessionID)
		if err != nil {
			return nil, fmt.Errorf("read claims for %s: %w", sessionID, err)
		}
		previous, err := readSessionMeta(sessionMetaPath(m.runtimeDir, sessionID))
		if err != nil {
			return nil, fmt.Errorf("read previous meta for %s: %w", sessionID, err)
		}

		meta := DeriveSessionMeta(sessionID, observation, claims, previous, now, m.stuckThreshold)
		if err := writeSessionMeta(sessionMetaPath(m.runtimeDir, sessionID), meta); err != nil {
			return nil, fmt.Errorf("write meta for %s: %w", sessionID, err)
		}
		metas = append(metas, meta)
	}

	if err := m.tickets.Materialize(ctx, sessionIDs); err != nil {
		return nil, fmt.Errorf("materialize tickets: %w", err)
	}

	m.mu.Lock()
	m.lastRunFinished = now
	m.mu.Unlock()
	return metas, nil
}

func (m *Monitor) Run(ctx context.Context) error {
	if strings.TrimSpace(m.runtimeDir) == "" {
		return fmt.Errorf("runtime directory is required")
	}
	sessionsPath := filepath.Join(m.runtimeDir, "sessions")
	if err := ensureSessionsPath(sessionsPath); err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create fsnotify watcher: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(sessionsPath); err != nil {
		return fmt.Errorf("watch sessions directory: %w", err)
	}
	watchedDirs := map[string]struct{}{sessionsPath: {}}
	if err := addSessionDirWatches(watcher, watchedDirs, sessionsPath); err != nil {
		return err
	}

	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	trigger := make(chan struct{}, 1)
	trigger <- struct{}{}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			enqueueTrigger(trigger)
		case event := <-watcher.Events:
			if event.Op&fsnotify.Create != 0 {
				if err := maybeAddSessionDirWatch(watcher, watchedDirs, event.Name); err != nil {
					return err
				}
			}
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) != 0 {
				enqueueTrigger(trigger)
			}
		case err := <-watcher.Errors:
			if err != nil {
				return fmt.Errorf("watch sessions directory: %w", err)
			}
		case <-trigger:
			if !m.shouldRunNow() {
				continue
			}
			if _, err := m.RunOnce(ctx); err != nil {
				return err
			}
		}
	}
}

func (m *Monitor) shouldRunNow() bool {
	if m.debounce <= 0 {
		return true
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lastRunFinished.IsZero() {
		return true
	}
	return m.now().UTC().Sub(m.lastRunFinished) >= m.debounce
}

func enqueueTrigger(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

func ensureSessionsPath(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create sessions directory: %w", err)
	}
	return nil
}

func addSessionDirWatches(
	watcher *fsnotify.Watcher,
	watchedDirs map[string]struct{},
	sessionsPath string,
) error {
	entries, err := os.ReadDir(sessionsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read sessions directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(sessionsPath, entry.Name())
		if _, ok := watchedDirs[path]; ok {
			continue
		}
		if err := watcher.Add(path); err != nil {
			return fmt.Errorf("watch session directory %s: %w", path, err)
		}
		watchedDirs[path] = struct{}{}
	}
	return nil
}

func maybeAddSessionDirWatch(
	watcher *fsnotify.Watcher,
	watchedDirs map[string]struct{},
	path string,
) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat watched path %s: %w", path, err)
	}
	if !info.IsDir() {
		return nil
	}
	if _, ok := watchedDirs[path]; ok {
		return nil
	}
	if err := watcher.Add(path); err != nil {
		return fmt.Errorf("watch session directory %s: %w", path, err)
	}
	watchedDirs[path] = struct{}{}
	return nil
}
