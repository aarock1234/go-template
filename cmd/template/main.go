// Package main is the entry point for the template application.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http" // [server]
	"os"
	"os/signal"
	"syscall"
	"time" // [server]

	"github.com/go-chi/chi/v5" // [server]

	"github.com/aarock1234/go-template/pkg/db" // [postgres]
	"github.com/aarock1234/go-template/pkg/env"
	"github.com/aarock1234/go-template/pkg/handler"    // [server] [postgres]
	"github.com/aarock1234/go-template/pkg/middleware" // [server]
	"github.com/aarock1234/go-template/pkg/service"    // [server] [postgres]
	"github.com/aarock1234/go-template/pkg/template"   // [client] [postgres]

	_ "github.com/aarock1234/go-template/pkg/log" // structured logger init
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	config, err := env.New()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// [server]
	if err := serve(ctx, config); err != nil {
		if cause := context.Cause(ctx); cause != nil {
			slog.ErrorContext(ctx, "shutting down", slog.Any("cause", cause))

			return nil
		}

		return err
	}
	// [/server]
	// [client]
	if err := scrape(ctx, config); err != nil {
		if cause := context.Cause(ctx); cause != nil {
			slog.ErrorContext(ctx, "shutting down", slog.Any("cause", cause))

			return nil
		}

		return err
	}
	// [/client]

	return nil
}

// [server]

// serve starts the HTTP server with graceful shutdown.
func serve(ctx context.Context, config *env.Config) error {
	// [postgres]
	database, err := db.New(ctx, config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database: %w", err)
	}
	defer database.Close()

	svc := service.New(database)
	h := handler.New(svc)
	// [/postgres]

	r := chi.NewRouter()
	r.Use(middleware.Recover)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)

	// [postgres]
	r.Get("/health", handler.Health(database))
	r.Route("/api", func(r chi.Router) {
		r.Get("/example", h.GetExample)
	})
	// [/postgres]

	srv := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      r,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.InfoContext(ctx, "server listening", slog.String("addr", srv.Addr))
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen: %w", err)
		}
	case <-ctx.Done():
		slog.InfoContext(ctx, "shutting down server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
	}

	return nil
}

// [/server]

// [client]

// scrape runs the client-side scraper logic.
func scrape(ctx context.Context, config *env.Config) error {
	slog.InfoContext(ctx, "template application started")

	// [postgres]
	database, err := db.New(ctx, config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create database: %w", err)
	}
	defer database.Close()

	client, err := template.New(database, nil)
	if err != nil {
		return fmt.Errorf("create template: %w", err)
	}

	resp, err := client.Example(ctx)
	if err != nil {
		return fmt.Errorf("example: %w", err)
	}

	slog.InfoContext(ctx, "example response",
		slog.String("peetprint", resp.TLS.PeetPrint),
		slog.String("peetprint_hash", resp.TLS.PeetPrintHash),
	)
	// [/postgres]

	return nil
}

// [/client]
