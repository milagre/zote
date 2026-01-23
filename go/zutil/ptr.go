package zutil

func Ptr[T any](v T) *T {
	return &v
}
