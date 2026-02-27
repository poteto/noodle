package recover

import (
	"encoding/json"
	"strings"
)

func extractPayloadString(payload json.RawMessage, key string) string {
	if len(payload) == 0 {
		return ""
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		return ""
	}
	value, ok := body[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func extractLikelyPath(payload json.RawMessage) []string {
	message := extractPayloadString(payload, "message")
	if message == "" {
		return nil
	}
	fields := strings.Fields(message)
	paths := make([]string, 0, 2)
	for _, field := range fields {
		field = strings.Trim(field, ",.;:()[]{}")
		if strings.Contains(field, "/") || strings.Contains(field, ".") {
			paths = append(paths, field)
		}
	}
	return paths
}
