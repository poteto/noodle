package stamp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/poteto/noodle/parse"
)

// Processor stamps NDJSON lines and emits canonical sidecar events.
type Processor struct {
	Registry     *parse.Registry
	Now          func() time.Time
	MaxLineBytes int
}

func NewProcessor() *Processor {
	return &Processor{
		Registry:     parse.NewRegistry(),
		Now:          time.Now,
		MaxLineBytes: 64 << 20,
	}
}

// Process streams stdin-like input into stamped NDJSON + optional events.
func (p *Processor) Process(
	ctx context.Context,
	in io.Reader,
	stampedOut io.Writer,
	eventsOut io.Writer,
) error {
	scanner := bufio.NewScanner(in)
	maxLineBytes := p.MaxLineBytes
	if maxLineBytes <= 0 {
		maxLineBytes = 64 << 20
	}
	initialBuffer := 64 * 1024
	if maxLineBytes < initialBuffer {
		initialBuffer = maxLineBytes
	}
	if initialBuffer < 1 {
		initialBuffer = 1
	}
	scanner.Buffer(make([]byte, 0, initialBuffer), maxLineBytes)

	lineNumber := 0
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		lineNumber++
		rawLine := strings.TrimSpace(scanner.Text())
		if rawLine == "" {
			continue
		}

		// Skip non-JSON lines (e.g., codex "Reading prompt from stdin..." banner).
		if rawLine[0] != '{' {
			continue
		}

		stamped, events, err := p.ProcessLine([]byte(rawLine))
		if err != nil {
			return fmt.Errorf("process line %d: %w", lineNumber, err)
		}

		if _, err := stampedOut.Write(append(stamped, '\n')); err != nil {
			return fmt.Errorf("write stamped line %d: %w", lineNumber, err)
		}

		if eventsOut == nil {
			continue
		}
		for _, event := range events {
			encoded, err := json.Marshal(event)
			if err != nil {
				return fmt.Errorf("encode sidecar event line %d: %w", lineNumber, err)
			}
			if _, err := eventsOut.Write(append(encoded, '\n')); err != nil {
				return fmt.Errorf("write sidecar event line %d: %w", lineNumber, err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	return nil
}

func (p *Processor) ProcessLine(line []byte) ([]byte, []parse.CanonicalEvent, error) {
	stampedLine, stampTime, err := injectTimestamp(line, p.Now().UTC())
	if err != nil {
		return nil, nil, err
	}

	provider, events, err := p.Registry.ParseLine(stampedLine)
	if err != nil {
		return stampedLine, []parse.CanonicalEvent{{
			Type:      parse.EventError,
			Message:   "canonical routing failed: " + err.Error(),
			Timestamp: stampTime,
		}}, nil
	}

	for i := range events {
		if events[i].Provider == "" {
			events[i].Provider = provider
		}
		if events[i].Timestamp.IsZero() {
			events[i].Timestamp = stampTime
		}
	}

	return stampedLine, events, nil
}

func injectTimestamp(line []byte, now time.Time) ([]byte, time.Time, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(line, &payload); err != nil {
		return nil, time.Time{}, fmt.Errorf("parse JSON object: %w", err)
	}

	ts := now.Format(time.RFC3339Nano)
	encodedTimestamp, err := json.Marshal(ts)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("encode timestamp: %w", err)
	}
	payload["_ts"] = encodedTimestamp

	stamped, err := json.Marshal(payload)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("encode stamped line: %w", err)
	}
	return stamped, now, nil
}
