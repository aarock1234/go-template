# go-template

![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-Elastic_2.0-blue)
![CI](https://github.com/aarock1234/go-template/actions/workflows/ci.yaml/badge.svg)

An opinionated Go project template for scraper, bot, and service workloads. Batteries included: a TLS-fingerprinted HTTP client via [`fphttp`](https://github.com/aarock1234/fphttp), an HTTP server with graceful shutdown, exponential backoff, bounded concurrency, crash-safe file-backed state, PostgreSQL, and structured logging.

## Quick Start

Install [`gonew`](https://pkg.go.dev/golang.org/x/tools/cmd/gonew):

```bash
go install golang.org/x/tools/cmd/gonew@latest
```

Scaffold a new project, which clones the template and rewrites all import paths to your module name:

```bash
gonew github.com/aarock1234/go-template@latest github.com/you/myproject
```

Then choose the features you need. Setup removes the rest, strips its markers, tidies modules, and deletes itself:

```bash
make setup                                          # interactive
make setup ARGS="-y"                                # accept every default
make setup ARGS="-y -server=false -packages=retry"  # scraper only, keep pkg/retry
```

Flags: `-server`, `-client`, `-docker`, `-postgres` (default `true`), `-postgres-hosting` (`docker` or `external`), `-packages` (comma-separated: `worker`, `retry`, `state`, `cycle`, `fake`, `ptr`), and `-y` to skip the form.

## Setup

**Prerequisites:** [Go 1.27+](https://go.dev/dl/), and [Docker](https://docs.docker.com/get-docker/) for the container and database targets. Tools such as golangci-lint, goose, and sqlc run through `go run` at pinned versions.

1. Replace `pkg/template` with your own domain logic
2. Wire your services in `cmd/template/main.go`
3. Add migrations with `make migrate-new name=<name>`, queries to `pkg/db/queries/`, then run `make generate`
4. Copy and configure your environment:

```bash
cp .env.example .env
```

## Running

Locally:

```bash
make db    # optional: start PostgreSQL on localhost:5432
make dev
```

With Docker Compose, which also points the app at the bundled database:

```bash
make up     # build and start in the background
make watch  # or: hot reload on file changes
```

The entrypoint runs each enabled workload concurrently: the HTTP server until it receives `SIGINT` or `SIGTERM`, and the client workload once. If either fails, the other is stopped and the process exits non-zero.

## Project Structure

```
go-template/
├── .agents/skills/     go-style coding guide for AI agents
├── cmd/
│   ├── setup/          interactive feature selection (deletes itself)
│   └── template/       entrypoint: config, dependencies, workloads
├── pkg/
│   ├── apperr/         application error types and sentinels
│   ├── client/         fphttp client: TLS/HTTP2 fingerprints, cookies, redirects, proxies
│   ├── cycle/          thread-safe round-robin file-backed line rotation
│   ├── db/             PostgreSQL pool, sqlc queries, transactions, advisory locks
│   ├── env/            .env loader and struct-tag config parsing
│   ├── fake/           fake data generation
│   ├── handler/        HTTP handlers, liveness and readiness probes
│   ├── log/            slog with tint and request-scoped context values
│   ├── middleware/     request ID, request logging, panic recovery
│   ├── ptr/            nil-safe pointer helpers
│   ├── respond/        JSON response helpers
│   ├── retry/          exponential backoff with full jitter
│   ├── service/        business logic behind consumer-defined interfaces
│   ├── state/          file-backed JSON state with locking and atomic saves
│   ├── template/       example client workload (replace this)
│   └── worker/         bounded-concurrency primitives via errgroup
├── Dockerfile          multi-stage: dev, builder (multi-arch), production (Alpine, non-root)
├── compose.yaml        app and PostgreSQL with docker compose watch
└── Makefile            run `make` to list targets
```

## Development

| Command         | Description                                  |
| --------------- | -------------------------------------------- |
| `make dev`      | Run the application                          |
| `make build`    | Compile the binary to `bin/template`         |
| `make lint`     | Static analysis via golangci-lint            |
| `make format`   | Format with gofmt and goimports              |
| `make fix`      | Modernize code via `go fix`                  |
| `make test`     | Run tests with the race detector             |
| `make generate` | Run code generation (`sqlc generate`)        |

### Database

| Command                       | Description                            |
| ----------------------------- | -------------------------------------- |
| `make db`                     | Start PostgreSQL only (localhost:5432) |
| `make db-down`                | Stop PostgreSQL                        |
| `make migrate`                | Apply migrations                       |
| `make migrate-down`           | Roll back the last migration           |
| `make migrate-new name=<name>` | Create a migration file               |

To use an external database, choose external hosting during setup and set `DATABASE_URL` in `.env`.

### Docker

| Command      | Description                         |
| ------------ | ----------------------------------- |
| `make up`    | Build and start the stack           |
| `make down`  | Stop the stack                      |
| `make watch` | Hot reload via docker compose watch |

### HTTP Endpoints

| Route              | Description                                 |
| ------------------ | ------------------------------------------- |
| `GET /livez`       | Liveness: the process is serving requests   |
| `GET /readyz`      | Readiness: the database answers a ping      |
| `GET /api/example` | Example handler backed by a sqlc query      |

### Releases

Pushing a `v*.*.*` tag runs the CI checks and publishes a GitHub Release with generated notes:

```bash
git tag v1.2.3
git push origin v1.2.3
```

## Configuration

Configured via environment variables, loaded from `.env` when present. Process environment variables take precedence.

| Variable           | Required | Default | Description                                       |
| ------------------ | -------- | ------- | ------------------------------------------------- |
| `LOG_LEVEL`        | No       | `info`  | `debug`, `info`, `warn`, `error`                  |
| `PROXY`            | No       | none    | Client proxy: `host:port` or `host:port:user:pass` |
| `DATABASE_URL`     | Yes      | none    | PostgreSQL connection string                      |
| `PORT`             | No       | `8080`  | HTTP server port                                  |
| `READ_TIMEOUT`     | No       | `10s`   | Server read timeout                               |
| `WRITE_TIMEOUT`    | No       | `10s`   | Server write timeout                              |
| `SHUTDOWN_TIMEOUT` | No       | `10s`   | Graceful shutdown deadline                        |

The HTTP client resolves the latest catalogued Chrome identity on Windows and
verifies TLS certificates. Select a product, platform, and version with
`client.WithBrowser`, `client.WithPlatform`, and `client.WithBrowserVersion`.
Version prefixes resolve matching wire profiles as well as headers. Unknown or
unavailable builds return errors; inspect `Descriptor()` for availability and
configuration sources. `client.WithFingerprint` and `client.WithClientHelloID` remain explicit
manual overrides. `client.WithProfile` accepts a resolved profile for selecting
channels, devices, and explicit configuration variants. Manual overrides can use
known published identity metadata even when its catalog wire definition is
unavailable.

`Do` generates basic identity headers. Use `DoWithOptions` with fphttp's
`browser.RequestOptions` for explicit navigation, fetch, and resource context.
The browser client owns redirects, contextual headers, hint restarts, and
response decoding. The template retains its cookie jar, cookie serializer,
custom extractor, and proxy configuration. `ClearCookies` also clears learned
client hints. `client.WithDisableHeaders(true)` disables generated fields;
explicit headers and configured overrides still apply.

Enable `client.WithInsecureSkipVerify(true)` only for targets that require it.

## Style Guide

Code follows the `go-style` skill in [`.agents/skills/go-style`](.agents/skills/go-style/SKILL.md), which agents such as Codex, Cursor, and Gemini discover automatically.

## License

[Elastic License 2.0](LICENSE)
