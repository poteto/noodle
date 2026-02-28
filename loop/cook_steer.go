package loop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/poteto/noodle/event"
)

func (l *Loop) killCook(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("kill requires name")
	}
	for _, cook := range l.cooks.activeCooksByOrder {
		if cook.worktreeName == name || cook.session.ID() == name {
			return cook.session.Kill()
		}
	}
	return fmt.Errorf("session not found")
}

// steerMu serializes concurrent steers to the same session.
var steerMu sync.Map // session ID → *sync.Mutex

func sessionSteerMutex(sessionID string) *sync.Mutex {
	val, _ := steerMu.LoadOrStore(sessionID, &sync.Mutex{})
	return val.(*sync.Mutex)
}

func (l *Loop) steer(target string, prompt string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("steer requires target")
	}
	if strings.EqualFold(target, ScheduleTaskKey()) {
		return l.rescheduleForChefPrompt(prompt)
	}
	for _, cook := range l.cooks.activeCooksByOrder {
		if cook.worktreeName != target && cook.session.ID() != target {
			continue
		}

		controller := cook.session.Controller()
		if controller.Steerable() {
			// Live steering — interrupt + redirect without killing the session.
			// Run async so the control loop isn't blocked.
			sessionID := cook.session.ID()
			steerPrompt := strings.TrimSpace(prompt)
			go l.steerLive(sessionID, controller, steerPrompt)
			return nil
		}

		// Not steerable — fall back to kill + respawn with resume context.
		return l.steerRespawn(cook, prompt)
	}
	return errors.New("session not found")
}

// steerLive interrupts a steerable session and sends a new prompt.
// Runs in a goroutine. Falls back to kill+respawn if interrupt fails.
func (l *Loop) steerLive(sessionID string, controller interface {
	Interrupt(ctx context.Context) error
	SendMessage(ctx context.Context, prompt string) error
}, prompt string) {
	mu := sessionSteerMutex(sessionID)
	mu.Lock()
	defer mu.Unlock()
	defer steerMu.Delete(sessionID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := controller.Interrupt(ctx); err != nil {
		l.logger.Warn("steer interrupt failed, falling back to respawn",
			"session", sessionID, "error", err)
		l.steerFallbackRespawn(sessionID, prompt)
		return
	}

	if err := controller.SendMessage(ctx, prompt); err != nil {
		l.logger.Warn("steer send failed after interrupt",
			"session", sessionID, "error", err)
		return
	}

	l.logger.Info("steered session", "session", sessionID)
}

// steerFallbackRespawn is called from a goroutine when live interrupt fails.
// It performs the same kill+respawn as steerRespawn but must look up the cook
// again since it runs asynchronously.
func (l *Loop) steerFallbackRespawn(sessionID string, prompt string) {
	// The loop state is not protected by a mutex — steer fallback cannot
	// safely mutate cook maps from a goroutine. Log the failure so the
	// operator can manually re-steer or kill+enqueue.
	l.logger.Warn("steer fallback: live interrupt failed, session may need manual re-steer",
		"session", sessionID)
}

// steerRespawn is the original kill+respawn steer path for non-steerable sessions.
func (l *Loop) steerRespawn(cook *cookHandle, prompt string) error {
	resumeCtx := buildSteerResumeContext(l.runtimeDir, cook.session.ID())
	steerPrompt := strings.TrimSpace(prompt)
	if resumeCtx != "" {
		steerPrompt = "Resume context: " + resumeCtx + "\n\nChef steering: " + steerPrompt
	}

	if err := cook.session.Kill(); err != nil {
		return err
	}
	l.trackCookCompleted(cook, StageResult{
		SessionID:   cook.session.ID(),
		Status:      StageResultCancelled,
		CompletedAt: l.deps.Now(),
	})
	delete(l.cooks.activeCooksByOrder, cook.orderID)
	cand := dispatchCandidate{
		OrderID:    cook.orderID,
		StageIndex: cook.stageIndex,
		Stage:      cook.stage,
	}
	order := Order{
		ID:     cook.orderID,
		Status: cook.orderStatus,
		Plan:   cook.plan,
	}
	return l.spawnCook(context.Background(), cand, order, spawnOptions{
		attempt:     cook.attempt,
		resume:      steerPrompt,
		displayName: cook.displayName,
	})
}

// buildSteerResumeContext reads a session's event log and extracts a progress
// summary so the respawned session doesn't start from scratch.
func buildSteerResumeContext(runtimeDir string, sessionID string) string {
	reader := event.NewEventReader(runtimeDir)
	events, err := reader.ReadSession(sessionID, event.EventFilter{})
	if err != nil || len(events) == 0 {
		return ""
	}

	files := make(map[string]struct{})
	var lastActions []string
	var ticketProgress []string

	for _, ev := range events {
		switch ev.Type {
		case event.EventAction:
			var action struct {
				Tool    string `json:"tool"`
				Path    string `json:"path"`
				Summary string `json:"summary"`
			}
			_ = json.Unmarshal(ev.Payload, &action)
			tool := strings.ToLower(strings.TrimSpace(action.Tool))
			if path := strings.TrimSpace(action.Path); path != "" {
				switch tool {
				case "read", "edit", "write":
					files[path] = struct{}{}
				}
			}
			summary := strings.TrimSpace(action.Summary)
			if summary == "" {
				summary = strings.TrimSpace(action.Tool)
			}
			if summary != "" {
				lastActions = append(lastActions, summary)
			}
		case event.EventTicketProgress, event.EventTicketDone:
			var payload struct {
				Summary string `json:"summary"`
				Outcome string `json:"outcome"`
			}
			_ = json.Unmarshal(ev.Payload, &payload)
			if s := strings.TrimSpace(payload.Summary); s != "" {
				ticketProgress = append(ticketProgress, s)
			} else if s := strings.TrimSpace(payload.Outcome); s != "" {
				ticketProgress = append(ticketProgress, s)
			}
		}
	}

	var parts []string
	if len(files) > 0 {
		fileList := make([]string, 0, len(files))
		for f := range files {
			fileList = append(fileList, f)
		}
		if len(fileList) > 10 {
			fileList = fileList[:10]
		}
		parts = append(parts, fmt.Sprintf("Files touched: %s", strings.Join(fileList, ", ")))
	}
	if len(ticketProgress) > 0 {
		if len(ticketProgress) > 3 {
			ticketProgress = ticketProgress[len(ticketProgress)-3:]
		}
		parts = append(parts, fmt.Sprintf("Progress: %s", strings.Join(ticketProgress, "; ")))
	}
	if len(lastActions) > 0 {
		tail := lastActions
		if len(tail) > 5 {
			tail = tail[len(tail)-5:]
		}
		parts = append(parts, fmt.Sprintf("Recent actions: %s", strings.Join(tail, " → ")))
	}

	return strings.Join(parts, ". ")
}
