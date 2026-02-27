package loop

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/poteto/noodle/config"
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
		l.setState(StatePaused)
	case "resume":
		l.setState(StateRunning)
	case "drain":
		l.setState(StateDraining)
	case "skip":
		if err := l.controlSkip(cmd.OrderID); err != nil {
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
		if err := l.controlMerge(cmd.OrderID); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	case "reject":
		if err := l.controlReject(cmd.OrderID); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	case "request-changes":
		if err := l.controlRequestChanges(cmd.OrderID, cmd.Prompt); err != nil {
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
		if err := l.controlRequeue(cmd.OrderID); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	case "edit-item":
		if err := l.controlEditItem(cmd); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	case "reorder":
		if err := l.controlReorder(cmd); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	case "stop":
		if err := l.controlStop(cmd.Name); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	case "set-max-cooks":
		if err := l.controlSetMaxCooks(cmd.Value); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	default:
		ack.Status = "error"
		ack.Message = "unsupported action"
	}
	if ack.Status == "error" {
		l.logger.Warn("control command failed", "action", cmd.Action, "message", ack.Message)
	} else {
		l.logger.Info("control command", "action", cmd.Action, "status", ack.Status)
	}
	return ack
}

func (l *Loop) controlMerge(orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("merge: order ID empty")
	}
	pending, ok := l.pendingReview[orderID]
	if !ok {
		return fmt.Errorf("no pending review for %q", orderID)
	}

	// Quality verdict gate — even the manual merge path respects quality verdicts.
	verdict, hasVerdict := l.readQualityVerdict(pending.sessionID)
	if hasVerdict && !verdict.Accept {
		reason := "quality rejected: " + verdict.Feedback
		orders, terminal, err := failStage(l.orders, orderID, reason)
		if err != nil {
			return err
		}
		l.orders = orders
		l.markOrdersDirty()
		if strings.TrimSpace(pending.worktreeName) != "" {
			_ = l.deps.Worktree.Cleanup(pending.worktreeName, true)
		}
		if terminal {
			if err := l.markFailed(orderID, reason); err != nil {
				return err
			}
		}
		delete(l.pendingReview, orderID)
		return l.writePendingReview()
	}

	// Merge the worktree.
	sess := &adoptedSession{id: pending.sessionID, status: "completed"}
	cook := &cookHandle{
		orderID:      pending.orderID,
		stageIndex:   pending.stageIndex,
		stage:        pending.stage,
		isOnFailure:  false,
		orderStatus:  OrderStatusActive,
		plan:         pending.plan,
		worktreeName: pending.worktreeName,
		worktreePath: pending.worktreePath,
		session:      sess,
		done:         sess.Done(),
	}
	// Determine actual order status for advanceAndPersist.
	for _, o := range l.orders.Orders {
		if o.ID == orderID {
			cook.orderStatus = o.Status
			cook.isOnFailure = o.Status == OrderStatusFailing
			break
		}
	}
	if err := l.mergeCookWorktree(context.Background(), cook); err != nil {
		return err
	}
	if err := l.advanceAndPersist(context.Background(), cook); err != nil {
		return err
	}
	delete(l.pendingReview, orderID)
	return l.writePendingReview()
}

func (l *Loop) controlReject(orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("reject: order ID empty")
	}
	pending, ok := l.pendingReview[orderID]
	if !ok {
		return fmt.Errorf("no pending review for %q", orderID)
	}
	if strings.TrimSpace(pending.worktreeName) != "" {
		_ = l.deps.Worktree.Cleanup(pending.worktreeName, true)
	}
	// User rejection skips OnFailure — cancel and remove the order directly.
	orders, err := cancelOrder(l.orders, orderID)
	if err != nil {
		l.logger.Warn("controlReject: cancelOrder", "error", err)
	} else {
		l.orders = orders
		l.markOrdersDirty()
	}
	if err := l.markFailed(orderID, "rejected by user"); err != nil {
		return err
	}
	delete(l.pendingReview, orderID)
	return l.writePendingReview()
}

func (l *Loop) controlRequestChanges(orderID, feedback string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("request-changes: order ID empty")
	}
	pending, ok := l.pendingReview[orderID]
	if !ok {
		return fmt.Errorf("no pending review for %q", orderID)
	}
	if l.atMaxConcurrency() {
		l.logger.Info("request-changes deferred: at max concurrency", "order", orderID)
		return nil
	}

	reason := "changes requested"
	trimmedFeedback := strings.TrimSpace(feedback)
	if trimmedFeedback != "" {
		reason += ": " + trimmedFeedback
	}
	orders, terminal, err := failStage(l.orders, orderID, reason)
	if err != nil {
		return err
	}
	l.orders = orders
	l.markOrdersDirty()
	if terminal {
		if err := l.markFailed(orderID, reason); err != nil {
			return err
		}
	}

	// Clean up the worktree for the failed stage.
	if strings.TrimSpace(pending.worktreeName) != "" {
		_ = l.deps.Worktree.Cleanup(pending.worktreeName, true)
	}

	delete(l.pendingReview, orderID)
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
	orderID := strings.TrimSpace(cmd.OrderID)
	if orderID == "" {
		return fmt.Errorf("enqueue requires order_id")
	}
	prompt := strings.TrimSpace(cmd.Prompt)
	taskKey := strings.TrimSpace(cmd.TaskKey)
	if taskKey == "" {
		taskKey = "execute"
	}

	newOrder := Order{
		ID:    orderID,
		Title: titleFromPrompt(prompt, 8),
		Status: OrderStatusActive,
		Stages: []Stage{{
			TaskKey:  taskKey,
			Prompt:   prompt,
			Skill:    strings.TrimSpace(cmd.Skill),
			Provider: strings.TrimSpace(cmd.Provider),
			Model:    strings.TrimSpace(cmd.Model),
			Status:   StageStatusPending,
		}},
	}
	l.orders.Orders = append(l.orders.Orders, newOrder)
	l.markOrdersDirty()
	return nil
}

func (l *Loop) controlEditItem(cmd ControlCommand) error {
	orderID := strings.TrimSpace(cmd.OrderID)
	if orderID == "" {
		return fmt.Errorf("edit-item requires order_id")
	}
	if _, active := l.activeCooksByOrder[orderID]; active {
		return fmt.Errorf("order %q is currently cooking", orderID)
	}
	found := false
	for i := range l.orders.Orders {
		if l.orders.Orders[i].ID != orderID {
			continue
		}
		found = true
		// Edit order-level fields.
		if title := strings.TrimSpace(cmd.Prompt); title != "" {
			l.orders.Orders[i].Title = titleFromPrompt(title, 8)
		}
		stageIdx, stage := activeStageForOrder(l.orders.Orders[i])
		if stageIdx < 0 || stage == nil {
			return fmt.Errorf("order %q has no editable stage", orderID)
		}
		stages := &l.orders.Orders[i].Stages
		if l.orders.Orders[i].Status == OrderStatusFailing {
			stages = &l.orders.Orders[i].OnFailure
		}
		if prompt := strings.TrimSpace(cmd.Prompt); prompt != "" {
			(*stages)[stageIdx].Prompt = prompt
		}
		if taskKey := strings.TrimSpace(cmd.TaskKey); taskKey != "" {
			(*stages)[stageIdx].TaskKey = taskKey
		}
		if provider := strings.TrimSpace(cmd.Provider); provider != "" {
			(*stages)[stageIdx].Provider = provider
		}
		if model := strings.TrimSpace(cmd.Model); model != "" {
			(*stages)[stageIdx].Model = model
		}
		if skill := strings.TrimSpace(cmd.Skill); skill != "" {
			(*stages)[stageIdx].Skill = skill
		}
		break
	}
	if !found {
		return fmt.Errorf("order %q not found", orderID)
	}
	l.markOrdersDirty()
	return nil
}

func (l *Loop) controlStopAll() {
	for _, cook := range l.activeCooksByOrder {
		_ = cook.session.Kill()
	}
}

func (l *Loop) controlSkip(orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("skip requires order_id")
	}
	orders, err := cancelOrder(l.orders, orderID)
	if err != nil {
		return err
	}
	l.orders = orders
	l.markOrdersDirty()
	return nil
}

func (l *Loop) controlRequeue(orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return fmt.Errorf("requeue requires order_id")
	}
	if _, ok := l.failedTargets[orderID]; !ok {
		return fmt.Errorf("order %q not in failed state", orderID)
	}

	for i := range l.orders.Orders {
		if l.orders.Orders[i].ID != orderID {
			continue
		}
		l.orders.Orders[i].Status = OrderStatusActive
		resetStages(&l.orders.Orders[i].Stages)
		resetStages(&l.orders.Orders[i].OnFailure)
		l.markOrdersDirty()
		break
	}
	delete(l.failedTargets, orderID)
	return l.writeFailedTargets()
}

// resetStages resets all failed/cancelled stages to pending.
func resetStages(stages *[]Stage) {
	for i := range *stages {
		switch (*stages)[i].Status {
		case StageStatusFailed, StageStatusCancelled:
			(*stages)[i].Status = StageStatusPending
		}
	}
}

func (l *Loop) controlReorder(cmd ControlCommand) error {
	orderID := strings.TrimSpace(cmd.OrderID)
	if orderID == "" {
		return fmt.Errorf("reorder requires order_id")
	}
	newIndex := 0
	if v := strings.TrimSpace(cmd.Value); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("reorder: invalid index %q", v)
		}
		newIndex = n
	}
	srcIdx := -1
	for i := range l.orders.Orders {
		if l.orders.Orders[i].ID == orderID {
			srcIdx = i
			break
		}
	}
	if srcIdx < 0 {
		return fmt.Errorf("order %q not found", orderID)
	}
	order := l.orders.Orders[srcIdx]
	l.orders.Orders = append(l.orders.Orders[:srcIdx], l.orders.Orders[srcIdx+1:]...)
	if newIndex < 0 {
		newIndex = 0
	}
	if newIndex > len(l.orders.Orders) {
		newIndex = len(l.orders.Orders)
	}
	l.orders.Orders = append(l.orders.Orders[:newIndex], append([]Order{order}, l.orders.Orders[newIndex:]...)...)
	l.markOrdersDirty()
	return nil
}

func (l *Loop) controlStop(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("stop requires name")
	}
	for orderID, cook := range l.activeCooksByOrder {
		if cook.worktreeName == name || cook.session.ID() == name {
			_ = cook.session.Kill()
			delete(l.activeCooksByOrder, orderID)
			if cook.worktreeName != "" {
				_ = l.deps.Worktree.Cleanup(cook.worktreeName, true)
			}
			return nil
		}
	}
	return fmt.Errorf("session not found")
}

func (l *Loop) controlSetMaxCooks(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("set-max-cooks requires value")
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("set-max-cooks: invalid value %q", value)
	}
	if n < 1 {
		return fmt.Errorf("max_cooks must be at least 1")
	}
	l.config.Concurrency.MaxCooks = n
	return nil
}
