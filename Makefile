-include .env
export

# [setup]
.PHONY: setup
setup: ## Interactive project setup
	@go run ./cmd/setup
# [/setup]

# [postgres-docker]
.PHONY: db db-down
# [/postgres-docker]

# [postgres]
.PHONY: migrate migrate-down migrate-new
# [/postgres]

.PHONY: up down watch generate lint format test build dev

# Docker Commands
up:
	docker compose --profile postgres up -d

down:
	docker compose --profile postgres down

watch:
	docker compose watch

# [postgres-docker]
# Database
db:
	docker compose --profile postgres up -d postgres

db-down:
	docker compose --profile postgres down
# [/postgres-docker]

# [postgres]
# Migrations
migrate:
	goose -dir package/db/migrations postgres $(DATABASE_URL) up

migrate-down:
	goose -dir package/db/migrations postgres $(DATABASE_URL) down

migrate-new:
	@read -p "Migration name: " name && \
	goose -dir package/db/migrations create $$name sql
# [/postgres]

# Go Commands
generate:
	go generate ./...

lint:
	go vet ./...

format:
	go fmt ./...

test:
	go test -race ./...

build:
	go build -o bin/template ./cmd/template

dev:
	go run ./cmd/template
