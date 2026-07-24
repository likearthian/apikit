# apikit

## Versions

- **v2** (recommended for new projects) — see [`v2/`](v2/) and the [Migration Guide](v2/MIGRATION.md)
- **v1** (legacy support) — this directory. Maintained through at least 2027.

Import paths:

| Version | Module path |
|---------|-------------|
| v2 | `github.com/likearthian/apikit/v2` |
| v1 | `github.com/likearthian/apikit` |

---

`apikit` is a small, opinionated Go library for building HTTP services with
type-safe, go-kit-style endpoints. It provides generic endpoint and middleware
primitives, HTTP transports with decode/encode, request binding, JWT and API-key
authentication, structured-logging adapters, and (v2) platform packages for
observability, health checks, and correlation IDs.

It is heavily inspired by (and adapted from) [`go-kit/kit`](https://github.com/go-kit/kit),
narrowed down to the pieces likearthian services actually use.

## Features

- **Generic endpoints** — `api.Endpoint[I, O]` and chainable `api.Middleware[I, O]`
  let you compose business logic in a type-safe way (Go 1.18+ generics).
- **HTTP transport** — `NewServer` validates dependencies on construction and
  supports decode/encode, before/after hooks, error encoders, finalizers, and
  Chi-compatible URL routing.
- **Request binding** — `BindURLQuery` / `BindFormData` populate structs (and
  maps) from `url.Values` using struct tags (`query`, `form`). Structured
  `*BindingError` provides field-level failure context.
- **JWT authentication** — centralized `api.TokenVerifier` for endpoint and
  middleware use. HTTP middleware supports JWT-only, API-key-only, or combined
  (JWT-or-API-key) auth, with configurable signing method, audience, and
  custom claims.
- **Structured logging** — a single `logger.Logger` interface with adapters for
  logrus, zerolog, apex/log, and a no-op logger. `MakeEndpointLoggingMiddleware`
  wires request-id, endpoint name and duration into every endpoint automatically.
- **Standard responses & DTOs** — `BaseResponse[T]`, `PaginationDTO`,
  `PagedData[T]`, and `SuccessResponse`/`ErrorResponse` helpers.
- **File uploads/downloads** — `FilePayload` / `FileStreamPayload` DTOs and
  stream-decoding support.
- **Observability** *(v2)* — `MetricsRecorder` interface with a Prometheus
  implementation (histogram, counter, summary) and server finalizer hooks.
- **Health checks** *(v2)* — Liveness/readiness HTTP handlers.
- **Correlation IDs** *(v2)* — Crypto-random ID generation.

## Installation

```bash
go get github.com/likearthian/apikit/v2
```

Requires **Go 1.19+** (generics are used throughout).

## Package layout

```
apikit/v2/
├── api/                          # core, transport-agnostic primitives
│   ├── endpoint.go               # Endpoint[I, O], Middleware[I, O], Chain
│   ├── middleware.go             # Middleware type and composition helpers
│   ├── auth.go                   # JWT claims, CreateToken, endpoint JWT middleware
│   ├── token_verifier.go         # TokenVerifier — centralized JWT parsing
│   ├── context.go                # typed context keys (JWT, claims, API key)
│   ├── errs.go                   # sentinel errors + Err2code mapping
│   ├── result.go                 # Result[T] value/error wrapper
│   └── base.dto.go               # BaseResponse[T], PaginationDTO, PagedData[T], ListItem
├── logger/                       # Logger interface + adapters (logrus, zerolog, apex, noop)
├── transport/
│   ├── error_handler.go          # ErrorHandler interface, LogErrorHandler, NopErrorHandler
│   └── http/
│       ├── server.go             # generic Server[I, O] + options (NewServer returns error)
│       ├── encode_decode.go      # Decode/Encode funcs, default JSON/paged encoders
│       ├── request_response_funcs.go
│       ├── middlewares.go         # JWT, API-key, and combined auth middleware
│       ├── bind.go                # struct/map binding from query/form
│       ├── bind_error.go          # BindingError struct
│       ├── chi.go                 # Chi URL params into context
│       ├── context.go             # request-context populating funcs
│       ├── file.go                # File/FileStream DTOs and uploaders
│       └── const.go              # HTTP header constant names
├── platform/
│   ├── correlation/
│   │   └── id.go                 # NewID — crypto-random correlation ID
│   ├── health/
│   │   └── health.go             # LivenessHandler / ReadinessHandler
│   └── observability/
│       ├── metrics.go            # MetricsRecorder interface, StartTimer, ServerFinalizer
│       └── prometheus/
│           └── prometheus.go     # Prometheus histogram, counter, summary
├── logging.go                    # MakeEndpointLoggingMiddleware
└── example_test.go               # Runnable example
```

## Quick start

```go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/likearthian/apikit/v2"
	"github.com/likearthian/apikit/v2/api"
	httptransport "github.com/likearthian/apikit/v2/transport/http"
	"github.com/likearthian/apikit/v2/logger"
)

type greetRequest struct {
	Name string `query:"name" json:"name"`
}

type greetResponse struct {
	Message string `json:"message"`
}

func greet(_ context.Context, req greetRequest) (greetResponse, error) {
	if req.Name == "" {
		return greetResponse{}, api.ErrBadRequest
	}
	return greetResponse{Message: "hello, " + req.Name}, nil
}

func main() {
	logger := logger.NewNoopLogger() // swap for zerolog/logrus/apex in production

	endpoint := apikit.MakeEndpointLoggingMiddleware[greetRequest, greetResponse](
		logger, "Greet",
	)(greet)

	server, err := httptransport.NewServer(
		endpoint,
		func(_ context.Context, r *http.Request) (greetRequest, error) {
			var req greetRequest
			if err := httptransport.BindURLQuery(&req, r.URL.Query()); err != nil {
				return req, err
			}
			return req, nil
		},
		httptransport.DefaultJSONResponseEncoder[greetResponse],
	)
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/greet", server)
	_ = http.ListenAndServe(":8080", nil)
}
```

Run it:

```bash
curl 'http://localhost:8080/greet?name=world'
# {"request_id":"","message":"success","data":{"message":"hello, world"}}
```

### Auth

```go
keyFn := func(_ *jwt.Token) (any, error) {
	return []byte("replace-with-a-secret"), nil
}

// JWT middleware
auth := httptransport.MakeHTTPJWTMiddleware(keyFn, api.WithAudience("my-service"))

// API-key middleware
apiKeyAuth := httptransport.MakeHTTPAPIKeyMiddleware(func(ctx context.Context, key string) (jwt.Claims, error) {
	if key == "expected-key" {
		return &api.AuthClaims{Username: "service-account"}, nil
	}
	return nil, api.ErrUnauthorized
})

// Combined: JWT or API-key
either := httptransport.MakeHTTPJWTOrAPIKeyMiddleware(keyFn, validateAPIKey, api.WithAudience("my-service"))

r.With(auth).Get("/secret", protectedHandler)
```

Claims are stored at `api.ContextKeyAuthClaims`; the raw JWT at `api.ContextKeyJWTToken`;
the API key at `api.ContextKeyAPIKey`. Use `api.GetAPIKeyFromContext(ctx)` to retrieve it.

### Logging

```go
logger.NewZerolog(zerolog.New(os.Stdout))   // zerolog
logger.NewRusLog(logrus.New())              // logrus
logger.NewApexLogger(...)                   // apex/log
logger.NewNoopLogger()                      // discard everything
```

### Paged responses

```go
endpoint := func(ctx context.Context, req listRequest) (api.PagedData[[]user], error) {
	return api.PagedData[[]user]{
		Data:       users,
		Pagination: api.PaginationDTO{Page: 1, Total: 42},
	}, nil
}

server, err := httptransport.NewServer(
	endpoint,
	decodeListRequest,
	httptransport.DefaultPagedJSONResponseEncoder[[]user],
)
```

### Request binding

```go
var input struct {
	Page   uint     `query:"page"`
	Labels []string `query:"label"`
}
if err := httptransport.BindURLQuery(&input, r.URL.Query()); err != nil {
	var bindingErr *httptransport.BindingError
	if errors.As(err, &bindingErr) {
		log.Printf("invalid %s=%q: %v", bindingErr.Field, bindingErr.Value, bindingErr.Err)
	}
}
```

### Health checks (v2)

```go
health := health.State{}
health.SetReady(true)

mux := chi.NewRouter()
mux.Get("/healthz", health.LivenessHandler)
mux.Get("/readyz", health.ReadinessHandler)
```

### Observability (v2)

```go
metrics := prometheus.NewMetrics("myapp", prometheus.LowCardinalityHistogramBuckets)
mux.Handle("/metrics", prometheus.Handler(registry))

server, err := httptransport.NewServer(
	endpoint, decoder, encoder,
	httptransport.ServerBefore(observability.StartTimer),
	httptransport.ServerFinalizer(observability.ServerFinalizer(metrics)),
)
```

## Error handling

`api.Err2code(err)` maps sentinel errors (defined in `api/errs.go`) to HTTP status
codes. The HTTP `Server` uses `DefaultErrorEncoder` to write those back; supply
your own `ErrorEncoder` via `ServerErrorEncoder` for custom behavior.

```go
server, err := httptransport.NewServer(
	endpoint, decoder, encoder,
	httptransport.ServerErrorEncoder(encodeError),
	httptransport.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
)
```

## Status

This is an internal library used by likearthian services. The API is stable
within the services that depend on it; expect breaking changes between minor
versions rather than respecting strict semver. Pin to a tag for reproducible
builds.

## License

See the project's license file for usage terms. Based on work from
[go-kit/kit](https://github.com/go-kit/kit) by Peter Bourgon and contributors.
