// Package ptr contains utilities to deal with pointers
package ptr

// NewPtr creates a new pointer to the value provided in input
func NewPtr[T any](v T) *T {
	return &v
}

func Int64(value int64) *int64 {
	return &value
}
