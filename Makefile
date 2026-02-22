.PHONY: help build test test-short vet lintarch ci run start status skills commands fixtures-loop fixtures-hash bugs worktree watch watch-verbose clean

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
	@echo "  make lintarch    Run architecture lint checks"
	@echo "  make ci          Run full local CI checks"
	@echo "  make run         Alias for start"
	@echo "  make start       Run scheduling loop"
	@echo "  make status      Show runtime status"
	@echo "  make skills      List resolved skills"
	@echo "  make commands    List available commands"
	@echo "  make fixtures-loop MODE=check|record  Verify or regenerate loop runtime dump fixtures"
	@echo "  make fixtures-hash MODE=check|sync    Verify or sync fixture source hashes"
	@echo "  make bugs        List fixture expected.md files with bug=true"
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

lintarch:
	./scripts/lint-arch.sh

ci: test vet lintarch fixtures-loop fixtures-hash

run: start

start:
	$(NOODLE) start

status:
	$(NOODLE) status

skills:
	$(NOODLE) skills

commands:
	$(NOODLE) commands --json

fixtures-loop:
	@mode="$${MODE:-check}"; \
	case "$$mode" in \
		record) \
			echo "loop fixtures: record"; \
			NOODLE_LOOP_FIXTURE_MODE=record $(GO) test ./loop -run TestLoopDirectoryFixtures -count=1; \
			$(GO) run ./scripts/fixturehash sync --root loop/testdata; \
			;; \
		check) \
			echo "loop fixtures: check"; \
			NOODLE_LOOP_FIXTURE_MODE=check $(GO) test ./loop -run TestLoopDirectoryFixtures -count=1; \
			$(GO) run ./scripts/fixturehash check --root loop/testdata; \
			;; \
		*) \
			echo "unsupported MODE=$$mode (expected check|record)"; \
			exit 1; \
			;; \
	esac

fixtures-hash:
	@mode="$${MODE:-check}"; \
	case "$$mode" in \
		sync) \
			$(GO) run ./scripts/fixturehash sync; \
			;; \
		check) \
			$(GO) run ./scripts/fixturehash check; \
			;; \
		*) \
			echo "unsupported MODE=$$mode (expected check|sync)"; \
			exit 1; \
			;; \
	esac

bugs:
	@rg --files -g '**/expected.md' | sort | while IFS= read -r f; do \
		if awk 'BEGIN{in_frontmatter=0; found=0} \
			/^---[[:space:]]*$$/ { in_frontmatter = !in_frontmatter; next } \
			in_frontmatter && /^bug:[[:space:]]*true[[:space:]]*$$/ { found=1; exit } \
			in_frontmatter && /^bug:[[:space:]]*/ { exit } \
			END { exit !found }' "$$f"; then \
			echo "$$f"; \
		fi; \
	done

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
