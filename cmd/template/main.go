// Package main is the entry point for the template application.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"      // [server]
	"net/http" // [server]
	"os"
	"os/signal"
	"syscall"
	"time" // [server]

	"github.com/go-chi/chi/v5" // [server]
	"golang.org/x/sync/errgroup"

	"github.com/aarock1234/go-template/pkg/client" // [client]
	"github.com/aarock1234/go-template/pkg/db"     // [postgres]
	"github.com/aarock1234/go-template/pkg/env"
	"github.com/aarock1234/go-template/pkg/handler" // [server]
	"github.com/aarock1234/go-template/pkg/log"
	"github.com/aarock1234/go-template/pkg/middleware" // [server]
	"github.com/aarock1234/go-template/pkg/service"    // [server] [postgres]
	"github.com/aarock1234/go-template/pkg/template"   // [client]
)

// Config holds the environment configuration for this process.
type Config struct {
	LogLevel slog.Level `env:"LOG_LEVEL,default=info"`
	// [client]
	Proxy string `env:"PROXY"`
	// [/client]
	// [postgres]
	DatabaseURL string `env:"DATABASE_URL,required"`
	// [/postgres]
	// [server]
	Port            string        `env:"PORT,default=8080"`
	ReadTimeout     time.Duration `env:"READ_TIMEOUT,default=10s"`
	WriteTimeout    time.Duration `env:"WRITE_TIMEOUT,default=10s"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT,default=10s"`
	// [/server]
}

// app holds the dependencies shared by the process's workloads.
type app struct {
	config *Config
	// [postgres]
	database *db.DB
	// [/postgres]
}

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", slog.Any("error", err))
		os.Exit(1)
	}
}

// run loads configuration, opens shared dependencies, and runs every
// workload until all of them finish, one of them fails, or the process
// receives an interrupt or termination signal.
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	config, err := env.New[Config]()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log.SetLevel(config.LogLevel)

	app := &app{
		config: config,
	}

	// [postgres]
	database, err := db.New(ctx, config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer database.Close()

	app.database = database
	// [/postgres]

	// The first workload to fail cancels the others. Each workload returns
	// nil when it stops because the context was canceled.
	group, ctx := errgroup.WithContext(ctx)

	// [server]
	group.Go(func() error {
		return serve(ctx, app)
	})
	// [/server]

	// [client]
	group.Go(func() error {
		return scrape(ctx, app)
	})
	// [/client]

	return group.Wait()
}

// [server]

// serve runs the HTTP server until ctx is canceled, then shuts it down
// within the configured timeout. A listener or shutdown failure is returned
// even when it happens during cancellation.
func serve(ctx context.Context, app *app) error {
	server := &http.Server{
		Addr:         net.JoinHostPort("", app.config.Port),
		Handler:      routes(app),
		ReadTimeout:  app.config.ReadTimeout,
		WriteTimeout: app.config.WriteTimeout,
	}

	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(ctx, "tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	slog.InfoContext(ctx, "server listening", slog.String("addr", listener.Addr().String()))

	serveResult := make(chan error, 1)

	go func() {
		serveResult <- server.Serve(listener)
	}()

	var serveErr error
	serveFinished := false

	select {
	case err := <-serveResult:
		serveErr = serverError(err)
		serveFinished = true
	case <-ctx.Done():
		slog.InfoContext(ctx, "shutting down server", slog.Any("cause", context.Cause(ctx)))
	}

	shutdownErr := shutdownServer(server, app.config.ShutdownTimeout)

	if !serveFinished {
		serveErr = serverError(<-serveResult)
	}

	return errors.Join(serveErr, shutdownErr)
}

// routes builds the HTTP handler tree. Middleware order matters: the
// request ID is assigned first so the logger and panic recovery can use it,
// and recovery runs inside the logger so recovered panics are logged as 500s.
func routes(app *app) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recover)

	router.Get("/livez", handler.Live)
	// [postgres]
	router.Get("/readyz", handler.Ready(app.database))

	example := handler.New(service.New(app.database))

	router.Route("/api", func(r chi.Router) {
		r.Get("/example", example.GetExample)
	})
	// [/postgres]

	return router
}

// shutdownServer stops the server gracefully, forcing open connections
// closed when the timeout expires first. It uses a fresh context because
// the process context is already canceled.
func shutdownServer(server *http.Server, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		closeErr := server.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf("close server: %w", closeErr)
		}

		return errors.Join(fmt.Errorf("shutdown server: %w", err), closeErr)
	}

	return nil
}

// serverError converts the result of Serve into an error, treating the
// expected result of a shutdown as success.
func serverError(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return fmt.Errorf("serve: %w", err)
}

// [/server]

// [client]

// scrape runs the client workload once: it fetches the fingerprint observed
// over HTTP/1.1, then over HTTP/2. It returns nil if ctx is canceled before
// the work completes.
func scrape(ctx context.Context, app *app) error {
	proxy, err := client.ParseProxy(app.config.Proxy, client.ProxySchemeHTTP)
	if err != nil {
		return fmt.Errorf("parse proxy: %w", err)
	}

	scraper, err := template.New(proxy)
	if err != nil {
		return fmt.Errorf("create scraper: %w", err)
	}
	defer scraper.Close()

	for _, protocol := range []template.Protocol{template.HTTP1, template.HTTP2} {
		network, err := scraper.Network(ctx, protocol)
		if err != nil {
			if ctx.Err() != nil && errors.Is(err, context.Canceled) {
				return nil
			}

			return err
		}

		slog.InfoContext(ctx, "network fingerprint",
			slog.String("protocol", string(protocol)),
			slog.Any("network", network),
		)
	}

	return nil
}

// [/client]
