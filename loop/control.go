package loop

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/config"
	"github.com/poteto/noodle/dispatcher"
	"github.com/poteto/noodle/recover"
)

func (l *Loop) controlPaths() (controlPath string, ackPath string, lockPath string) {
	return filepath.Join(l.runtimeDir, "control.ndjson"),
		filepath.Join(l.runtimeDir, "control-ack.ndjson"),
		filepath.Join(l.runtimeDir, "control.lock")
}

func (l *Loop) hydrateProcessedCommands() error {
	_, ackPath, _ := l.controlPaths()
	file, err := os.Open(ackPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open control ack file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ack ControlAck
		if err := json.Unmarshal([]byte(line), &ack); err != nil {
			continue
		}
		if strings.TrimSpace(ack.ID) != "" {
			l.processedIDs[ack.ID] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan control ack file: %w", err)
	}
	return nil
}

func (l *Loop) processControlCommands() error {
	controlPath, ackPath, lockPath := l.controlPaths()
	if err := os.MkdirAll(l.runtimeDir, 0o755); err != nil {
		return fmt.Errorf("create runtime directory: %w", err)
	}
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open control lock: %w", err)
	}
	defer lock.Close()

	if err := acquireFileLock(lock.Fd()); err != nil {
		return fmt.Errorf("lock control file: %w", err)
	}
	defer func() {
		_ = releaseFileLock(lock.Fd())
	}()

	data, err := os.ReadFile(controlPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read control file: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	acks := make([]ControlAck, 0, len(lines))
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		var cmd ControlCommand
		if err := json.Unmarshal([]byte(line), &cmd); err != nil {
			acks = append(acks, ControlAck{
				ID:      "",
				Action:  "unknown",
				Status:  "error",
				Message: "invalid command JSON",
				At:      l.deps.Now(),
			})
			continue
		}
		if cmd.ID == "" {
			cmd.ID = fmt.Sprintf("cmd-%d", l.deps.Now().UnixNano())
		}
		if _, seen := l.processedIDs[cmd.ID]; seen {
			acks = append(acks, ControlAck{ID: cmd.ID, Action: cmd.Action, Status: "ok", At: l.deps.Now()})
			continue
		}
		ack := l.applyControlCommand(cmd)
		acks = append(acks, ack)
		l.processedIDs[cmd.ID] = struct{}{}
	}
	if len(acks) > 0 {
		if err := appendAcks(ackPath, acks); err != nil {
			return err
		}
	}
	if err := os.WriteFile(controlPath, []byte{}, 0o644); err != nil {
		return fmt.Errorf("truncate control file: %w", err)
	}
	return nil
}

func appendAcks(path string, acks []ControlAck) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create ack parent directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open ack file: %w", err)
	}
	defer file.Close()

	for _, ack := range acks {
		line, err := json.Marshal(ack)
		if err != nil {
			return fmt.Errorf("encode ack: %w", err)
		}
		if _, err := file.Write(append(line, '\n')); err != nil {
			return fmt.Errorf("append ack: %w", err)
		}
	}
	return nil
}

func (l *Loop) applyControlCommand(cmd ControlCommand) ControlAck {
	ack := ControlAck{ID: cmd.ID, Action: cmd.Action, Status: "ok", At: l.deps.Now()}
	switch strings.ToLower(strings.TrimSpace(cmd.Action)) {
	case "pause":
		l.state = StatePaused
	case "resume":
		l.state = StateRunning
	case "drain":
		l.state = StateDraining
	case "skip":
		if err := l.skipQueueItem(cmd.Item); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	case "kill":
		if err := l.killCook(cmd.Name); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	case "steer":
		if err := l.steer(cmd.Target, cmd.Prompt); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	case "merge":
		if err := l.controlMerge(cmd.Item); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	case "reject":
		if err := l.controlReject(cmd.Item); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	case "request-changes":
		if err := l.controlRequestChanges(cmd.Item, cmd.Prompt); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	case "autonomy":
		if err := l.controlAutonomy(cmd.Value); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	case "enqueue":
		if err := l.controlEnqueue(cmd); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	case "stop-all":
		l.controlStopAll()
	case "requeue":
		if err := l.controlRequeue(cmd.Item); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	case "edit-item":
		if err := l.controlEditItem(cmd); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	default:
		ack.Status = "error"
		ack.Message = "unsupported action"
	}
	return ack
}

func (l *Loop) controlMerge(itemID string) error {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return fmt.Errorf("merge: item ID empty")
	}
	pending, ok := l.pendingReview[itemID]
	if !ok {
		return fmt.Errorf("no pending review for %q", itemID)
	}
	if err := l.mergeCook(context.Background(), pending.queueItem, pending.worktreeName, pending.sessionID); err != nil {
		return err
	}
	delete(l.pendingReview, itemID)
	return l.writePendingReview()
}

func (l *Loop) controlReject(itemID string) error {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return fmt.Errorf("reject: item ID empty")
	}
	pending, ok := l.pendingReview[itemID]
	if !ok {
		return fmt.Errorf("no pending review for %q", itemID)
	}
	if strings.TrimSpace(pending.worktreeName) != "" {
		_ = l.deps.Worktree.Cleanup(pending.worktreeName, true)
	}
	if err := l.markFailed(itemID, "rejected by user"); err != nil {
		return err
	}
	if err := l.skipQueueItem(itemID); err != nil {
		return err
	}
	delete(l.pendingReview, itemID)
	return l.writePendingReview()
}

func (l *Loop) controlRequestChanges(itemID, feedback string) error {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return fmt.Errorf("request-changes: item ID empty")
	}
	pending, ok := l.pendingReview[itemID]
	if !ok {
		return fmt.Errorf("no pending review for %q", itemID)
	}

	resumePrompt := "Previous work needs changes."
	trimmedFeedback := strings.TrimSpace(feedback)
	if trimmedFeedback != "" {
		resumePrompt += " Feedback: " + trimmedFeedback
	}

	name := strings.TrimSpace(pending.worktreeName)
	if name == "" {
		name = cookBaseName(pending.queueItem)
	}
	path := strings.TrimSpace(pending.worktreePath)
	if path == "" {
		path = l.worktreePath(name)
	}

	taskType, _ := l.registry.ByKey(pending.queueItem.TaskKey)
	req := dispatcher.DispatchRequest{
		Name:         name,
		Prompt:       buildCookPrompt(pending.queueItem, resumePrompt),
		Provider:     nonEmpty(pending.queueItem.Provider, l.config.Routing.Defaults.Provider),
		Model:        nonEmpty(pending.queueItem.Model, l.config.Routing.Defaults.Model),
		Skill:        pending.queueItem.Skill,
		WorktreePath: path,
		TaskKey:      taskType.Key,
		Runtime:      nonEmpty(pending.queueItem.Runtime, "tmux"),
	}
	if taskType.Key == "execute" {
		if adapter, exists := l.config.Adapters["backlog"]; exists {
			req.DomainSkill = adapter.Skill
		}
	}
	session, err := l.deps.Dispatcher.Dispatch(context.Background(), req)
	if err != nil {
		return err
	}
	l.activeByTarget[itemID] = &activeCook{
		queueItem:    pending.queueItem,
		session:      session,
		worktreeName: name,
		worktreePath: path,
		attempt:      recover.RecoveryChainLength(name),
	}
	l.activeByID[session.ID()] = l.activeByTarget[itemID]

	delete(l.pendingReview, itemID)
	return l.writePendingReview()
}

func (l *Loop) controlAutonomy(value string) error {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case config.AutonomyAuto, config.AutonomyApprove:
		l.config.Autonomy = value
		return nil
	default:
		return fmt.Errorf("unsupported autonomy value %q", value)
	}
}

func (l *Loop) controlEnqueue(cmd ControlCommand) error {
	item := strings.TrimSpace(cmd.Item)
	if item == "" {
		return fmt.Errorf("enqueue requires item")
	}
	queue, err := readQueue(l.deps.QueueFile)
	if err != nil {
		return err
	}
	prompt := strings.TrimSpace(cmd.Prompt)
	queue.Items = append(queue.Items, QueueItem{
		ID:       item,
		Title:    titleFromPrompt(prompt, 8),
		Prompt:   prompt,
		TaskKey:  strings.TrimSpace(cmd.TaskKey),
		Provider: strings.TrimSpace(cmd.Provider),
		Model:    strings.TrimSpace(cmd.Model),
		Skill:    strings.TrimSpace(cmd.Skill),
	})
	return writeQueueAtomic(l.deps.QueueFile, queue)
}

func (l *Loop) controlEditItem(cmd ControlCommand) error {
	itemID := strings.TrimSpace(cmd.Item)
	if itemID == "" {
		return fmt.Errorf("edit-item requires item")
	}
	if _, active := l.activeByTarget[itemID]; active {
		return fmt.Errorf("item %q is currently cooking", itemID)
	}
	queue, err := readQueue(l.deps.QueueFile)
	if err != nil {
		return err
	}
	found := false
	for i := range queue.Items {
		if queue.Items[i].ID != itemID {
			continue
		}
		found = true
		prompt := strings.TrimSpace(cmd.Prompt)
		queue.Items[i].Title = titleFromPrompt(prompt, 8)
		queue.Items[i].Prompt = prompt
		queue.Items[i].TaskKey = strings.TrimSpace(cmd.TaskKey)
		queue.Items[i].Provider = strings.TrimSpace(cmd.Provider)
		queue.Items[i].Model = strings.TrimSpace(cmd.Model)
		queue.Items[i].Skill = strings.TrimSpace(cmd.Skill)
		break
	}
	if !found {
		return fmt.Errorf("item %q not found in queue", itemID)
	}
	return writeQueueAtomic(l.deps.QueueFile, queue)
}

func (l *Loop) controlStopAll() {
	for _, cook := range l.activeByID {
		_ = cook.session.Kill()
	}
}

func (l *Loop) controlRequeue(itemID string) error {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return fmt.Errorf("requeue requires item")
	}
	if _, ok := l.failedTargets[itemID]; !ok {
		return fmt.Errorf("item %q not in failed state", itemID)
	}
	delete(l.failedTargets, itemID)
	return l.writeFailedTargets()
}
