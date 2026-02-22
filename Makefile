.PHONY: help build test test-short vet run start status tui skills commands worktree watch clean

GO ?= go
BIN ?= ./bin/noodle
NOODLE ?= $(GO) run .
AIR ?= air

.DEFAULT_GOAL := help

help:
	@echo "Noodle convenience commands"
	@echo ""
	@echo "  make build       Build noodle binary"
	@echo "  make test        Run all tests"
	@echo "  make test-short  Run short tests"
	@echo "  make vet         Run go vet"
	@echo "  make run         Alias for start"
	@echo "  make start       Run scheduling loop"
	@echo "  make status      Show runtime status"
	@echo "  make tui         Open TUI dashboard"
	@echo "  make skills      List resolved skills"
	@echo "  make commands    List available commands"
	@echo "  make worktree    Show worktree help"
	@echo "  make watch       Rebuild on changes with Air"
	@echo "  make clean       Remove built binary"

build:
	$(GO) build -o $(BIN) .

test:
	$(GO) test ./...

test-short:
	$(GO) test -short ./...

vet:
	$(GO) vet ./...

run: start

start:
	$(NOODLE) start

status:
	$(NOODLE) status

tui:
	$(NOODLE) tui

skills:
	$(NOODLE) skills

commands:
	$(NOODLE) commands --json

worktree:
	$(NOODLE) worktree --help

watch:
	@if command -v $(AIR) >/dev/null 2>&1; then \
		$(AIR) -c .air.toml; \
	else \
		echo "air not found; running via 'go run github.com/air-verse/air@latest'"; \
		$(GO) run github.com/air-verse/air@latest -c .air.toml; \
	fi

clean:
	rm -f $(BIN)
