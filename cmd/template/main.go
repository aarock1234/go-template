package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go-template/pkg/db"
	"go-template/pkg/env"
	_ "go-template/pkg/log" // structured logger init
	"go-template/pkg/template"
)

type config struct {
	DatabaseURL string `env:"DATABASE_URL"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := env.Load(); err != nil {
		slog.ErrorContext(ctx, "failed to load environment variables", "error", err)
		os.Exit(1)
	}

	var cfg config
	if err := env.Validate(&cfg); err != nil {
		slog.ErrorContext(ctx, "failed to validate environment", "error", err)
		os.Exit(1)
	}

	if err := run(ctx, cfg); err != nil {
		slog.ErrorContext(ctx, "failed to run", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config) error {
	slog.InfoContext(ctx, "template application started")

	database, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer database.Close()

	example, err := template.New(database, nil)
	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	resp, err := example.Example(ctx)
	if err != nil {
		return fmt.Errorf("failed to get example: %w", err)
	}

	slog.InfoContext(ctx, "example response", "response", resp)

	return nil
}
