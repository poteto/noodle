package shellx

import "strings"

// Quote wraps a shell token in single quotes and escapes embedded single quotes.
func Quote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

// SanitizeToken normalizes a free-form token into lowercase dash-separated text.
func SanitizeToken(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = fallback
	}

	var out strings.Builder
	lastHyphen := false
	for _, r := range value {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			out.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			out.WriteByte('-')
			lastHyphen = true
		}
	}

	result := strings.Trim(out.String(), "-")
	if result == "" {
		result = fallback
	}
	if len(result) > 48 {
		result = strings.Trim(result[:48], "-")
	}
	if result == "" {
		return fallback
	}
	return result
}
