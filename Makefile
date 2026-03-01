-include .env
export

.PHONY: up down watch db db-down migrate migrate-down migrate-new generate lint format test build dev

# Docker Commands
up:
	docker compose --profile postgres up -d

down:
	docker compose --profile postgres down

watch:
	docker compose watch

# Database
db:
	docker compose --profile postgres up -d postgres

db-down:
	docker compose --profile postgres down

# Migrations
migrate:
	goose -dir package/db/migrations postgres $(DATABASE_URL) up

migrate-down:
	goose -dir package/db/migrations postgres $(DATABASE_URL) down

migrate-new:
	@read -p "Migration name: " name && \
	goose -dir package/db/migrations create $$name sql

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
