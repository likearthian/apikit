package api

// Middleware is a chainable behavior modifier for endpoints.
type Middleware[I, O any] func(Endpoint[I, O]) Endpoint[I, O]

// Chain is a helper function for composing middlewares. Requests will
// traverse them in the order they're declared. That is, the first middleware
// is treated as the outermost middleware.
func Chain[I, O any](outer Middleware[I, O], others ...Middleware[I, O]) Middleware[I, O] {
	return func(next Endpoint[I, O]) Endpoint[I, O] {
		for i := len(others) - 1; i >= 0; i-- { // reverse
			next = others[i](next)
		}
		return outer(next)
	}
}
