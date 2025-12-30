package http

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func ChiURLParamIntoContext(ctx context.Context, r *http.Request) context.Context {
	params := make(map[string]string)
	if rctx := chi.RouteContext(ctx); rctx != nil {
		keys := rctx.URLParams.Keys
		values := rctx.URLParams.Values
		for idx, key := range keys {
			if key != "*" {
				params[key] = values[idx]
			}
		}
	}

	return context.WithValue(ctx, ContextKeyURLParams, params)
}
