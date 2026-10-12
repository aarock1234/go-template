-include .env
export

# Tool versions. Keep GOLANGCI_LINT_VERSION in sync with .github/workflows/ci.yaml.
GOLANGCI_LINT_VERSION := v2.14.0
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
# [postgres]
GOOSE_VERSION := v3.29.0
GOOSE := go run github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir pkg/db/migrations
# [/postgres]

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show available targets
	@awk 'BEGIN { FS = ":.*## " } /^[a-zA-Z_-]+:.*## / { printf "  %-14s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# [setup]
.PHONY: setup
setup: ## Configure optional features (pass flags with ARGS="-y ...")
	@go run ./cmd/setup $(ARGS)
# [/setup]

.PHONY: dev build generate fix format lint test

dev: ## Run the application
	go run ./cmd/template

build: ## Compile the binary to bin/template
	go build -trimpath -o bin/template ./cmd/template

generate: ## Run code generation
	go generate ./...

fix: ## Modernize code with go fix
	go fix ./...

format: ## Format code with gofmt and goimports
	$(GOLANGCI_LINT) fmt

lint: ## Run static analysis with golangci-lint
	$(GOLANGCI_LINT) run

test: ## Run tests with the race detector
	go test -race ./...

# [docker]
.PHONY: up down watch

up: ## Build and start the stack with Docker Compose
	docker compose up -d --build

down: ## Stop the Docker Compose stack
	docker compose down

watch: ## Run the stack with hot reload
	docker compose watch
# [/docker]

# [postgres-docker]
.PHONY: db db-down

db: ## Start only PostgreSQL on localhost:5432
	docker compose up -d --wait postgres

db-down: ## Stop PostgreSQL
	docker compose stop postgres
# [/postgres-docker]

# [postgres]
.PHONY: migrate migrate-down migrate-new

migrate: ## Apply all migrations
	$(GOOSE) postgres "$(DATABASE_URL)" up

migrate-down: ## Roll back the last migration
	$(GOOSE) postgres "$(DATABASE_URL)" down

migrate-new: ## Create a migration (make migrate-new name=create_users)
	@test -n "$(name)" || { echo "usage: make migrate-new name=<migration_name>"; exit 1; }
	$(GOOSE) create $(name) sql
# [/postgres]
