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
// and truncates inner segments (e.g., "/Users/…/orders.json").
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

// KitchenName generates a deterministic short display name from a session ID.
// Returns an adjective-noun pair from kitchen/food terms (e.g., "crispy-basil").
func KitchenName(sessionID string) string {
	adjectives := []string{
		"crispy", "smoky", "tangy", "zesty", "golden",
		"spicy", "savory", "tender", "bitter", "silky",
		"briny", "toasty", "mellow", "rustic", "bright",
		"hearty", "flaky", "velvety", "snappy", "robust",
	}
	nouns := []string{
		"basil", "thyme", "sage", "fennel", "ginger",
		"turnip", "shallot", "leek", "chive", "sorrel",
		"cumin", "sumac", "saffron", "nutmeg", "clove",
		"radish", "endive", "cress", "miso", "tahini",
	}
	h := fnvHash(sessionID)
	adj := adjectives[h%uint32(len(adjectives))]
	noun := nouns[(h/uint32(len(adjectives)))%uint32(len(nouns))]
	return adj + "-" + noun
}

func fnvHash(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
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
