package loop

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/taskreg"
)

// QueueAuditEvent records a queue or registry change for TUI consumption.
type QueueAuditEvent struct {
	At      time.Time `json:"at"`
	Type    string    `json:"type"`
	Target  string    `json:"target,omitempty"`
	Skill   string    `json:"skill,omitempty"`
	Reason  string    `json:"reason,omitempty"`
	Added   []string  `json:"added,omitempty"`
	Removed []string  `json:"removed,omitempty"`
}

// RegistryDiff captures what changed between two registry snapshots.
type RegistryDiff struct {
	Added   []string
	Removed []string
}

// diffRegistryKeys compares old and new registry key sets.
func diffRegistryKeys(old, new taskreg.Registry) RegistryDiff {
	oldKeys := make(map[string]struct{})
	for _, tt := range old.All() {
		oldKeys[tt.Key] = struct{}{}
	}
	newKeys := make(map[string]struct{})
	for _, tt := range new.All() {
		newKeys[tt.Key] = struct{}{}
	}

	var added []string
	for k := range newKeys {
		if _, ok := oldKeys[k]; !ok {
			added = append(added, k)
		}
	}
	var removed []string
	for k := range oldKeys {
		if _, ok := newKeys[k]; !ok {
			removed = append(removed, k)
		}
	}
	return RegistryDiff{Added: added, Removed: removed}
}

// auditQueue checks each queue item against the current registry.
// Items referencing unknown task types are removed. Returns dropped items.
func (l *Loop) auditQueue() []QueueItem {
	queue, err := readQueue(l.deps.QueueFile)
	if err != nil {
		// If we can't read the queue, skip the audit silently.
		return nil
	}

	var kept []QueueItem
	var dropped []QueueItem
	for _, item := range queue.Items {
		input := taskreg.QueueItemInput{
			ID:      item.ID,
			TaskKey: item.TaskKey,
			Title:   item.Title,
			Skill:   item.Skill,
		}
		if _, ok := l.registry.ResolveQueueItem(input); ok {
			kept = append(kept, item)
		} else {
			dropped = append(dropped, item)
			skillName := item.Skill
			if skillName == "" {
				skillName = item.TaskKey
			}
			fmt.Fprintf(os.Stderr, "dropped queue item %q: skill %q no longer exists\n", item.ID, skillName)
		}
	}

	if len(dropped) == 0 {
		return nil
	}

	// Write back the filtered queue.
	queue.Items = kept
	if err := writeQueueAtomic(l.deps.QueueFile, queue); err != nil {
		fmt.Fprintf(os.Stderr, "queue-audit: write queue: %v\n", err)
		return dropped
	}

	// Write drop events.
	eventsPath := filepath.Join(l.runtimeDir, "queue-events.ndjson")
	now := l.deps.Now().UTC()
	for _, item := range dropped {
		skillName := item.Skill
		if skillName == "" {
			skillName = item.TaskKey
		}
		event := QueueAuditEvent{
			At:     now,
			Type:   "queue_drop",
			Target: item.ID,
			Skill:  skillName,
			Reason: "skill no longer exists",
		}
		appendQueueEvent(eventsPath, event)
	}

	return dropped
}

const maxQueueEventLines = 200

// appendQueueEvent marshals event as JSON and appends to an NDJSON file.
// Truncates to the last 200 lines if the file exceeds that.
func appendQueueEvent(path string, event QueueAuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, _ = f.Write(append(data, '\n'))
	_ = f.Close()

	truncateQueueEvents(path)
}

// truncateQueueEvents keeps only the last maxQueueEventLines lines.
func truncateQueueEvents(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	_ = f.Close()

	if len(lines) <= maxQueueEventLines {
		return
	}

	// Keep only the last maxQueueEventLines lines.
	lines = lines[len(lines)-maxQueueEventLines:]
	var buf strings.Builder
	for _, line := range lines {
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	_ = os.WriteFile(path, []byte(buf.String()), 0o644)
}
