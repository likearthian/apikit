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

`apikit` is a small, opinionated Go library for building HTTP services on top of
[go-kit](https://github.com/go-kit/kit) concepts. It provides generic, type-safe
endpoint and transport primitives, pluggable logging adapters, JWT auth
middleware, request/response binding, and helpers for files and paging — all
with minimal dependencies.

It is heavily inspired by (and adapted from) [`go-kit/kit`](https://github.com/go-kit/kit),
narrowed down to the pieces the likearthian services actually use.

## Features

- **Generic endpoints** — `api.Endpoint[I, O]` and chainable `api.Middleware[I, O]`
  let you compose business logic in a type-safe way (Go 1.18+ generics).
- **HTTP transport** — server-side decode/encode, request/response function
  hooks, error encoders, finalizers, and an `ErrorEncoder` to translate domain
  errors into HTTP status codes. Chi-compatible URL params and routing.
- **Request binding** — `BindURLQuery` / `BindFormData` populate structs (and
  maps) from `url.Values` using struct tags (`query`, `form`), without pulling
  in a full web framework.
- **JWT auth** — HTTP middleware (`MakeHttpJwtMiddleware`) and helpers
  (`TokenFromHeader`, `TokenFromContext`) built on `dgrijalva/jwt-go/v4`, with
  configurable signing method, audience and custom claims.
- **Structured logging** — a single `logger.Logger` interface with adapters for
  `logrus`, `zerolog`, `apex/log`, and a no-op logger. `MakeEndpointLoggingMiddleware`
  wires request-id, endpoint name and duration into every endpoint automatically.
- **Standard responses & DTOs** — `BaseResponse[T]`, `PaginationDTO`,
  `PagedData[T]`, `ListItem`, and `SuccessResponse`/`PagedResponse` helpers.
- **File uploads/downloads** — `FilePayload` / `FileStreamPayload` DTOs and
  stream-decoding support.

## Installation

```bash
go get github.com/likearthian/apikit
```

Requires **Go 1.19+** (generics are used throughout).

## Package layout

```
.
├── api/                  # core, transport-agnostic primitives
│   ├── endpoint.go       # Endpoint[I, O], Middleware[I, O], Chain
│   ├── middleware.go     # Middleware type and composition helpers
│   ├── auth.go           # JWT claims, options, signing
│   ├── context.go        # typed context keys (JWT token, claims, apikey)
│   ├── errs.go           # sentinel errors + Err2code mapping
│   ├── result.go        #  Result[T] value/error wrapper
│   └── base.dto.go       # BaseResponse[T], Paging, ListItem DTOs
├── logger/               # Logger interface + adapters (logrus, zerolog, apex, noop)
├── transport/
│   ├── error_handler.go  # LogErrorHandler for transport errors
│   └── http/             # HTTP transport implementation
│       ├── server.go             # generic Server[I, O] + options
│       ├── encode_decode.go       # Decode/Encode/Create request funcs
│       ├── request_response_funcs.go
│       ├── middlewares.go         # JWT auth + CORS-style helpers
│       ├── bind.go                # struct/map binding from query/form
│       ├── file.go                # file/FileStream DTOs
│       ├── context.go             # request-context populating funcs
│       ├── chi.go                # Chi URL params into context
│       └── const.go              # HTTP header constant names
├── logging.go             # MakeEndpointLoggingMiddleware
└── response.go            # legacy BaseResponse / ResponseType map
```

## Quick start

A minimal service exposes an `api.Endpoint[I, O]` and wires it to an HTTP
transport server with a decoder, encoder, and logging middleware:

```go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/likearthian/apikit"
	"github.com/likearthian/apikit/api"
	apitransport "github.com/likearthian/apikit/transport/http"
	log "github.com/likearthian/apikit/logger"
)

// Request / response DTOs.
type greetRequest struct {
	Name string `query:"name" json:"name"`
}

type greetResponse struct {
	Message string `json:"message"`
}

// Business logic as an Endpoint.
func greet(_ context.Context, req greetRequest) (greetResponse, error) {
	if req.Name == "" {
		return greetResponse{}, errors.New("name is required")
	}
	return greetResponse{Message: "hello, " + req.Name}, nil
}

func main() {
	logger := log.NewNoopLogger() // swap for zerolog/logrus/apex in production

	// Compose middleware: logging wraps the business endpoint.
	endpoint := apikit.MakeEndpointLoggingMiddleware[greetRequest, greetResponse](
		logger, "Greet",
	)(greet)

	server := apitransport.NewServer(
		endpoint,
		// decode incoming request (query + JSON body supported via BindURLQuery)
		func(_ context.Context, r *http.Request) (greetRequest, error) {
			var req greetRequest
			if err := apitransport.BindURLQuery(&req, r.URL.Query()); err != nil {
				return req, err
			}
			return req, nil
		},
		// encode the response as JSON
		func(_ context.Context, w http.ResponseWriter, resp greetResponse) error {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			return json.NewEncoder(w).Encode(api.SuccessResponse("", resp))
		},
	)

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

Protect a handler with JWT using the HTTP JWT middleware:

```go
auth := apitransport.MakeHttpJwtMiddleware(func(t *jwt.Token) (interface{}, error) {
	return []byte("my-secret"), nil
}, api.WithAudience("my-service"))

r.With(auth).Get("/secret", protectedHandler)
```

Claims (`api.AuthClaims`) are placed in the request context under
`api.ContextKeyAuthClaims`; the raw token under `api.ContextKeyJWTToken`. Use
the typed helpers in `api/context.go` to retrieve them.

### Logging

Pick any adapter — they all implement `logger.Logger`:

```go
log.NewZerolog(zerolog.New(os.Stdout))   // zerolog
log.NewRusLog(logrus.New())              // logrus
log.NewApexLogger(...)                    // apex/log
log.NewNoopLogger()                       // discard everything
```

`MakeEndpointLoggingMiddleware` logs request id, endpoint name, duration, and
error code on every call.

## Error handling

`api.Err2code(err)` maps common sentinel errors (defined in `api/errs.go`) to
HTTP status codes. The HTTP `Server` uses `DefaultErrorEncoder` to write those
back; supply your own `ErrorEncoder` via `ServerWithErrorEncoder` for custom
behavior. Transport errors may also be routed to a `LogErrorHandler`.

## Status

This is an internal library used by likearthian services. The API is stable
within the services that depend on it; expect breaking changes between minor
versions rather than respecting strict semver. Pin to a tag for reproducible
builds.

## License

See the project's license file for usage terms. Based on work from
[go-kit/kit](https://github.com/go-kit/kit) by Peter Bourgon and contributors.