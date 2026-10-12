# Iteration and lazy processing

[Back to the Go entrypoint](../SKILL.md). Read the sections matching the current change; follow crosslinks when the change crosses a boundary.

- [Iterators](#go-iterators)

<a id="go-iterators"></a>

## Iterators

Use Go 1.23+ `iter.Seq` and `iter.Seq2` to make custom types rangeable when lazy consumption is useful. Returning a slice is often clearer when callers need a stable snapshot or repeated indexed access. Document ordering, whether iteration may repeat, how errors are reported, and whether values alias mutable state.

Stop immediately when `yield` returns false. Avoid holding a state lock while invoking arbitrary consumer work unless that blocking contract is intentional. A lazy iterator does not make concurrent map mutation safe; the simple map-backed examples assume no concurrent mutation during iteration.

```go
import "iter"

// Seq for single-value iteration
func (s *Set[E]) All() iter.Seq[E] {
    return func(yield func(E) bool) {
        for e := range s.m {
            if !yield(e) {
                return
            }
        }
    }
}

// Seq2 for key-value iteration
func (s *Store) All() iter.Seq2[string, Item] {
    return func(yield func(string, Item) bool) {
        for k, v := range s.items {
            if !yield(k, v) {
                return
            }
        }
    }
}

// Usage: range directly over custom types
for item := range mySet.All() {
    process(item)
}

for key, item := range store.All() {
    fmt.Println(key, item)
}
```

Compose iterators when avoiding intermediate slices makes the operation clearer or materially reduces allocation. Prefer a direct loop when a chain would obscure control flow or ownership.

```go
// Lazy filtering; no intermediate allocation
func FilterIter[V any](seq iter.Seq[V], pred func(V) bool) iter.Seq[V] {
    return func(yield func(V) bool) {
        for v := range seq {
            if pred(v) {
                if !yield(v) {
                    return
                }
            }
        }
    }
}

// Usage
for item := range FilterIter(mySet.All(), isActive) {
    process(item)
}
```

When collecting a sequence into a snapshot, preserve its [empty-result and ownership contract](types.md#go-collection-contracts); laziness does not establish deep isolation or a serialization shape.


Sources: [reference 1](https://pkg.go.dev/iter), [reference 2](https://go.dev/doc/go1.23#language).
