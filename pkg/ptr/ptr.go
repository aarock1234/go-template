// Package ptr provides nil-safe pointer dereferencing and comparison
// helpers. To take the address of a value, use new(expr).
package ptr

// ValueOrZero returns the value p points to, or the zero value of T if p is
// nil.
func ValueOrZero[T any](p *T) T {
	if p == nil {
		var zero T

		return zero
	}

	return *p
}

// ValueOr returns the value p points to, or fallback if p is nil.
func ValueOr[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}

	return *p
}

// Equal reports whether a and b point to equal values. Two nil pointers are
// equal; a nil and a non-nil pointer are not.
func Equal[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return *a == *b
}
