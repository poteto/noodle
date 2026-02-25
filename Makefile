.PHONY: help setup build generate test test-short vet lintarch ci reset run start status skills fixtures-loop fixtures-hash bugs watch watch-verbose clean sandbox

GO ?= go
BIN ?= ./bin/noodle
NOODLE ?= $(GO) run .
AIR ?= air
AIR_FLAGS ?=
AIR_CONFIG ?= .air.toml
UI_MODULES ?= ui/node_modules/.modules.yaml

.DEFAULT_GOAL := help

help:
	@echo "Noodle convenience commands"
	@echo ""
	@printf "  %-40s %s\n" "make setup" "Install all dependencies (Go + UI)"
	@printf "  %-40s %s\n" "make build" "Build noodle binary (includes UI)"
	@printf "  %-40s %s\n" "make test" "Run all tests"
	@printf "  %-40s %s\n" "make test-short" "Run short tests"
	@printf "  %-40s %s\n" "make vet" "Run go vet"
	@printf "  %-40s %s\n" "make lintarch" "Run architecture lint checks"
	@printf "  %-40s %s\n" "make ci" "Run full local CI checks"
	@printf "  %-40s %s\n" "make reset" "Delete runtime state files for debugging"
	@printf "  %-40s %s\n" "make run" "Alias for start"
	@printf "  %-40s %s\n" "make start" "Run scheduling loop"
	@printf "  %-40s %s\n" "make status" "Show runtime status"
	@printf "  %-40s %s\n" "make skills" "List resolved skills"
	@printf "  %-40s %s\n" "make fixtures-loop MODE=check|record" "Verify or regenerate loop runtime dump fixtures"
	@printf "  %-40s %s\n" "make fixtures-hash MODE=check|sync" "Verify or sync fixture source hashes"
	@printf "  %-40s %s\n" "make bugs" "List fixture expected.md files with bug=true"
	@printf "  %-40s %s\n" "make watch" "Dev mode: Air (Go) + Vite (UI) with HMR"
	@printf "  %-40s %s\n" "make watch-verbose" "Dev mode with Air debug logging"
	@printf "  %-40s %s\n" "make generate" "Regenerate auto-generated files"
	@printf "  %-40s %s\n" "make clean" "Remove built binary and UI dist"
	@printf "  %-40s %s\n" "make sandbox STAGE=bare|init|wip|full" "Create a temp sandbox project"

# Install UI dependencies when package.json changes.
$(UI_MODULES): ui/package.json ui/pnpm-lock.yaml
	cd ui && pnpm install
	@touch $@

setup: $(UI_MODULES)
	@echo "dependencies installed"

build: $(UI_MODULES)
	cd ui && pnpm run build
	$(GO) build -o $(BIN) .

generate:
	$(GO) generate ./generate/...

test:
	$(GO) test ./...

test-short:
	$(GO) test -short ./...

vet:
	$(GO) vet ./...

lintarch:
	./scripts/lint-arch.sh

ci: build test vet lintarch fixtures-loop fixtures-hash

reset:
	@for runtime in .noodle .noodles; do \
		rm -f "$$runtime/mise.json" "$$runtime/queue.json" "$$runtime/ticket.json" "$$runtime/tickets.json"; \
		rm -rf "$$runtime/sessions"; \
	done; \
	echo "runtime state reset (.noodle/.noodles)"

run: start

start:
	$(NOODLE) start

status:
	$(NOODLE) status

skills:
	$(NOODLE) skills list

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

watch: $(UI_MODULES)
	@mkdir -p ui/dist/client; \
	trap 'kill 0' EXIT; \
	(cd ui && pnpm run dev) & \
	if $(GO) tool -n air >/dev/null 2>&1; then \
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

sandbox:
	@./scripts/sandbox.sh $(STAGE)

clean:
	rm -f $(BIN)
	rm -rf ui/dist/client/assets ui/dist/client/index.html
