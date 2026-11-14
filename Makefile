# StackLab Makefile. Windows 11 target: run inside WSL2 (README has setup).
# Targets stay thin wrappers so `make up` works on a clean machine.

SHELL := /bin/bash
COMPOSE := docker compose

.PHONY: help up down dev logs ps backend-build gen lint test verify-content reset-playground

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

up: ## Start everything (app-db, playground-db, backend, frontend)
	$(COMPOSE) up -d --build
	@echo "frontend: http://127.0.0.1:5173  backend: http://127.0.0.1:8080/healthz"

down: ## Stop and remove containers (volumes survive)
	$(COMPOSE) down

dev: ## Follow logs while developing
	$(COMPOSE) logs -f --tail=50

ps: ## Show container status
	$(COMPOSE) ps

backend-build: ## Build the Go binaries (requires local Go or use `up`)
	(cd server && go build ./...)

gen: ## Generate sidebar data + lesson bundle
	npm run gen:curriculum
	node scripts/build-lessons.mjs

lint: ## Content checks + Go vet + ESLint (go vet needs local Go)
	node scripts/build-lessons.mjs
	(cd server && go vet ./...)
	npm run lint

test: ## Go tests (integration tag compiles real programs; needs local Go)
	(cd server && go test ./...)
	(cd server && go test -tags integration ./internal/gorun/)

verify-content: ## Validate all lesson content against the schema and style guide
	node scripts/build-lessons.mjs

reset-playground: ## Wipe the SQL playground database
	$(COMPOSE) stop playground-db && $(COMPOSE) rm -f playground-db && docker volume rm stacklab_playground-db-data || true
	$(COMPOSE) up -d playground-db
