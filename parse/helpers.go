package parse

import (
	"encoding/json"
	"strings"
	"time"
)

func parseRFC3339(ts string) time.Time {
	ts = strings.TrimSpace(ts)
	if ts == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func parseTimestamp(preferred string, fallback string) time.Time {
	if ts := parseRFC3339(preferred); !ts.IsZero() {
		return ts
	}
	return parseRFC3339(fallback)
}

func extractStringField(raw json.RawMessage, key string) string {
	if len(raw) == 0 {
		return ""
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return ""
	}
	value, ok := fields[key]
	if !ok {
		return ""
	}
	var out string
	if err := json.Unmarshal(value, &out); err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func parseTimestampField(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.TrimSpace(text)
	}

	return ""
}
