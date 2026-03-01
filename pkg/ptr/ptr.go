// Package ptr provides utilities for working with pointers.
package ptr

// To creates a pointer to a value
func To[T any](v T) *T {
	return &v
}

// ValueOrEmpty returns the value of the pointer if it is not nil, otherwise it returns the empty string.
func ValueOrEmpty[T any](v *T) T {
	if v == nil {
		return *new(T)
	}
	return *v
}
