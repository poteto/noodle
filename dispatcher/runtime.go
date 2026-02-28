package dispatcher

import "strings"

// NormalizeRuntime lowercases and trims the runtime string, defaulting to "process".
func NormalizeRuntime(runtime string) string {
	runtime = strings.ToLower(strings.TrimSpace(runtime))
	if runtime == "" {
		return "process"
	}
	return runtime
}
