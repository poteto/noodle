package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	sprites "github.com/superfly/sprites-go"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/parse"
	"github.com/poteto/noodle/stamp"
)

type spritesSessionConfig struct {
	id            string
	sprite        spriteHandle
	spriteName    string
	cmd           *sprites.Cmd
	stdout        io.ReadCloser
	runtimeDir    string
	stampedPath   string
	canonicalPath string
	eventWriter   *event.EventWriter
	prompt        string
	warnings      []string
	remoteURL     string
}

type spritesSession struct {
	id         string
	sprite     spriteHandle
	spriteName string
	cmd        *sprites.Cmd
	stdout     io.ReadCloser
	runtimeDir string
	eventWriter *event.EventWriter
	remoteURL  string

	stampedPath   string
	canonicalPath string

	mu       sync.Mutex
	status   string
	costUSD  float64
	done     chan struct{}
	doneOnce sync.Once
	events   chan SessionEvent
	eventsMu sync.Once
	dropped  atomic.Uint64
	wg       sync.WaitGroup

	startWarnings []string
	prompt        string
	promptLogged  bool
}

func newSpritesSession(cfg spritesSessionConfig) *spritesSession {
	return &spritesSession{
		id:            cfg.id,
		sprite:        cfg.sprite,
		spriteName:    cfg.spriteName,
		cmd:           cfg.cmd,
		stdout:        cfg.stdout,
		runtimeDir:    cfg.runtimeDir,
		eventWriter:   cfg.eventWriter,
		remoteURL:     cfg.remoteURL,
		stampedPath:   cfg.stampedPath,
		canonicalPath: cfg.canonicalPath,
		status:        "running",
		done:          make(chan struct{}),
		events:        make(chan SessionEvent, 32),
		startWarnings: append([]string(nil), cfg.warnings...),
		prompt:        cfg.prompt,
	}
}

func (s *spritesSession) start(ctx context.Context) {
	s.wg.Add(2)

	// Goroutine 1: read stdout, parse events, emit.
	go func() {
		defer s.wg.Done()
		s.processStream(ctx)
	}()

	// Goroutine 2: wait for command exit, push changes, mark done.
	go func() {
		defer s.wg.Done()
		s.waitAndSync(ctx)
	}()

	go s.closeEventsWhenDone()

	for _, warning := range s.startWarnings {
		s.publish(SessionEvent{
			Type:      "warning",
			Message:   warning,
			Timestamp: nowUTC(),
		})
	}
}

func (s *spritesSession) processStream(ctx context.Context) {
	stampedFile, err := os.Create(s.stampedPath)
	if err != nil {
		s.publish(SessionEvent{Type: "warning", Message: "create stamped file: " + err.Error(), Timestamp: nowUTC()})
		return
	}
	defer stampedFile.Close()

	canonicalFile, err := os.Create(s.canonicalPath)
	if err != nil {
		s.publish(SessionEvent{Type: "warning", Message: "create canonical file: " + err.Error(), Timestamp: nowUTC()})
		return
	}
	defer canonicalFile.Close()

	processor := stamp.NewProcessor()

	// Use a custom writer to intercept canonical events for live publishing.
	eventInterceptor := &canonicalEventInterceptor{session: s}

	// Process reads from stdout, writes stamped to file, canonical events to interceptor.
	if err := processor.Process(ctx, s.stdout, stampedFile, io.MultiWriter(canonicalFile, eventInterceptor)); err != nil {
		if ctx.Err() == nil {
			s.publish(SessionEvent{Type: "warning", Message: "process stream: " + err.Error(), Timestamp: nowUTC()})
		}
	}
}

func (s *spritesSession) waitAndSync(ctx context.Context) {
	err := s.cmd.Wait()

	terminalStatus := "completed"
	if err != nil {
		var exitErr *sprites.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() != 0 {
				terminalStatus = "failed"
			}
		} else if ctx.Err() != nil {
			terminalStatus = "cancelled"
		} else {
			terminalStatus = "failed"
		}
	}

	// Push changes from sprite back to git remote.
	if s.remoteURL != "" && (terminalStatus == "completed" || terminalStatus == "failed") {
		syncCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		result, pushErr := pushChangesFromSprite(syncCtx, s.sprite, s.id)
		if pushErr != nil {
			s.publish(SessionEvent{
				Type:      "warning",
				Message:   "sync-back push failed: " + pushErr.Error(),
				Timestamp: nowUTC(),
			})
		} else if result.Type == SyncResultTypeBranch {
			if writeErr := writeSyncResult(s.runtimeDir, s.id, result); writeErr != nil {
				s.publish(SessionEvent{
					Type:      "warning",
					Message:   "write sync result: " + writeErr.Error(),
					Timestamp: nowUTC(),
				})
			}
		}
	}

	s.markDone(terminalStatus)
}

// canonicalEventInterceptor captures canonical event lines for live session event publishing.
type canonicalEventInterceptor struct {
	session *spritesSession
}

func (w *canonicalEventInterceptor) Write(p []byte) (int, error) {
	// Each write is one canonical event line (from stamp.Processor).
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		w.session.consumeCanonicalLine([]byte(line))
	}
	return len(p), nil
}

func (s *spritesSession) consumeCanonicalLine(line []byte) {
	var ce parse.CanonicalEvent
	if err := json.Unmarshal(line, &ce); err != nil {
		return
	}

	if ce.CostUSD > 0 {
		s.mu.Lock()
		s.costUSD += ce.CostUSD
		s.mu.Unlock()
	}

	message := strings.TrimSpace(ce.Message)
	if message == "" {
		message = string(ce.Type)
	}

	s.publish(SessionEvent{
		Type:      string(ce.Type),
		Message:   message,
		Timestamp: ce.Timestamp,
		CostUSD:   ce.CostUSD,
		TokensIn:  ce.TokensIn,
		TokensOut: ce.TokensOut,
	})

	// Write to event log (reuse tmuxSession's eventFromCanonical).
	if s.eventWriter != nil {
		if record, ok := eventFromCanonical(s.id, ce); ok {
			if err := s.eventWriter.Append(context.Background(), record); err != nil {
				s.publish(SessionEvent{
					Type:      "warning",
					Message:   "event log append failed: " + err.Error(),
					Timestamp: nowUTC(),
				})
			}
		}
	}

	if ce.Type == parse.EventInit {
		s.emitPromptEvent(ce.Timestamp)
	}
}

func (s *spritesSession) emitPromptEvent(timestamp time.Time) {
	if s.promptLogged {
		return
	}
	s.promptLogged = true
	prompt := strings.TrimSpace(s.prompt)
	if prompt == "" {
		return
	}
	if timestamp.IsZero() {
		timestamp = nowUTC()
	}
	s.publish(SessionEvent{
		Type:      "action",
		Message:   prompt,
		Timestamp: timestamp,
	})
}

func (s *spritesSession) ID() string      { return s.id }
func (s *spritesSession) Status() string   { s.mu.Lock(); defer s.mu.Unlock(); return s.status }
func (s *spritesSession) Events() <-chan SessionEvent { return s.events }
func (s *spritesSession) Done() <-chan struct{}       { return s.done }
func (s *spritesSession) TotalCost() float64 { s.mu.Lock(); defer s.mu.Unlock(); return s.costUSD }

func (s *spritesSession) Kill() error {
	if s.cmd != nil {
		_ = s.cmd.Signal("SIGKILL")
	}
	s.markDone("killed")
	return nil
}

func (s *spritesSession) Controller() AgentController { return noopController{} }

func (s *spritesSession) publish(ev SessionEvent) {
	if ev.Timestamp.IsZero() {
		ev.Timestamp = nowUTC()
	}
	select {
	case <-s.done:
		return
	default:
	}
	select {
	case s.events <- ev:
	default:
		// Drop oldest to keep stream flowing.
		select {
		case <-s.events:
			s.dropped.Add(1)
		default:
		}
		select {
		case <-s.done:
			return
		default:
		}
		select {
		case s.events <- ev:
		default:
		}
	}
}

func (s *spritesSession) markDone(status string) {
	s.doneOnce.Do(func() {
		s.mu.Lock()
		s.status = status
		s.mu.Unlock()
		close(s.done)
	})
}

func (s *spritesSession) closeEventsWhenDone() {
	<-s.done
	s.wg.Wait()
	s.emitDroppedSummary()
	s.eventsMu.Do(func() {
		close(s.events)
	})
}

func (s *spritesSession) emitDroppedSummary() {
	dropped := s.dropped.Load()
	if dropped == 0 {
		return
	}
	message := fmt.Sprintf("dropped %d live session event(s) because the consumer was slow", dropped)
	select {
	case s.events <- SessionEvent{Type: "warning", Message: message, Timestamp: nowUTC()}:
	default:
		select {
		case <-s.events:
		default:
		}
		select {
		case s.events <- SessionEvent{Type: "warning", Message: message, Timestamp: nowUTC()}:
		default:
		}
	}
}

var _ Session = (*spritesSession)(nil)
