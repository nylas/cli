package util

// Map transforms a slice using a mapping function.
// Returns nil if the input slice is nil.
func Map[T, U any](s []T, f func(T) U) []U {
	if s == nil {
		return nil
	}
	result := make([]U, len(s))
	for i, v := range s {
		result[i] = f(v)
	}
	return result
}

// Filter returns a new slice containing only elements that satisfy the predicate.
// Returns nil if the input slice is nil.
func Filter[T any](s []T, keep func(T) bool) []T {
	if s == nil {
		return nil
	}
	result := make([]T, 0, len(s))
	for _, v := range s {
		if keep(v) {
			result = append(result, v)
		}
	}
	return result
}

// Reduce accumulates values using a reducer function.
// The reducer function takes an accumulator and the current element,
// and returns the new accumulator value.
func Reduce[T, U any](s []T, initial U, reduce func(U, T) U) U {
	acc := initial
	for _, v := range s {
		acc = reduce(acc, v)
	}
	return acc
}

// Contains checks if a slice contains a value.
// Uses == for comparison, so T must be comparable.
func Contains[T comparable](s []T, v T) bool {
	for _, item := range s {
		if item == v {
			return true
		}
	}
	return false
}

// Partition splits a slice into two based on a predicate.
// The first slice contains elements where predicate returns true,
// the second contains elements where it returns false.
// Returns (nil, nil) if the input slice is nil.
func Partition[T any](s []T, predicate func(T) bool) ([]T, []T) {
	if s == nil {
		return nil, nil
	}
	trueSlice := make([]T, 0, len(s)/2)
	falseSlice := make([]T, 0, len(s)/2)
	for _, v := range s {
		if predicate(v) {
			trueSlice = append(trueSlice, v)
		} else {
			falseSlice = append(falseSlice, v)
		}
	}
	return trueSlice, falseSlice
}

// Find returns the first element that satisfies the predicate.
// Returns the zero value of T and false if no element is found.
func Find[T any](s []T, predicate func(T) bool) (T, bool) {
	for _, v := range s {
		if predicate(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// Any returns true if any element satisfies the predicate.
func Any[T any](s []T, predicate func(T) bool) bool {
	for _, v := range s {
		if predicate(v) {
			return true
		}
	}
	return false
}

// All returns true if all elements satisfy the predicate.
// Returns true for empty slices.
func All[T any](s []T, predicate func(T) bool) bool {
	for _, v := range s {
		if !predicate(v) {
			return false
		}
	}
	return true
}
