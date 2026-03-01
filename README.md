# go-template

A minimal Go service template.

## Getting started

Copy `.env.example` to `.env` and set your values, then run:

```sh
go run ./cmd/...
```

## Database

The template reads `DATABASE_URL` from `.env`. Point it at any postgres instance you want, local or external.

If you want a local postgres spun up via Docker, use the bundled compose profile:

```sh
docker compose --profile postgres up postgres
```

The container exposes port `5432` on `localhost`, so tools running on the host (goose, psql, the app itself) connect normally using `localhost:5432`.

To run the full stack in Docker:

```sh
docker compose --profile postgres up
```

If you already have postgres running elsewhere, skip the profile and just set `DATABASE_URL` in `.env` to point at it.
