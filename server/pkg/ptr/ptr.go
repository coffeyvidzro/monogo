package ptr

// To returns a pointer to value.
func To[T any](value T) *T {
	return &value
}

// Deref returns the pointed-to value,
// or the type's zero value if value is nil.
func Deref[T any](value *T) T {
	if value == nil {
		var zero T
		return zero
	}

	return *value
}

// ValueOr returns the pointed-to value,
// or fallback if value is nil.
func ValueOr[T any](value *T, fallback T) T {
	if value == nil {
		return fallback
	}

	return *value
}

// Clone returns a pointer to a copy of value.
// It returns nil if value is nil.
func Clone[T any](value *T) *T {
	if value == nil {
		return nil
	}

	copy := *value

	return &copy
}

// Equal compares two pointers by nil state
// and pointed-to value.
func Equal[T comparable](left, right *T) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}

// Map returns a pointer to the transformed value.
// It returns nil if value is nil.
func Map[T any, R any](value *T, fn func(T) R) *R {
	if value == nil {
		return nil
	}

	result := fn(*value)

	return &result
}
