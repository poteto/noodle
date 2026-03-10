#!/bin/sh
#
# Create a temporary Noodle sandbox project at various stages.
# Prints the sandbox path to stdout (all other output goes to stderr).
#
# Usage:
#   cd "$(scripts/sandbox.sh)"        # bare repo
#   cd "$(scripts/sandbox.sh init)"   # after first-run scaffolding
#   cd "$(scripts/sandbox.sh wip)"    # has todos + active plan
#   cd "$(scripts/sandbox.sh full)"   # rich state: todos, plans, sessions
#
# Stages are cumulative — each includes everything from prior stages.

set -e

STAGE="${1:-bare}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
NOODLE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

log() { printf '%s\n' "$@" >&2; }

# --- Create sandbox ---

DIR=$(mktemp -d /tmp/noodle-sandbox.XXXXXX)
cd "$DIR"
git init -q
git commit -q --allow-empty -m "initial commit"

# Dummy project file so it feels like a real repo
cat > main.go <<'GOEOF'
package main

import "fmt"

func main() {
	fmt.Println("hello from noodle sandbox")
}
GOEOF
git add main.go
git commit -q -m "add main.go"

log "sandbox: $DIR (stage: $STAGE)"

if [ "$STAGE" = "bare" ]; then
	echo "$DIR"
	exit 0
fi

# --- Stage: init (first-run scaffolding) ---

mkdir -p brain/plans .noodle .agents/skills

cat > brain/index.md <<'EOF'
# Brain
EOF

cat > brain/todos.md <<'EOF'
# Todos

<!-- next-id: 1 -->
EOF

cat > brain/principles.md <<'EOF'
# Principles
EOF

cat > brain/plans/index.md <<'EOF'
# Plans
EOF

cat > .noodle.toml <<'EOF'
autonomy = "auto"

[routing.defaults]
provider = "codex"
model = "gpt-5.4"

[skills]
paths = [".agents/skills"]
EOF

git add -A
git commit -q -m "noodle init scaffolding"

if [ "$STAGE" = "init" ]; then
	echo "$DIR"
	exit 0
fi

# --- Stage: wip (todos + active plan) ---

cat > brain/todos.md <<'EOF'
# Todos

<!-- next-id: 6 -->

## Features

1. [ ] Add user authentication — support OAuth2 with Google and GitHub providers ~medium
2. [ ] REST API for widgets — CRUD endpoints under /api/v1/widgets with JSON schema validation ~medium
3. [ ] Rate limiting middleware — token bucket per API key, configurable in config.toml ~small

## Bugs

4. [ ] Fix goroutine leak in WebSocket handler — connections aren't cleaned up on client disconnect ~small #bug
5. [ ] Config hot-reload ignores nested keys — changes to [database.pool] require full restart ~small #bug
EOF

mkdir -p brain/plans/01-user-auth

cat > brain/plans/01-user-auth/overview.md <<'EOF'
---
id: 1
created: 2026-02-24
status: active
---

# User Authentication

## Context

The sandbox app needs OAuth2 authentication with Google and GitHub. Sessions stored server-side with secure cookies.

## Scope

**In:** OAuth2 flow (Google, GitHub), session middleware, login/logout endpoints, user model.
**Out:** RBAC, API key auth, admin panel.

## Phases

- [[plans/01-user-auth/phase-01-user-model-and-session-store]]
- [[plans/01-user-auth/phase-02-oauth2-flow]]
- [[plans/01-user-auth/phase-03-session-middleware]]

## Verification

```bash
go test ./auth/... && go test ./middleware/...
```
EOF

cat > brain/plans/01-user-auth/phase-01-user-model-and-session-store.md <<'EOF'
Back to [[plans/01-user-auth/overview]]

# Phase 1: User model and session store

**Routing:** `codex` / `gpt-5.4` — clear spec, mechanical types

## Goal

Define the User struct, session store interface, and in-memory implementation.

## Changes

- `auth/user.go` — User struct with ID, email, name, provider, created/updated timestamps
- `auth/session.go` — SessionStore interface (Create, Get, Delete, Cleanup)
- `auth/memory_store.go` — in-memory implementation with expiry

## Verification

```bash
go test ./auth/...
```
EOF

cat > brain/plans/01-user-auth/phase-02-oauth2-flow.md <<'EOF'
Back to [[plans/01-user-auth/overview]]

# Phase 2: OAuth2 flow

**Routing:** `codex` / `gpt-5.4` — integration work, provider-specific quirks

## Goal

Implement OAuth2 authorization code flow for Google and GitHub.

## Changes

- `auth/providers.go` — provider config, callback handlers
- `auth/oauth.go` — shared OAuth2 utilities (state generation, token exchange)

## Verification

```bash
go test ./auth/... -run TestOAuth
```
EOF

cat > brain/plans/01-user-auth/phase-03-session-middleware.md <<'EOF'
Back to [[plans/01-user-auth/overview]]

# Phase 3: Session middleware

**Routing:** `codex` / `gpt-5.4` — straightforward middleware pattern

## Goal

HTTP middleware that validates session cookies and injects user into request context.

## Changes

- `middleware/session.go` — RequireAuth and OptionalAuth middleware
- `middleware/session_test.go` — tests with mock session store

## Verification

```bash
go test ./middleware/...
```
EOF

cat > brain/plans/index.md <<'EOF'
# Plans

- [ ] [[plans/01-user-auth/overview]]
EOF

# Copy core skills from the noodle repo
for skill in schedule execute quality; do
	cp -R "$NOODLE_ROOT/.agents/skills/$skill" ".agents/skills/$skill"
done

git add -A
git commit -q -m "seed todos, active plan, and core skills"

if [ "$STAGE" = "wip" ]; then
	echo "$DIR"
	exit 0
fi

# --- Stage: full (archived plan + runtime state + session history) ---

mkdir -p brain/archive/plans/00-project-setup

cat > brain/archive/plans/00-project-setup/overview.md <<'EOF'
---
id: 0
created: 2026-02-23
status: completed
---

# Project Setup

## Context

Bootstrap the Go project with module, CI, and basic HTTP server.

## Phases

- [[archive/plans/00-project-setup/phase-01-module-and-ci]]

## Verification

```bash
go build . && go test ./...
```
EOF

cat > brain/archive/plans/00-project-setup/phase-01-module-and-ci.md <<'EOF'
Back to [[archive/plans/00-project-setup/overview]]

# Phase 1: Module and CI

## Goal

Initialize Go module, add Makefile, configure GitHub Actions.

## Changes

- `go.mod`, `Makefile`, `.github/workflows/ci.yml`

## Verification

```bash
make ci
```
EOF

# Runtime state
cat > .noodle/mise.json <<'EOF'
{
  "project_dir": ".",
  "todos": [
    {"id": 1, "text": "Add user authentication", "done": false},
    {"id": 2, "text": "REST API for widgets", "done": false},
    {"id": 3, "text": "Rate limiting middleware", "done": false},
    {"id": 4, "text": "Fix goroutine leak in WebSocket handler", "done": false},
    {"id": 5, "text": "Config hot-reload ignores nested keys", "done": false}
  ],
  "active_plans": ["plans/01-user-auth/overview"],
  "available_runtimes": ["tmux"]
}
EOF

cat > .noodle/orders.json <<'EOF'
{
  "orders": [
    {
      "id": "todo-4",
      "title": "Fix goroutine leak in WebSocket handler",
      "status": "active",
      "rationale": "Bug fix — reliability issue, small scope, high impact.",
      "stages": [
        {
          "task_key": "execute",
          "provider": "codex",
          "model": "gpt-5.4",
          "runtime": "tmux",
          "status": "pending"
        }
      ]
    }
  ]
}
EOF

cat > .noodle/status.json <<'EOF'
{
  "active": [],
  "loop_state": "running",
  "autonomy": "auto",
  "max_concurrency": 1
}
EOF

git add -A
git commit -q -m "seed archived plan and runtime state"

echo "$DIR"
