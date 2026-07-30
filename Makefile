# ============================================================
# EnviToo — Makefile
# ============================================================

.PHONY: help \
	dev stop backend frontend worker mailer email \
	setup install install-backend install-frontend \
	build build-backend build-frontend build-docker \
	dc-up dc-stop dc-down dc-logs dc-ps dc-full \
	test check check-backend check-frontend ci clean

# ------------------------------------------------------------
# Config
# ------------------------------------------------------------

GO            ?= go
NPM           ?= npm

BACKEND       := backend
FRONTEND      := client

BACKEND_PKG   := ./cmd/server
WORKER_PKG    := ./cmd/worker
MAILER_PKG    := ./cmd/mailer

BACKEND_PORT  := 8080
FRONTEND_PORT := 5173

# ============================================================
# Help
# ============================================================

help: ## Show available targets
	@awk 'BEGIN {FS=":.*##"; printf "Usage:\n  make <target>\n\nTargets:\n"} \
	/^[a-zA-Z0-9_.-]+:.*##/ {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ============================================================
# Development
# ============================================================

dev: stop dc-up ## Run backend + frontend with infra
	@trap 'kill 0' EXIT; \
	echo "Starting backend and frontend… (Ctrl-C to stop)"; \
	$(MAKE) -s backend & \
	$(MAKE) -s frontend & \
	wait

backend: ## Run Go backend on :8080
	@echo "backend → http://localhost:$(BACKEND_PORT)"
	@cd $(BACKEND) && $(GO) run $(BACKEND_PKG)

frontend: ## Run Vite frontend on :5173
	@echo "frontend → http://localhost:$(FRONTEND_PORT)"
	@cd $(FRONTEND) && $(NPM) run dev -- --port $(FRONTEND_PORT) --strictPort

worker: ## Run outbox → NATS drainer
	@echo "Starting email worker..."
	@cd $(BACKEND) && $(GO) run $(WORKER_PKG)

mailer: ## Run NATS → Resend mailer
	@echo "Starting email mailer..."
	@cd $(BACKEND) && $(GO) run $(MAILER_PKG)

email: dc-up ## Run infra + worker + mailer (email pipeline)
	@echo "Email pipeline starting (worker + mailer)… (Ctrl-C to stop)"
	@cd $(BACKEND) && $(GO) run $(WORKER_PKG) &
	@cd $(BACKEND) && $(GO) run $(MAILER_PKG) &
	@wait

stop: ## Kill processes on frontend/backend dev ports
	@for port in $(FRONTEND_PORT) $(BACKEND_PORT); do \
		pids=$$(lsof -ti :$$port 2>/dev/null || true); \
		[ -z "$$pids" ] && continue; \
		echo "Stopping process(es) on port $$port: $$pids"; \
		kill $$pids 2>/dev/null || true; \
		sleep 0.5; \
		pids=$$(lsof -ti :$$port 2>/dev/null || true); \
		[ -n "$$pids" ] && kill -9 $$pids 2>/dev/null || true; \
	done

# ============================================================
# Setup / Install
# ============================================================

setup: ## Install git hooks, then all dependencies
	@CURRENT=$$(git config core.hooksPath 2>/dev/null || echo ""); \
	if [ "$$CURRENT" != ".githooks" ]; then \
		git config core.hooksPath .githooks; \
		echo "✓ git hooks configured (.githooks/)"; \
	else \
		echo "✓ git hooks already configured (.githooks/)"; \
	fi
	@$(MAKE) install

install: install-backend install-frontend ## Install all dependencies

install-backend: ## Download Go module dependencies
	@cd $(BACKEND) && $(GO) mod download

install-frontend: ## Install frontend npm dependencies
	@cd $(FRONTEND) && $(NPM) install

# ============================================================
# Build
# ============================================================

build: build-backend build-frontend ## Build both backend and frontend

build-backend: ## Build the Go backend binary
	@mkdir -p $(BACKEND)/bin
	@cd $(BACKEND) && $(GO) build -o bin/server $(BACKEND_PKG)

build-frontend: ## Build the frontend for production
	@cd $(FRONTEND) && $(NPM) run build

build-docker: ## Build the backend Docker image
	docker build -t envitoo/backend:latest ./backend

# ============================================================
# Docker Compose (infra)
# ============================================================

dc-up: ## Start infra services (postgres, redis, nats) in background
	docker compose up -d --wait

dc-stop: ## Stop infra services
	docker compose stop

dc-down: ## Stop and remove containers, networks, volumes
	docker compose down

dc-logs: ## Tail infra service logs
	docker compose logs -f

dc-ps: ## List running infra services
	docker compose ps

dc-full: ## Build and run the full stack (infra + backend) in docker
	docker compose --profile full up --build -d

# ============================================================
# Quality checks
# ============================================================

test: ## Run backend tests
	@cd $(BACKEND) && $(GO) test ./...

check-backend: ## Backend: format, vet, test
	@cd $(BACKEND) && test -z "$$(gofmt -l .)"
	@cd $(BACKEND) && $(GO) vet ./...
	@cd $(BACKEND) && $(GO) test ./...

check-frontend: ## Frontend: lint, typecheck, build
	@cd $(FRONTEND) && $(NPM) run lint
	@cd $(FRONTEND) && $(NPM) run typecheck
	@cd $(FRONTEND) && $(NPM) run build

check: check-backend check-frontend ## Run all checks (backend + frontend)

ci: check ## Alias for check (CI / pre-commit entry point)

# ============================================================
# Misc
# ============================================================

clean: ## Remove build artifacts
	@rm -rf $(BACKEND)/bin $(FRONTEND)/dist
