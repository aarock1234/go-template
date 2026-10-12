# Application lifetime and shutdown

[Back to the Go entrypoint](../SKILL.md). Read the sections matching the current change; follow crosslinks when the change crosses a boundary.

- [Graceful Shutdown](#go-shutdown)
- [Persistence guarantees and file ownership](#go-persistence-contracts)

<a id="go-shutdown"></a>

### Graceful Shutdown

Production servers should handle process signals, stop accepting work, and allow in-flight work to finish within a deadline. Go 1.26+ records the received signal as the cancellation cause from `signal.NotifyContext`, available through `context.Cause`.

Give the listener an owner that receives its result. A listener failure must end startup or service execution rather than leave the process waiting forever for a signal. Use a fresh bounded context for shutdown because the process context is already canceled. Keep `os.Exit` in `main` so `run` can finish its deferred cleanup.

```go
func main() {
	if err := run(); err != nil {
		slog.Error("server failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return serve(ctx, server)
}

func serve(ctx context.Context, server *http.Server) error {
	listenerResult := make(chan error, 1)

	go func() {
		listenerResult <- server.ListenAndServe()
	}()

	var listenErr error
	listenerFinished := false

	select {
	case err := <-listenerResult:
		listenErr = listenerError(err)
		listenerFinished = true
	case <-ctx.Done():
		slog.InfoContext(ctx, "shutting down", slog.Any("cause", context.Cause(ctx)))
	}

	shutdownErr := shutdownServer(server)

	if !listenerFinished {
		listenErr = listenerError(<-listenerResult)
	}

	return errors.Join(listenErr, shutdownErr)
}

func shutdownServer(server *http.Server) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		closeErr := server.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf("close server: %w", closeErr)
		}

		return errors.Join(fmt.Errorf("shutdown server: %w", err), closeErr)
	}

	return nil
}

func listenerError(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return fmt.Errorf("listen: %w", err)
}
```

`mux` represents the application's configured handler. In this process-owned example, ordinary signal shutdown succeeds; a listener failure or shutdown timeout remains an error even if the process context is canceled. The explicit `Close` fallback gives up waiting for ordinary active connections after the shutdown deadline. Background workers and hijacked connections such as WebSockets need their own cancellation and join policy.

The original example logged listener and shutdown failures without returning them. Its listener goroutine could exit while the main goroutine waited indefinitely for a signal. The result channel and returned failures make that lifetime and outcome explicit.

Both exit triggers run bounded server cleanup before returning. `Close` terminates ordinary connections; it does not join application goroutines. If dependencies must survive unfinished handlers after a forced close, the application needs a separate handler completion policy. Keep separate ownership for background workers and hijacked connections.


Sources: [reference 1](https://pkg.go.dev/net/http@go1.26.1#Server.Shutdown), [reference 2](https://pkg.go.dev/os/signal@go1.26.1#NotifyContext).

<a id="go-persistence-contracts"></a>

## Persistence guarantees and file ownership

State the durability and concurrency contract before choosing a persistence pattern. A process lock, a goroutine mutex, successful writes, and crash-safe replacement solve different problems. File locking between processes does not serialize methods sharing one handle inside a process, especially when those methods mutate a shared file offset.

Truncating a state file before writing exposes the previous data to loss if a later write fails or the process stops. That may be acceptable for a rebuildable cache; it is usually a poor implicit contract for authoritative state. Choose an appropriate transactional storage mechanism or a replacement protocol that preserves the old state until the new state is complete.

Do not assume `os.Rename` is atomic on every supported operating system. A rename-based design also needs a locking strategy that survives replacement: retaining a lock on the old file does not automatically lock the replacement file. Define the guarantees for the actual filesystem and platforms rather than labeling any temporary-file pattern “atomic.”

Document who closes the handle, whether concurrent methods are allowed, and whether success promises only visibility or also persistence across a crash. Check the relevant write, flush, sync, commit, and close outcomes for that promise. Read [resource cleanup](errors.md#go-close-errors) before deciding which errors can be discarded.

The template's `pkg/state/` shows a generic file handle that holds a platform-specific lock on a companion lock file and saves by flushing a temporary file and renaming it over the state file. Inspect its current implementation against the required guarantee before adopting it for state that cannot be reconstructed.

Database-backed persistence has its own [transaction and query contracts](data.md); file replacement rules do not substitute for those guarantees.


Sources: [reference 1](https://pkg.go.dev/os#Rename), [reference 2](https://pkg.go.dev/os#File.Sync), [reference 3](https://pkg.go.dev/os#File.Close).
