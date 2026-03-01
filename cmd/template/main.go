// Package main is the entry point for the template application.
package main

import (
	"context"
	"log/slog"
	"os"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"go-template/pkg/db"
	"go-template/pkg/env"
	_ "go-template/pkg/log" // structured logger init
	"go-template/pkg/template"
)

func main() {
	app := fx.New(
		fx.WithLogger(func() fxevent.Logger { return fxevent.NopLogger }),
		env.Module,
		db.Module,
		template.Module,
		fx.Invoke(run),
	)

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		slog.Error("start failed", "error", err)
		os.Exit(1)
	}

	if err := app.Stop(ctx); err != nil {
		slog.Error("stop failed", "error", err)
		os.Exit(1)
	}
}

func run(t *template.Template) error {
	resp, err := t.Example(context.Background())
	if err != nil {
		return err
	}

	slog.Info("example response", "response", resp)
	return nil
}
