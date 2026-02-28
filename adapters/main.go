package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	reSectionHeading = regexp.MustCompile(`^##\s+(.+?)\s*$`)
	reTodoLine       = regexp.MustCompile(`^([0-9]+)\. \[([ xX])\] (.*)$`)
	reTag            = regexp.MustCompile(`#[A-Za-z0-9_-]+`)
	reEstimate       = regexp.MustCompile(`~(small|medium|large)`)
	rePlanRef = regexp.MustCompile(`\[\[plans/([0-9]+-[^/]+)/overview\]\]`)
	reSpaces  = regexp.MustCompile(`\s+`)
)

type backlogPayload struct {
	Title   string `json:"title"`
	Section string `json:"section"`
}

func main() {
	if len(os.Args) < 2 {
		die("command is required")
	}

	var err error
	switch os.Args[1] {
	case "backlog-sync":
		err = backlogSync()
	case "backlog-add":
		err = backlogAdd()
	case "backlog-done":
		err = backlogDone(arg(2))
	case "backlog-edit":
		err = backlogEdit(arg(2))
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		die(err.Error())
	}
}

func arg(index int) string {
	if len(os.Args) <= index {
		return ""
	}
	return strings.TrimSpace(os.Args[index])
}

func die(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}

func backlogSync() error {
	todosFile := envOrDefault("NOODLE_TODOS_FILE", "brain/todos.md")
	file, err := os.Open(todosFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	section := ""
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if m := reSectionHeading.FindStringSubmatch(line); len(m) == 2 {
			section = strings.TrimSpace(m[1])
			continue
		}

		m := reTodoLine.FindStringSubmatch(line)
		if len(m) != 4 {
			continue
		}

		id := m[1]
		mark := m[2]
		raw := m[3]
		status := "open"
		if mark == "x" || mark == "X" {
			status = "done"
		}

		tags := reTag.FindAllString(raw, -1)
		normalizedTags := make([]string, 0, len(tags))
		for _, tag := range tags {
			tag = strings.TrimPrefix(tag, "#")
			if tag == "" {
				continue
			}
			normalizedTags = append(normalizedTags, tag)
		}

		estimate := ""
		if em := reEstimate.FindStringSubmatch(raw); len(em) == 2 {
			estimate = em[1]
		}

		plan := ""
		if pm := rePlanRef.FindStringSubmatch(raw); len(pm) == 2 {
			plan = "brain/plans/" + pm[1] + "/overview.md"
		}

		title := raw
		title = reTag.ReplaceAllString(title, "")
		title = reEstimate.ReplaceAllString(title, "")
		title = rePlanRef.ReplaceAllString(title, "")
		title = reSpaces.ReplaceAllString(strings.TrimSpace(title), " ")

		item := map[string]any{
			"id":     id,
			"title":  title,
			"status": status,
			"tags":   normalizedTags,
		}
		if section != "" {
			item["section"] = section
		}
		if plan != "" {
			item["plan"] = plan
		}
		if estimate != "" {
			item["estimate"] = estimate
		}
		if err := encoder.Encode(item); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func backlogAdd() error {
	todosFile := envOrDefault("NOODLE_TODOS_FILE", "brain/todos.md")
	if err := ensureTodosFile(todosFile); err != nil {
		return err
	}

	payload, err := readBacklogPayload(os.Stdin)
	if err != nil {
		return err
	}
	if strings.TrimSpace(payload.Title) == "" {
		return fmt.Errorf("title required")
	}
	section := strings.TrimSpace(payload.Section)
	if section == "" {
		section = "Inbox"
	}

	lines, err := readLines(todosFile)
	if err != nil {
		return err
	}
	id := nextTodoID(lines)

	if !hasSection(lines, section) {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "## "+section)
	}
	lines = append(lines, fmt.Sprintf("%d. [ ] %s", id, payload.Title))
	lines = updateNextID(lines, id+1)

	if err := writeLinesAtomic(todosFile, lines); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, strconv.Itoa(id))
	return nil
}

func backlogDone(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id required")
	}
	todosFile := envOrDefault("NOODLE_TODOS_FILE", "brain/todos.md")
	lines, err := readLines(todosFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("todos file not found")
		}
		return err
	}

	prefix := id + ". [ ]"
	replacement := id + ". [x]"
	for i, line := range lines {
		if strings.HasPrefix(line, prefix) {
			lines[i] = replacement + strings.TrimPrefix(line, prefix)
		}
	}
	return writeLinesAtomic(todosFile, lines)
}

func backlogEdit(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id required")
	}
	payload, err := readBacklogPayload(os.Stdin)
	if err != nil {
		return err
	}
	if strings.TrimSpace(payload.Title) == "" {
		return fmt.Errorf("title required")
	}

	todosFile := envOrDefault("NOODLE_TODOS_FILE", "brain/todos.md")
	lines, err := readLines(todosFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("todos file not found")
		}
		return err
	}

	re := regexp.MustCompile(`^` + regexp.QuoteMeta(id) + `\. \[[ xX]\] `)
	for i, line := range lines {
		if re.MatchString(line) {
			prefix := re.FindString(line)
			lines[i] = prefix + payload.Title
		}
	}
	return writeLinesAtomic(todosFile, lines)
}

func readBacklogPayload(r io.Reader) (backlogPayload, error) {
	var payload backlogPayload
	data, err := io.ReadAll(r)
	if err != nil {
		return payload, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return payload, nil
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return payload, err
	}
	return payload, nil
}

func ensureTodosFile(path string) error {
	_, statErr := os.Stat(path)
	if statErr == nil {
		return nil
	}
	if !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	seed := strings.Join([]string{
		"# Todos",
		"",
		"<!-- next-id: 1 -->",
		"",
		"## Inbox",
		"",
	}, "\n")
	return os.WriteFile(path, []byte(seed), 0o644)
}

func readLines(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	trimmed := strings.TrimSuffix(string(content), "\n")
	if trimmed == "" {
		return []string{}, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

func writeLinesAtomic(path string, lines []string) error {
	data := strings.Join(lines, "\n")
	if len(lines) > 0 {
		data += "\n"
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(data), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func nextTodoID(lines []string) int {
	for _, line := range lines {
		if strings.Contains(line, "next-id:") {
			if m := regexp.MustCompile(`next-id:\s*([0-9]+)`).FindStringSubmatch(line); len(m) == 2 {
				if id, err := strconv.Atoi(m[1]); err == nil {
					return id
				}
			}
		}
	}
	maxID := 0
	for _, line := range lines {
		if m := reTodoLine.FindStringSubmatch(line); len(m) == 4 {
			if id, err := strconv.Atoi(m[1]); err == nil && id > maxID {
				maxID = id
			}
		}
	}
	return maxID + 1
}

func updateNextID(lines []string, nextID int) []string {
	updated := false
	for i, line := range lines {
		if strings.Contains(line, "next-id:") {
			lines[i] = fmt.Sprintf("<!-- next-id: %d -->", nextID)
			updated = true
			break
		}
	}
	if updated {
		return lines
	}
	out := make([]string, 0, len(lines)+2)
	inserted := false
	for i, line := range lines {
		out = append(out, line)
		if i == 1 {
			out = append(out, "", fmt.Sprintf("<!-- next-id: %d -->", nextID))
			inserted = true
		}
	}
	if !inserted {
		out = append([]string{"# Todos", "", fmt.Sprintf("<!-- next-id: %d -->", nextID), ""}, lines...)
	}
	return out
}

func hasSection(lines []string, section string) bool {
	target := "## " + section
	for _, line := range lines {
		if strings.TrimSpace(line) == target {
			return true
		}
	}
	return false
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

