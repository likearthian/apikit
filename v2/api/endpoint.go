package api

import (
	"context"
)

type Endpoint[I, O any] func(context.Context, I) (O, error)

func (ep Endpoint[I, O]) Chain(outer Middleware[I, O], others ...Middleware[I, O]) Endpoint[I, O] {
	for i := len(others) - 1; i >= 0; i-- {
		ep = others[i](ep)
	}

	return outer(ep)
}

func Nop(context.Context, interface{}) (interface{}, error) { return struct{}{}, nil }

// Failer may be implemented by Go kit response types that contain business
// logic error details. If Failed returns a non-nil error, the Go kit transport
// layer may interpret this as a business logic error, and may encode it
// differently than a regular, successful response.
//
// It's not necessary for your response types to implement Failer, but it may
// help for more sophisticated use cases. The addsvc example shows how Failer
// should be used by a complete application.
type Failer interface {
	Failed() error
}
