package dispatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/poteto/noodle/event"
	"github.com/poteto/noodle/parse"
	"github.com/poteto/noodle/stamp"
)

// sessionBase contains the shared fields and methods for processSession and
// spritesSession. Both session types embed this struct and delegate the
// identical event-handling, publishing, and lifecycle plumbing to it.
type sessionBase struct {
	id          string
	eventWriter *event.EventWriter

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

	prompt       string
	promptLogged bool
	sink         SessionEventSink
}

// sessionBaseConfig holds the common fields needed to initialize a sessionBase.
type sessionBaseConfig struct {
	id            string
	eventWriter   *event.EventWriter
	stampedPath   string
	canonicalPath string
	prompt        string
	sink          SessionEventSink
}

func newSessionBase(cfg sessionBaseConfig) sessionBase {
	return sessionBase{
		id:            cfg.id,
		eventWriter:   cfg.eventWriter,
		stampedPath:   cfg.stampedPath,
		canonicalPath: cfg.canonicalPath,
		prompt:        cfg.prompt,
		sink:          cfg.sink,
		status:        "running",
		done:          make(chan struct{}),
		events:        make(chan SessionEvent, 32),
	}
}

// --- Session interface methods (promoted via embedding) ---

func (s *sessionBase) ID() string { return s.id }

func (s *sessionBase) Status() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *sessionBase) Events() <-chan SessionEvent { return s.events }
func (s *sessionBase) Done() <-chan struct{}       { return s.done }

func (s *sessionBase) TotalCost() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.costUSD
}

// --- Publishing ---

func (s *sessionBase) publish(ev SessionEvent) {
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

// --- Done / drain ---

func (s *sessionBase) markDone(status string) {
	s.doneOnce.Do(func() {
		s.mu.Lock()
		s.status = status
		s.mu.Unlock()
		close(s.done)
	})
}

func (s *sessionBase) closeEventsWhenDone() {
	<-s.done
	s.wg.Wait()
	s.emitDroppedSummary()
	s.eventsMu.Do(func() {
		close(s.events)
	})
}

func (s *sessionBase) emitDroppedSummary() {
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

// --- Stream processing ---

// processStream opens the stamped and canonical output files, runs the stamp
// processor, and routes canonical events through the given interceptor writer.
func (s *sessionBase) processStream(ctx context.Context, stdout io.Reader, interceptor io.Writer) {
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
	if err := processor.Process(ctx, stdout, stampedFile, io.MultiWriter(canonicalFile, interceptor)); err != nil {
		if ctx.Err() == nil {
			s.publish(SessionEvent{Type: "warning", Message: "process stream: " + err.Error(), Timestamp: nowUTC()})
		}
	}
}

// --- Canonical event handling ---

// canonicalLineHook is called by consumeCanonicalLine for each non-delta
// canonical event, allowing session-specific side effects (heartbeat writes,
// controller notification).
type canonicalLineHook func(ce parse.CanonicalEvent)

// consumeCanonicalLine parses a canonical event JSON line, routes deltas to
// the sink, accumulates cost, publishes session events, writes to the event
// log, and calls the optional hook for session-specific behavior.
func (s *sessionBase) consumeCanonicalLine(line []byte, hook canonicalLineHook) {
	var ce parse.CanonicalEvent
	if err := json.Unmarshal(line, &ce); err != nil {
		return
	}

	if ce.Type == parse.EventDelta {
		if s.sink != nil {
			s.sink.PublishSessionDelta(s.id, ce.Message, ce.Timestamp)
		}
		return
	}

	if hook != nil {
		hook(ce)
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

	if s.sink != nil {
		if ev, ok := FormatEventLine(s.id, ce); ok {
			s.sink.PublishSessionEvent(s.id, ev)
		}
	}

	if ce.Type == parse.EventInit {
		s.emitPromptEvent(ce.Timestamp)
	}
}

// emitPromptEvent publishes the user prompt as a session event and writes it
// to both the event log and the sink.
func (s *sessionBase) emitPromptEvent(timestamp time.Time) {
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
		Type:      string(parse.EventAction),
		Message:   prompt,
		Timestamp: timestamp,
	})

	payload, err := json.Marshal(map[string]any{
		"tool":    "prompt",
		"action":  "prompt_injected",
		"message": s.prompt,
	})
	if err != nil {
		return
	}
	promptEvent := event.Event{
		Type:      event.EventAction,
		Payload:   payload,
		Timestamp: timestamp,
		SessionID: s.id,
	}

	if s.eventWriter != nil {
		if err := s.eventWriter.Append(context.Background(), promptEvent); err != nil {
			s.publish(SessionEvent{
				Type:      "warning",
				Message:   "event log append failed: " + err.Error(),
				Timestamp: nowUTC(),
			})
		}
	}

	if s.sink != nil {
		s.sink.PublishSessionEvent(s.id, promptEvent)
	}
}

// --- Interceptor ---

// canonicalLineInterceptor is an io.Writer that splits incoming bytes into
// lines and forwards each to a callback.
type canonicalLineInterceptor struct {
	onLine func(line []byte)
}

func (w *canonicalLineInterceptor) Write(p []byte) (int, error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		w.onLine([]byte(line))
	}
	return len(p), nil
}
