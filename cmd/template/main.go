package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/samber/do/v2"

	"go-template/pkg/db"
	"go-template/pkg/env"
	_ "go-template/pkg/log" // structured logger init
	"go-template/pkg/template"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	i := do.New()
	env.Provide(i)
	db.Provide(i)
	template.Provide(i)

	if err := run(ctx, i); err != nil {
		slog.ErrorContext(ctx, "failed to run", "error", err)
		i.Shutdown() //nolint:errcheck
		os.Exit(1)
	}

	i.Shutdown() //nolint:errcheck
}

func run(ctx context.Context, i do.Injector) error {
	svc, err := do.Invoke[*template.Template](i)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	resp, err := svc.Example(ctx)
	if err != nil {
		return fmt.Errorf("example: %w", err)
	}

	slog.InfoContext(ctx, "example response", "response", resp)
	return nil
}
