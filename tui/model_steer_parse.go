package tui

import (
	"fmt"
	"strings"
)

func parseSteerInput(raw string, validTargets []string) ([]string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, "", fmt.Errorf("type @target and an instruction")
	}

	valid := map[string]string{}
	for _, target := range validTargets {
		key := strings.ToLower(strings.TrimSpace(target))
		if key == "" {
			continue
		}
		valid[key] = target
	}

	mentions := make([]string, 0, 2)
	words := make([]string, 0, 8)
	for _, token := range strings.Fields(raw) {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if !strings.HasPrefix(token, "@") {
			words = append(words, token)
			continue
		}
		mention := strings.TrimPrefix(token, "@")
		mention = strings.TrimSpace(strings.TrimRight(mention, ",.;:"))
		if mention == "" {
			continue
		}
		mentions = append(mentions, mention)
	}
	if len(mentions) == 0 {
		return nil, "", fmt.Errorf("missing @target mention")
	}

	prompt := strings.TrimSpace(strings.Join(words, " "))
	if prompt == "" {
		return nil, "", fmt.Errorf("instruction text is required")
	}

	resolved := make([]string, 0, len(validTargets))
	for _, mention := range mentions {
		mentionKey := strings.ToLower(mention)
		if mentionKey == "everyone" {
			resolved = append(resolved, validTargets...)
			continue
		}
		canonical, ok := valid[mentionKey]
		if !ok {
			return nil, "", fmt.Errorf("unknown target @%s", mention)
		}
		resolved = append(resolved, canonical)
	}
	resolved = uniqueStrings(resolved)
	if len(resolved) == 0 {
		return nil, "", fmt.Errorf("no valid targets selected")
	}
	return resolved, prompt, nil
}

func uniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func (m *Model) refreshSteerMentions() {
	start, query, ok := mentionQuery(m.steerInput)
	if !ok {
		m.closeSteerMentions()
		return
	}
	candidates := mentionCandidates(query, m.steerTargets())
	if len(candidates) == 0 {
		m.closeSteerMentions()
		return
	}
	m.steerMentionOpen = true
	m.steerMentionStart = start
	m.steerMentionItems = candidates
	if m.steerMentionIndex >= len(candidates) {
		m.steerMentionIndex = len(candidates) - 1
	}
	if m.steerMentionIndex < 0 {
		m.steerMentionIndex = 0
	}
}

func mentionQuery(input string) (int, string, bool) {
	if input == "" {
		return 0, "", false
	}
	start := len(input) - 1
	for start >= 0 && input[start] != ' ' && input[start] != '\t' && input[start] != '\n' {
		start--
	}
	start++
	if start >= len(input) || input[start] != '@' {
		return 0, "", false
	}
	return start, strings.ToLower(strings.TrimSpace(input[start+1:])), true
}

func mentionCandidates(query string, targets []string) []string {
	all := []string{"@everyone"}
	for _, target := range targets {
		all = append(all, "@"+target)
	}
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return all
	}
	out := make([]string, 0, len(all))
	for _, candidate := range all {
		if strings.HasPrefix(strings.ToLower(strings.TrimPrefix(candidate, "@")), query) {
			out = append(out, candidate)
		}
	}
	return out
}

func (m *Model) applySteerMention(selection string) {
	if m.steerMentionStart < 0 || m.steerMentionStart > len(m.steerInput) {
		m.steerInput = strings.TrimSpace(m.steerInput + " " + selection + " ")
		m.closeSteerMentions()
		return
	}
	prefix := m.steerInput[:m.steerMentionStart]
	m.steerInput = prefix + selection + " "
	m.closeSteerMentions()
}

func (m *Model) closeSteerMentions() {
	m.steerMentionOpen = false
	m.steerMentionItems = nil
	m.steerMentionIndex = 0
	m.steerMentionStart = -1
}

func dropLastRune(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= 1 {
		return ""
	}
	return string(runes[:len(runes)-1])
}
