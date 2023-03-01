package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
)

type ServiceFunc[I any, O any] func(ctx context.Context, req I)

func MakeEndpoint[I any, O any](svc func(context.Context, I) O) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(I)
		if !ok {

		}
	}
}
