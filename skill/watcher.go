package skill

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// SkillWatcher watches configured skill paths for filesystem changes and
// triggers a callback when skills are added, modified, or deleted.
type SkillWatcher struct {
	watcher    *fsnotify.Watcher
	onChange   func()
	debounce   time.Duration
	mu         sync.Mutex
	watchedDirs map[string]struct{}
}

// NewSkillWatcher creates a watcher for the given skill search paths.
// For each path: if the directory exists, add a watch and scan for
// subdirectories (each skill is a subdirectory with a SKILL.md).
// If a path doesn't exist yet, watch the nearest existing parent
// directory for Create events.
func NewSkillWatcher(paths []string, onChange func()) (*SkillWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	sw := &SkillWatcher{
		watcher:     w,
		onChange:     onChange,
		debounce:    200 * time.Millisecond,
		watchedDirs: map[string]struct{}{},
	}

	for _, raw := range paths {
		resolved, ok := resolveSearchPath(raw)
		if !ok {
			continue
		}
		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			// Path doesn't exist yet — watch nearest existing parent.
			sw.watchNearestParent(resolved)
			continue
		}
		sw.addDirWatch(resolved)
		sw.addSubdirWatches(resolved)
	}

	return sw, nil
}

// Run processes filesystem events until ctx is cancelled.
func (sw *SkillWatcher) Run(ctx context.Context) {
	var timer *time.Timer
	var timerC <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return
		case event, ok := <-sw.watcher.Events:
			if !ok {
				return
			}
			// Filter Chmod-only events (editor noise).
			if event.Op == fsnotify.Chmod {
				continue
			}
			// On Create for directories: add new watch.
			if event.Op&fsnotify.Create != 0 {
				sw.maybeAddDirWatch(event.Name)
			}
			// On Remove/Rename of directories: remove stale watch.
			if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
				sw.maybeRemoveDirWatch(event.Name)
			}
			// Debounce: reset timer on any qualifying event.
			if timer == nil {
				timer = time.NewTimer(sw.debounce)
				timerC = timer.C
			} else {
				timer.Reset(sw.debounce)
			}
		case <-timerC:
			sw.onChange()
			timer = nil
			timerC = nil
		case _, ok := <-sw.watcher.Errors:
			if !ok {
				return
			}
			// Log errors but don't crash — skill watching is best-effort.
		}
	}
}

// Close shuts down the underlying fsnotify watcher.
func (sw *SkillWatcher) Close() error {
	return sw.watcher.Close()
}

func (sw *SkillWatcher) addDirWatch(path string) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	if _, ok := sw.watchedDirs[path]; ok {
		return
	}
	if err := sw.watcher.Add(path); err != nil {
		return
	}
	sw.watchedDirs[path] = struct{}{}
}

func (sw *SkillWatcher) addSubdirWatches(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sw.addDirWatch(filepath.Join(dir, entry.Name()))
	}
}

func (sw *SkillWatcher) maybeAddDirWatch(path string) {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return
	}
	sw.addDirWatch(path)
}

func (sw *SkillWatcher) maybeRemoveDirWatch(path string) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	if _, ok := sw.watchedDirs[path]; !ok {
		return
	}
	_ = sw.watcher.Remove(path)
	delete(sw.watchedDirs, path)
}

func (sw *SkillWatcher) watchNearestParent(path string) {
	for {
		parent := filepath.Dir(path)
		if parent == path {
			// Reached filesystem root without finding an existing dir.
			return
		}
		info, err := os.Stat(parent)
		if err == nil && info.IsDir() {
			sw.addDirWatch(parent)
			return
		}
		path = parent
	}
}
