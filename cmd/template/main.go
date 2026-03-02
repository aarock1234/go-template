// Package main is the entry point for the template application.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/aarock1234/go-template/pkg/db"
	"github.com/aarock1234/go-template/pkg/env"
	_ "github.com/aarock1234/go-template/pkg/log" // structured logger init
	"github.com/aarock1234/go-template/pkg/template"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	config, err := env.New()
	if err != nil {
		slog.ErrorContext(ctx, "failed to load config", "error", err)
		os.Exit(1)
	}

	if err := run(ctx, config); err != nil {
		if cause := context.Cause(ctx); cause != nil {
			slog.ErrorContext(ctx, "shutting down", "cause", cause)
		} else {
			slog.ErrorContext(ctx, "failed to run", "error", err)
		}
		os.Exit(1)
	}
}

func run(ctx context.Context, config *env.Config) error {
	slog.InfoContext(ctx, "template application started")

	database, err := db.New(ctx, config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database: %w", err)
	}
	defer database.Close()

	example, err := template.New(database, nil)
	if err != nil {
		return fmt.Errorf("create template: %w", err)
	}

	resp, err := example.Example(ctx)
	if err != nil {
		return fmt.Errorf("example: %w", err)
	}

	slog.InfoContext(ctx, "example response", "response", resp)

	return nil
}
