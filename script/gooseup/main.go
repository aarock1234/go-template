// gooseup reads DATABASE_URL from an env file and runs goose migrations.
//
// Usage:
//
//	go run ./script/gooseup
//	go run ./script/gooseup -env .env.production
//	go run ./script/gooseup -dir package/db/migrations
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"go-template/package/env"
	_ "go-template/package/log" // structured logger init
)

var (
	envPath        = flag.String("env", ".env", "path to env file")
	migrationsPath = flag.String("dir", "package/db/migrations", "path to migrations directory")
)

type config struct {
	DatabaseURL string `env:"DATABASE_URL"`
}

func main() {
	flag.Parse()

	ctx := context.Background()

	if err := env.Load(*envPath); err != nil {
		slog.ErrorContext(ctx, "failed to load env file", "error", err)
		os.Exit(1)
	}

	var cfg config
	if err := env.Validate(&cfg); err != nil {
		slog.ErrorContext(ctx, "failed to validate environment", "error", err)
		os.Exit(1)
	}

	if err := run(ctx, cfg, *migrationsPath); err != nil {
		slog.ErrorContext(ctx, "failed to run", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config, migrationsPath string) error {
	if info, err := os.Stat(migrationsPath); err != nil || !info.IsDir() {
		return fmt.Errorf("migrations directory %q does not exist", migrationsPath)
	}

	cmd := exec.CommandContext(ctx, "goose", "-dir", migrationsPath, "postgres", cfg.DatabaseURL, "up")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}
