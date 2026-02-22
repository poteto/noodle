.PHONY: help build test test-short vet run start status tui skills commands worktree watch watch-verbose clean

GO ?= go
BIN ?= ./bin/noodle
NOODLE ?= $(GO) run .
AIR ?= air
AIR_FLAGS ?=
AIR_CONFIG ?= .air.toml

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
	@echo "  make skills      List resolved skills"
	@echo "  make commands    List available commands"
	@echo "  make worktree    Show worktree help"
	@echo "  make watch       Rebuild on changes with Air (silent mode)"
	@echo "  make watch-verbose Rebuild on changes with Air (debug file events)"
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
	@echo "make tui has been removed. Use: make start"
	@exit 1

skills:
	$(NOODLE) skills

commands:
	$(NOODLE) commands --json

worktree:
	$(NOODLE) worktree --help

watch:
	@if $(GO) tool -n air >/dev/null 2>&1; then \
		$(GO) tool air $(AIR_FLAGS) -c $(AIR_CONFIG); \
	elif command -v $(AIR) >/dev/null 2>&1; then \
		$(AIR) $(AIR_FLAGS) -c $(AIR_CONFIG); \
	else \
		echo "air not found. Install with: go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi

watch-verbose: AIR_FLAGS = -d
watch-verbose: AIR_CONFIG = .air.verbose.toml
watch-verbose: watch

clean:
	rm -f $(BIN)
