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
