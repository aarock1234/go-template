# go-template

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-Elastic_2.0-blue)
![CI](https://github.com/aarock1234/go-template/actions/workflows/ci.yaml/badge.svg)

An opinionated Go project template for scraper, bot, and service workloads. Batteries included: TLS-fingerprinted HTTP clients, exponential backoff, bounded concurrency, file-backed state, PostgreSQL, and structured logging.

## Quick Start

Install [`gonew`](https://pkg.go.dev/golang.org/x/tools/cmd/gonew):

```bash
go install golang.org/x/tools/cmd/gonew@latest
```

Scaffold a new project:

```bash
gonew github.com/aarock1234/go-template@latest github.com/you/myproject
```

Clones the template and rewrites all import paths to your module name.

After cloning, run the setup script to configure optional features:

```bash
./setup.sh
```

## Setup

**Prerequisites:** [Go 1.26+](https://go.dev/dl/), [Docker](https://docs.docker.com/get-docker/), [goose](https://github.com/pressly/goose), [sqlc](https://sqlc.dev)

1. Replace `pkg/template` with your own domain logic
2. Update `cmd/template/main.go` to wire your services
3. Add SQL queries to `pkg/db/queries/` and run `make generate`
4. Copy and configure your environment:

```bash
cp .env.example .env
```

## Running

Locally:

```bash
make dev
```

Or with Docker:

```bash
make up
```

## Project Structure

```
go-template/
├── cmd/template/       entrypoint
├── pkg/
│   ├── client/         HTTP client with TLS/HTTP2 fingerprinting, proxy, cookies
│   ├── cycle/          thread-safe round-robin file rotator
│   ├── db/             PostgreSQL pool, sqlc queries, transactions, advisory locks
│   ├── env/            .env loader and struct-tag validation
│   ├── log/            structured slog with tint and context injection
│   ├── ptr/            generic pointer helpers
│   ├── retry/          exponential backoff with jitter
│   ├── state/          file-backed JSON persistence with file locking
│   ├── template/       skeleton service (replace this)
│   └── worker/         bounded-concurrency primitives via errgroup
├── Dockerfile          multi-stage: dev, builder, production (Alpine)
├── compose.yaml        dev mode with docker compose watch
└── Makefile
```

## Development

| Command         | Description                      |
| --------------- | -------------------------------- |
| `make dev`      | Run the application              |
| `make build`    | Compile binary to `bin/template` |
| `make test`     | Run tests with race detector     |
| `make lint`     | Static analysis via `go vet`     |
| `make format`   | Format code                      |
| `make generate` | Run code generation (sqlc)       |

### Database

| Command             | Description                          |
| ------------------- | ------------------------------------ |
| `make db`           | Start postgres only (localhost:5432) |
| `make db-down`      | Stop postgres                        |
| `make migrate`      | Run migrations up                    |
| `make migrate-down` | Roll back last migration             |
| `make migrate-new`  | Create a new migration file          |

The postgres service is opt-in. `make db` starts it locally on `localhost:5432`. To use an external database instead, skip `make db` and set `DATABASE_URL` in `.env` to point at your instance.

### Docker

| Command      | Description                         |
| ------------ | ----------------------------------- |
| `make up`    | Start full stack (app + postgres)   |
| `make down`  | Stop all services                   |
| `make watch` | Hot reload via docker compose watch |

## Configuration

Configured via environment variables. Copy `.env.example` to `.env` to get started.

| Variable       | Required | Default | Description                      |
| -------------- | -------- | ------- | -------------------------------- |
| `DATABASE_URL` | Yes      | none    | PostgreSQL connection string     |
| `LOG_LEVEL`    | No       | `info`  | `debug`, `info`, `warn`, `error` |

## License

[Elastic License 2.0](LICENSE)
