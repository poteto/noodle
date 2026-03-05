package server

import (
	"slices"
	"sort"
)

// mergeWarnings combines static config warnings with dynamic loop warnings.
// Allocates a fresh slice to avoid aliasing either input — concurrent
// HTTP/WS goroutines may read the result simultaneously.
func mergeWarnings(static, dynamic []string) []string {
	merged := make([]string, 0, len(static)+len(dynamic))
	merged = append(merged, static...)
	merged = append(merged, dynamic...)
	sort.Strings(merged)
	return slices.Compact(merged)
}
