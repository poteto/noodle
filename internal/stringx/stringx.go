package stringx

import "strings"

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
