// Package worker provides generic bounded-concurrency primitives for
// processing items in parallel with context cancellation.
//
// Every function runs at most concurrency calls of fn at a time, stops
// starting new calls after the first failure or after ctx is canceled, and
// waits for every started call before returning. It returns the first error
// from fn, or ctx's error if ctx was canceled before all items were
// processed. fn must observe its context to stop early.
package worker

import (
	"context"
	"errors"
	"iter"
	"slices"
	"sync"

	"golang.org/x/sync/errgroup"
)

// ErrInvalidConcurrency is returned when concurrency is less than one.
var ErrInvalidConcurrency = errors.New("concurrency must be at least 1")

// Run calls fn for each item.
func Run[T any](
	ctx context.Context,
	items []T,
	concurrency int,
	fn func(context.Context, T) error,
) error {
	return RunIter(ctx, slices.Values(items), concurrency, fn)
}

// Map calls fn for each item and returns the results in input order. On
// error it returns a nil slice.
func Map[T, R any](
	ctx context.Context,
	items []T,
	concurrency int,
	fn func(context.Context, T) (R, error),
) ([]R, error) {
	results := make([]R, len(items))

	err := RunIter(ctx, enumerate(slices.Values(items)), concurrency, func(ctx context.Context, item indexedItem[T]) error {
		result, err := fn(ctx, item.value)
		if err != nil {
			return err
		}

		results[item.index] = result

		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

// RunIter calls fn for each item from seq. It stops pulling from seq after
// the first failure or cancellation.
func RunIter[T any](
	ctx context.Context,
	seq iter.Seq[T],
	concurrency int,
	fn func(context.Context, T) error,
) error {
	if concurrency < 1 {
		return ErrInvalidConcurrency
	}

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(concurrency)

	for item := range seq {
		if groupCtx.Err() != nil {
			break
		}

		group.Go(func() error {
			if err := groupCtx.Err(); err != nil {
				return err
			}

			return fn(groupCtx, item)
		})
	}

	if err := group.Wait(); err != nil {
		return err
	}

	return ctx.Err()
}

// MapIter calls fn for each item from seq and returns the results in the
// order seq produced the items. On error it returns a nil slice.
func MapIter[T, R any](
	ctx context.Context,
	seq iter.Seq[T],
	concurrency int,
	fn func(context.Context, T) (R, error),
) ([]R, error) {
	var (
		mu      sync.Mutex
		results []R
	)

	err := RunIter(ctx, enumerate(seq), concurrency, func(ctx context.Context, item indexedItem[T]) error {
		result, err := fn(ctx, item.value)
		if err != nil {
			return err
		}

		mu.Lock()
		defer mu.Unlock()

		if item.index >= len(results) {
			results = append(results, make([]R, item.index+1-len(results))...)
		}

		results[item.index] = result

		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

// indexedItem pairs an item with its position in the input.
type indexedItem[T any] struct {
	index int
	value T
}

// enumerate yields the items of seq with their positions.
func enumerate[T any](seq iter.Seq[T]) iter.Seq[indexedItem[T]] {
	return func(yield func(indexedItem[T]) bool) {
		index := 0

		for item := range seq {
			next := indexedItem[T]{
				index: index,
				value: item,
			}

			if !yield(next) {
				return
			}

			index++
		}
	}
}
