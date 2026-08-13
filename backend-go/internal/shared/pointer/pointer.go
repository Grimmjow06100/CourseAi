// Package pointer provides small, type-safe helpers for optional scalar values.
package pointer

// To returns a pointer to a copy of value.
func To[T any](value T) *T {
	return &value
}

// Clone returns a shallow copy of value or nil when value is nil.
func Clone[T any](value *T) *T {
	if value == nil {
		return nil
	}
	return To(*value)
}

// Equal compares two optional comparable values.
func Equal[T comparable](left, right *T) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

// Map transforms an optional value while preserving nil.
func Map[T, U any](value *T, transform func(T) U) *U {
	if value == nil {
		return nil
	}
	return To(transform(*value))
}
