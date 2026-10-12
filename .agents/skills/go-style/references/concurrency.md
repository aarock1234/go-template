# Workers, synchronization, and joining

[Back to the Go entrypoint](../SKILL.md). Read the sections matching the current change; follow crosslinks when the change crosses a boundary.

- [Concurrency Patterns](#go-concurrency)
- [Worker Pool](#go-worker-pool)
- [Parallel Fetch with Result Channels](#go-result-channels)
- [Errgroup for Concurrent Operations](#go-errgroup)
- [Thread-Safe State](#go-shared-state)
- [Retry contracts and bounded attempts](#go-retry-contracts)

<a id="go-concurrency"></a>

## Concurrency Patterns

Prefer synchronization primitives over channels for simple mutual exclusion. Channels are for communication and orchestration; `sync.Mutex` is for protecting shared state. Using the simpler tool makes intent clearer.

**When to use which pattern:**

- **errgroup**: default choice for a finite group of concurrent operations with shared error handling; use `SetLimit` to bound active work
- **Worker pool**: when persistent workers, queue ownership, or streaming input make explicit workers useful; both a pool and errgroup can impose concurrency limits
- **Result channels**: when goroutines produce heterogeneous values you need to collect individually

For each goroutine, identify who starts it, what ends it, and who waits for it. Cancellation is a request to stop, not a join. Define whether the operation stops at its first failure or deliberately completes best-effort work, and state what its return value means.


Sources: [reference 1](https://pkg.go.dev/golang.org/x/sync@v0.20.0/errgroup), [reference 2](https://go.dev/wiki/CodeReviewComments#goroutine-lifetimes).

<a id="go-worker-pool"></a>

### Worker Pool

This Go 1.25+ example uses `WaitGroup.Go`. On older baselines, call `Add` before launching each goroutine and defer `Done` inside it. Functions passed to `WaitGroup.Go` must not panic.

```go
func (s *Service) ProcessBatch(ctx context.Context, items []Item) error {
    const concurrency = 10
    taskCh := make(chan Item, concurrency*2)
    var wg sync.WaitGroup

    // Start workers
    for range concurrency {
        wg.Go(func() {
            for item := range taskCh {
                if ctx.Err() != nil {
                    return
                }

                if err := s.process(ctx, item); err != nil {
                    s.logger.ErrorContext(ctx, "process failed", slog.Any("error", err))
                }
            }
        })
    }

    // Send work
    wg.Go(func() {
        defer close(taskCh)
        for _, item := range items {
            select {
            case <-ctx.Done():
                return
            case taskCh <- item:
            }
        }
    })

    wg.Wait()

    return ctx.Err()
}

// Note: individual item errors are logged, not returned. For operations
// where errors must be collected, prefer errgroup or aggregate into a slice.
```

The owner waits for both workers and the producer. The producer alone closes the task channel, and workers stop taking new items after observing cancellation. A nil return means the best-effort pass ended without context cancellation; it does not mean every item succeeded.


Sources: [reference 1](https://pkg.go.dev/sync#WaitGroup.Go), [reference 2](https://pkg.go.dev/context).

<a id="go-result-channels"></a>

### Parallel Fetch with Result Channels

Result channels make independently named results explicit. Pass cancellation to both fetches, keep each single-result channel buffered, and wait for the producers before returning. The fetch functions must observe their context and return a non-nil response on success. This example uses Go 1.25+ `WaitGroup.Go`.

```go
func (s *Service) FetchBoth(ctx context.Context, id string) (*Data, error) {
    fetchCtx, cancel := context.WithCancel(ctx)

    type result struct {
        data *Response
        err  error
    }

    ch1 := make(chan result, 1)
    ch2 := make(chan result, 1)

    var workers sync.WaitGroup

    defer func() {
        cancel()
        workers.Wait()
    }()

    workers.Go(func() {
        data, err := s.fetchOne(fetchCtx, id)
        ch1 <- result{
            data: data,
            err:  err,
        }
    })

    workers.Go(func() {
        data, err := s.fetchTwo(fetchCtx, id)
        ch2 <- result{
            data: data,
            err:  err,
        }
    })

    var first, second *Response

    for range 2 {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case received := <-ch1:
            if received.err != nil {
                return nil, received.err
            }

            first = received.data
        case received := <-ch2:
            if received.err != nil {
                return nil, received.err
            }

            second = received.data
        }
    }

    return &Data{
        One: *first,
        Two: *second,
    }, nil
}
```

The original example accepted a context without using it. Buffering a result channel prevents its final send from blocking; it does not cancel the network operation or prove its goroutine has exited. For a simple parallel fetch with shared failure handling, errgroup is often clearer. Keep result channels when their collection semantics earn the extra coordination.


Sources: [reference 1](https://pkg.go.dev/context), [reference 2](https://go.dev/wiki/CodeReviewComments#goroutine-lifetimes), [reference 3](https://pkg.go.dev/sync#WaitGroup.Go).

<a id="go-errgroup"></a>

### Errgroup for Concurrent Operations

Use a positive limit for resource-sensitive work. An errgroup without a limit is unbounded; zero prevents new work from starting and a negative limit removes the bound. Set the limit before starting work and keep it fixed while goroutines are active.

```go
import "golang.org/x/sync/errgroup"

func (s *Service) ProcessAll(ctx context.Context, items []Item, concurrency int) error {
    if concurrency < 1 {
        return errors.New("concurrency must be positive")
    }

    group, workCtx := errgroup.WithContext(ctx)
    group.SetLimit(concurrency)

    for _, item := range items {
        if workCtx.Err() != nil {
            break
        }

        group.Go(func() error {
            if err := workCtx.Err(); err != nil {
                return err
            }

            return s.process(workCtx, item)
        })
    }

    if err := group.Wait(); err != nil {
        return err
    }

    return ctx.Err()
}
```

`Go` can block until an active slot is free. Cancellation does not forcibly stop an active operation; each operation must cooperate. This example stops enumerating after observing cancellation and joins all started work. Its explicit success policy reports parent cancellation even if no task returned an error.

The derived `workCtx` is canceled when the first task fails or when `Wait` returns, including success. Use the parent context for unrelated work after the group. The earlier example launched one goroutine per item despite being described as bounded; the explicit limit corrects that mismatch.


Sources: [reference 1](https://pkg.go.dev/golang.org/x/sync@v0.20.0/errgroup).

<a id="go-shared-state"></a>

### Thread-Safe State

```go
type Cycle[T any] struct {
    items []T
    index int
    mu    sync.Mutex
}

func (c *Cycle[T]) Next() (T, bool) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if len(c.items) == 0 {
        var zero T
        return zero, false
    }

    item := c.items[c.index]
    c.index = (c.index + 1) % len(c.items)

    return item, true
}
```


<a id="go-retry-contracts"></a>

## Retry contracts and bounded attempts

Retry only when the operation's contract permits repetition. A temporary failure does not prove that a remote side effect did not already happen. Require an idempotent operation or a server-supported deduplication key before automatically repeating a mutation. Decide which failures are transient; authentication failures, invalid input, and canceled work ordinarily need a different response.

Validate retry configuration before starting: attempts must be positive, delay bounds must be nonnegative and ordered, and the multiplier must be finite and appropriate for the chosen backoff. Avoid duration conversion or arithmetic that can overflow a random-number bound. State whether “attempts” includes the initial call; the template uses that meaning.

Check cancellation before the first attempt and before each subsequent call. A cancellable delay does not prevent another attempt if cancellation arrived before invoking the callback. Pass the operation context through each attempt and keep a total time budget; creating a fresh unlimited context per attempt defeats the caller's deadline.

Jitter reduces synchronized retries, while a concurrency limit bounds active calls. Those are separate controls. On exhaustion, preserve the final meaningful error with attempt context. When cancellation stops the retry loop, return an outcome that callers can recognize as cancellation under the function's documented contract.

Read [context propagation](context.md#go-context) for lifecycle ownership and [error propagation](errors.md#go-error-propagation) for public error contracts. `pkg/retry/retry.go` in the template is a structural example to inspect, not permission to retry every callback indiscriminately.


Sources: [reference 1](https://pkg.go.dev/net/http@go1.26.1#Transport), [reference 2](https://pkg.go.dev/context), [reference 3](https://pkg.go.dev/math/rand/v2#Int64N).
