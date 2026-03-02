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

# [docker]
.PHONY: up down watch
# [/docker]

.PHONY: generate fix lint format test build dev

# [docker]
up:
	docker compose --profile postgres up -d

down:
	docker compose --profile postgres down

watch:
	docker compose watch
# [/docker]

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

fix:
	go fix ./...

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
