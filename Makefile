.PHONY: help dev stop backend frontend install install-backend install-frontend \
        build build-frontend build-backend clean test logs

# Configurable paths and commands
GO        ?= go
NPM       ?= npm
BACKEND   := backend
FRONTEND  := frontend
BACKEND_PKG := ./cmd/server
FRONTEND_PORT := 5173
BACKEND_PORT := 8080

## Help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make <target>\n\nTargets:\n"} \
	/^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

## Run
stop: ## Stop dev servers bound to the default frontend/backend ports
	@for port in $(FRONTEND_PORT) $(BACKEND_PORT); do \
		pids=$$(lsof -ti :$$port 2>/dev/null || true); \
		if [ -n "$$pids" ]; then \
			echo "Stopping process(es) on port $$port: $$pids"; \
			kill $$pids 2>/dev/null || true; \
			sleep 0.5; \
			pids=$$(lsof -ti :$$port 2>/dev/null || true); \
			[ -n "$$pids" ] && kill -9 $$pids 2>/dev/null || true; \
		fi; \
	done

dev: stop ## Run backend and frontend together (Ctrl-C stops both)
	@trap 'kill 0' EXIT; \
	echo "Starting backend and frontend… (Ctrl-C to stop)"; \
	$(MAKE) -s backend & \
	$(MAKE) -s frontend & \
	wait

backend: ## Run the Go backend on :8080
	@echo "backend → http://localhost:$(BACKEND_PORT)"
	@cd $(BACKEND) && $(GO) run $(BACKEND_PKG)

frontend: ## Run the Vite frontend on :5173
	@echo "frontend → http://localhost:$(FRONTEND_PORT)"
	@cd $(FRONTEND) && $(NPM) run dev -- --port $(FRONTEND_PORT) --strictPort

## Install
install: install-backend install-frontend ## Install all dependencies

install-backend: ## Download Go module dependencies
	@cd $(BACKEND) && $(GO) mod download

install-frontend: ## Install frontend npm dependencies
	@cd $(FRONTEND) && $(NPM) install

## Build
build: build-backend build-frontend ## Build both

build-backend: ## Build the Go backend binary
	@cd $(BACKEND) && $(GO) build -o bin/pulse-server $(BACKEND_PKG)

build-frontend: ## Build the frontend for production
	@cd $(FRONTEND) && $(NPM) run build

## Misc
test: ## Run backend tests
	@cd $(BACKEND) && $(GO) test ./...

clean: ## Remove build artifacts
	@rm -rf $(BACKEND)/bin $(FRONTEND)/dist
