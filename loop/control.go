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
	"time"

	"github.com/poteto/noodle/internal/filex"
)

func (l *Loop) controlPaths() (controlPath string, ackPath string, lockPath string) {
	return filepath.Join(l.runtimeDir, "control.ndjson"),
		filepath.Join(l.runtimeDir, "control-ack.ndjson"),
		filepath.Join(l.runtimeDir, "control.lock")
}

func (l *Loop) lastAppliedSeqPath() string {
	return filepath.Join(l.runtimeDir, "last-applied-seq")
}

func (l *Loop) hydrateProcessedCommands() error {
	_, ackPath, _ := l.controlPaths()
	file, err := os.Open(ackPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("open control ack file: %w", err)
		}
	} else {
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
				l.cmds.processedIDs[ack.ID] = struct{}{}
			}
		}
		if err := scanner.Err(); err != nil {
			file.Close()
			return fmt.Errorf("scan control ack file: %w", err)
		}
		file.Close()
	}

	// Load last-applied sequence number.
	seqData, err := os.ReadFile(l.lastAppliedSeqPath())
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read last-applied-seq: %w", err)
		}
		return nil
	}
	seq, err := strconv.ParseUint(strings.TrimSpace(string(seqData)), 10, 64)
	if err != nil {
		// Corrupt file — start from zero.
		return nil
	}
	l.cmds.lastAppliedSeq = seq
	l.cmds.cmdSeqCounter = seq
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

		// Assign a monotonic sequence number.
		l.cmds.cmdSeqCounter++
		seq := l.cmds.cmdSeqCounter

		// Skip commands already applied (ID-based dedup or sequence-based).
		if _, seen := l.cmds.processedIDs[cmd.ID]; seen {
			acks = append(acks, ControlAck{ID: cmd.ID, Action: cmd.Action, Status: "ok", At: l.deps.Now()})
			continue
		}
		if seq <= l.cmds.lastAppliedSeq {
			acks = append(acks, ControlAck{ID: cmd.ID, Action: cmd.Action, Status: "ok", At: l.deps.Now()})
			l.cmds.processedIDs[cmd.ID] = struct{}{}
			continue
		}

		ack := l.applyControlCommand(cmd)
		acks = append(acks, ack)
		l.cmds.processedIDs[cmd.ID] = struct{}{}
		l.cmds.lastAppliedSeq = seq
	}
	if l.TestControlAckBarrier != nil {
		l.TestControlAckBarrier()
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
	case "advance":
		if err := l.controlAdvance(cmd.OrderID); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	case "add-stage":
		if err := l.controlAddStage(cmd); err != nil {
			ack.Status = "error"
			ack.Message = err.Error()
		}
	case "park-review":
		if err := l.controlParkReview(cmd.OrderID, cmd.Prompt); err != nil {
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

func (l *Loop) controlStopAll() {
	for _, cook := range l.cooks.activeCooksByOrder {
		_ = cook.session.Kill()
	}
}

func (l *Loop) controlStop(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("stop requires name")
	}
	for _, cook := range l.cooks.activeCooksByOrder {
		if cook.worktreeName != name && cook.session.ID() != name {
			continue
		}
		controller := cook.session.Controller()
		if !controller.Steerable() {
			return l.controlStopKill(cook)
		}
		// Interrupt gracefully — the session stays alive.
		sessionID := cook.session.ID()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := controller.Interrupt(ctx); err != nil {
				l.logger.Warn("stop interrupt failed",
					"session", sessionID, "error", err)
			}
		}()
		return nil
	}
	return fmt.Errorf("session not found")
}

// controlStopKill is the fallback for non-steerable sessions: kill the process
// and clean up the cook.
func (l *Loop) controlStopKill(cook *cookHandle) error {
	_ = cook.session.Kill()
	l.trackCookCompleted(cook, StageResult{
		SessionID:   cook.session.ID(),
		Status:      StageResultCancelled,
		CompletedAt: l.deps.Now(),
	})
	delete(l.cooks.activeCooksByOrder, cook.orderID)
	if cook.worktreeName != "" {
		_ = l.deps.Worktree.Cleanup(cook.worktreeName, true)
	}
	return nil
}

func (l *Loop) writeLastAppliedSeq() error {
	if l.cmds.lastAppliedSeq == 0 {
		return nil
	}
	data := []byte(strconv.FormatUint(l.cmds.lastAppliedSeq, 10) + "\n")
	return filex.WriteFileAtomic(l.lastAppliedSeqPath(), data)
}
