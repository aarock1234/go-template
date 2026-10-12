# syntax=docker/dockerfile:1

# base: toolchain, downloaded modules, and source
FROM --platform=$BUILDPLATFORM golang:1.27 AS base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# dev: builds from source on every start so docker compose watch can restart
# it; exec hands signals to the app for graceful shutdown
FROM base AS dev

CMD ["sh", "-c", "go build -o /tmp/template ./cmd/template && exec /tmp/template"]

# builder: cross-compiles a static binary for the target platform
FROM base AS builder

ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /bin/template ./cmd/template

# production: minimal runtime image running as an unprivileged user
FROM alpine:3.24 AS production

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/template /bin/template

USER 65534:65534

ENTRYPOINT ["/bin/template"]
