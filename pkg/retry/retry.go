// Package retry provides a generic retry loop with exponential backoff and
// full jitter for transient-error recovery.
//
// Retry only operations that are safe to repeat: a failed attempt does not
// prove that its side effects did not happen. Use WithRetryIf to stop on
// errors that are not transient.
package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"time"
)

const (
	defaultMaxAttempts  = 3
	defaultInitialDelay = 1 * time.Second
	defaultMaxDelay     = 10 * time.Second
	defaultMultiplier   = 2.0
)

// Option configures the retry behavior.
type Option func(*config)

type config struct {
	maxAttempts  int
	initialDelay time.Duration
	maxDelay     time.Duration
	multiplier   float64
	retryIf      func(error) bool
}

// WithMaxAttempts sets the maximum number of attempts, including the first.
// A value of 1 means no retries. Default: 3.
func WithMaxAttempts(n int) Option {
	return func(c *config) { c.maxAttempts = n }
}

// WithInitialDelay sets the base delay before the first retry. Default: 1s.
func WithInitialDelay(d time.Duration) Option {
	return func(c *config) { c.initialDelay = d }
}

// WithMaxDelay caps the backoff delay. Default: 10s.
func WithMaxDelay(d time.Duration) Option {
	return func(c *config) { c.maxDelay = d }
}

// WithMultiplier sets the exponential growth factor, which must be at
// least 1. Default: 2.
func WithMultiplier(m float64) Option {
	return func(c *config) { c.multiplier = m }
}

// WithRetryIf retries only errors for which retryable returns true; any
// other error is returned immediately. By default every error is retried.
func WithRetryIf(retryable func(error) bool) Option {
	return func(c *config) { c.retryIf = retryable }
}

func (c *config) validate() error {
	switch {
	case c.maxAttempts < 1:
		return errors.New("retry: max attempts must be at least 1")
	case c.initialDelay < 0:
		return errors.New("retry: initial delay must not be negative")
	case c.maxDelay < c.initialDelay:
		return errors.New("retry: max delay must not be less than initial delay")
	case math.IsNaN(c.multiplier) || math.IsInf(c.multiplier, 0) || c.multiplier < 1:
		return errors.New("retry: multiplier must be finite and at least 1")
	default:
		return nil
	}
}

// Do calls fn until it returns nil, it returns an error that is not
// retryable, ctx is canceled, or the maximum number of attempts is reached.
// Between attempts it sleeps for a random duration in
// [0, min(initialDelay*multiplier^retry, maxDelay)).
//
// On exhaustion the error wraps the last failure. On cancellation it wraps
// both ctx's error and the last failure, so errors.Is reports either.
func Do(ctx context.Context, fn func(ctx context.Context) error, opts ...Option) error {
	cfg := config{
		maxAttempts:  defaultMaxAttempts,
		initialDelay: defaultInitialDelay,
		maxDelay:     defaultMaxDelay,
		multiplier:   defaultMultiplier,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	if err := cfg.validate(); err != nil {
		return err
	}

	var lastErr error

	for attempt := range cfg.maxAttempts {
		if err := ctx.Err(); err != nil {
			return canceled(err, lastErr)
		}

		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}

		if cfg.retryIf != nil && !cfg.retryIf(lastErr) {
			return lastErr
		}

		if attempt == cfg.maxAttempts-1 {
			break
		}

		if err := sleep(ctx, backoff(attempt, cfg)); err != nil {
			return canceled(err, lastErr)
		}
	}

	return fmt.Errorf("retry: after %d attempts: %w", cfg.maxAttempts, lastErr)
}

// canceled reports that retrying stopped because of ctxErr.
func canceled(ctxErr, lastErr error) error {
	if lastErr == nil {
		return ctxErr
	}

	return fmt.Errorf("retry: %w: last error: %w", ctxErr, lastErr)
}

// sleep waits for d or until ctx is canceled.
func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// backoff returns the full-jitter delay before the retry that follows the
// given zero-based attempt.
func backoff(attempt int, cfg config) time.Duration {
	ceiling := cfg.maxDelay

	// Comparing as floats keeps an exponential overflow from wrapping.
	if exponential := float64(cfg.initialDelay) * math.Pow(cfg.multiplier, float64(attempt)); exponential < float64(cfg.maxDelay) {
		ceiling = time.Duration(exponential)
	}

	if ceiling <= 0 {
		return 0
	}

	return time.Duration(rand.Int64N(int64(ceiling)))
}
