package server

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poteto/noodle/loop"
)

func TestPostControlAcceptsBacklogActions(t *testing.T) {
	s, dir := testServer(t)

	bodies := []string{
		`{"action":"backlog-add","prompt":"Add search"}`,
		`{"action":"backlog-edit","target":"4","prompt":"Add fuzzy search"}`,
		`{"action":"backlog-done","target":"4"}`,
	}
	for _, body := range bodies {
		req := httptest.NewRequest("POST", "/api/control", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.httpServer.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, body %q", body, w.Code, w.Body.String())
		}
	}

	file, err := os.Open(filepath.Join(dir, "control.ndjson"))
	if err != nil {
		t.Fatalf("open control.ndjson: %v", err)
	}
	defer file.Close()
	var cmds []loop.ControlCommand
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var cmd loop.ControlCommand
		if err := json.Unmarshal(scanner.Bytes(), &cmd); err != nil {
			t.Fatalf("parse command: %v", err)
		}
		cmds = append(cmds, cmd)
	}
	if len(cmds) != 3 {
		t.Fatalf("written commands = %+v, want 3", cmds)
	}
	if cmds[0].Action != "backlog-add" || cmds[0].Prompt != "Add search" {
		t.Fatalf("add command = %+v", cmds[0])
	}
	if cmds[1].Action != "backlog-edit" || cmds[1].Target != "4" || cmds[1].Prompt != "Add fuzzy search" {
		t.Fatalf("edit command = %+v", cmds[1])
	}
	if cmds[2].Action != "backlog-done" || cmds[2].Target != "4" {
		t.Fatalf("done command = %+v", cmds[2])
	}
}
