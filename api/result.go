package api

type Result[T any] struct {
	err   error
	value T
}
