package stringx

import (
	"strings"
	"unicode/utf8"
)

// FirstNonEmpty returns the first non-empty trimmed value.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

// NonEmpty returns value when non-empty after trimming, otherwise fallback.
func NonEmpty(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return strings.TrimSpace(fallback)
	}
	return value
}

// MiddleTruncate shortens a string by replacing the middle with "…".
// For paths containing slashes, it preserves the first and last segments
// and truncates inner segments (e.g., "/Users/…/queue.json").
// For plain strings, it keeps the start and end.
func MiddleTruncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	n := utf8.RuneCountInString(s)
	if n <= maxWidth {
		return s
	}
	const ellipsis = "…"
	if maxWidth <= 1 {
		return ellipsis
	}

	// Path-aware: preserve first and last path segments.
	if strings.Contains(s, "/") {
		return middleTruncatePath(s, maxWidth)
	}

	// Plain string: keep start and end.
	half := (maxWidth - 1) / 2
	tail := maxWidth - 1 - half
	runes := []rune(s)
	return string(runes[:half]) + ellipsis + string(runes[n-tail:])
}

func middleTruncatePath(s string, maxWidth int) string {
	const ellipsis = "…"
	parts := strings.Split(s, "/")

	// Try keeping first and last segment with ellipsis between.
	if len(parts) >= 3 {
		first := parts[0]
		last := parts[len(parts)-1]
		candidate := first + "/…/" + last
		if utf8.RuneCountInString(candidate) <= maxWidth {
			// Try to fit more segments from the end.
			for i := len(parts) - 2; i >= 1; i-- {
				try := first + "/…/" + strings.Join(parts[i:], "/")
				if utf8.RuneCountInString(try) <= maxWidth {
					candidate = try
				} else {
					break
				}
			}
			return candidate
		}
	}

	// Fall back to plain middle truncation.
	n := utf8.RuneCountInString(s)
	half := (maxWidth - 1) / 2
	tail := maxWidth - 1 - half
	runes := []rune(s)
	return string(runes[:half]) + ellipsis + string(runes[n-tail:])
}
