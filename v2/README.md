# apikit v2

`apikit` is a small Go library for building typed HTTP services with go-kit-style
endpoints. It provides generic endpoint and middleware types, HTTP transports,
request binding, canonical JSON response envelopes, JWT and API-key
authentication, and structured-logging adapters.

This is an internal likearthian library. Version 2 deliberately removes legacy
APIs; pin a release and read [MIGRATION.md](MIGRATION.md) before upgrading.

## Requirements and installation

apikit v2 requires Go 1.19 or newer.

```bash
go get github.com/likearthian/apikit/v2
```

All imports must include the `/v2` suffix.

## Packages

| Package | Purpose |
| --- | --- |
| `github.com/likearthian/apikit/v2` | Endpoint logging middleware |
| `github.com/likearthian/apikit/v2/api` | Generic endpoints, middleware, response DTOs, errors, and JWT verification |
| `github.com/likearthian/apikit/v2/transport` | Transport error handlers |
| `github.com/likearthian/apikit/v2/transport/http` | HTTP server, codecs, binding, auth middleware, context hooks, and file helpers |
| `github.com/likearthian/apikit/v2/logger` | Logger interface and logrus, zerolog, apex, and no-op adapters |

## Minimal HTTP endpoint

```go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/likearthian/apikit/v2/api"
	httptransport "github.com/likearthian/apikit/v2/transport/http"
)

type greetRequest struct {
	Name string `query:"name"`
}

type greetResponse struct {
	Message string `json:"message"`
}

func main() {
	endpoint := api.Endpoint[greetRequest, greetResponse](
		func(_ context.Context, request greetRequest) (greetResponse, error) {
			return greetResponse{Message: "hello, " + request.Name}, nil
		},
	)

	decode := func(_ context.Context, request *http.Request) (greetRequest, error) {
		var input greetRequest
		err := httptransport.BindURLQuery(&input, request.URL.Query())
		return input, err
	}

	server, err := httptransport.NewServer(
		endpoint,
		decode,
		httptransport.DefaultJSONResponseEncoder[greetResponse],
	)
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/greet", server)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

`NewServer` validates its endpoint, decoder, and encoder and returns
`(*Server[I, O], error)`. Missing dependencies return `ErrNilEndpoint`,
`ErrNilRequestDecoder`, or `ErrNilResponseEncoder`.

## Canonical responses

Use the generic response types in `api`:

```go
response := api.SuccessResponse("request-42", greetResponse{Message: "hello"})

paged := api.PagedData[[]greetResponse]{
	Data: []greetResponse{{Message: "hello"}},
	Pagination: api.PaginationDTO{
		Page:  1,
		Total: 1,
	},
}
```

`api.BaseResponse[T]` is the canonical JSON envelope. Its fields are
`request_id`, `message`, `error`, `data`, and optional `pagination`.
`DefaultJSONResponseEncoder` wraps an endpoint result with
`api.SuccessResponse`; `DefaultPagedJSONResponseEncoder` accepts
`api.PagedData[T]` and places its pagination in the envelope.
`api.ErrorResponse(requestID, err)` constructs an error envelope but does not
choose an HTTP status.

## Request binding

`BindURLQuery` and `BindFormData` populate a pointer to a struct or
`map[string]string`. Struct fields use `query` or `form` tags, support scalar
and comma-separated slice conversion, and can implement
`encoding.TextUnmarshaler`.

```go
var input struct {
	Page   uint     `query:"page"`
	Labels []string `query:"label"`
}
if err := httptransport.BindURLQuery(&input, request.URL.Query()); err != nil {
	var bindingErr *httptransport.BindingError
	if errors.As(err, &bindingErr) {
		// bindingErr.Field, bindingErr.Value, and bindingErr.Err identify the failure.
	}
}
```

Conversion failures are reported as `*httptransport.BindingError`, which wraps
the underlying conversion cause. Callers can use `errors.As` for field context
and `errors.Is` for the underlying cause. Exact input keys win over
case-insensitive matches; ambiguous case-insensitive keys are rejected
deterministically.

## Authentication

`api.TokenVerifier` centralizes JWT parsing and verification for endpoint and
HTTP middleware. It requires a key function, creates fresh claims for each
verification, enforces the configured signing method, and returns errors that
can be inspected with `errors.Is`.

```go
keyFn := func(*jwt.Token) (any, error) {
	return []byte("replace-with-a-secret"), nil
}

verifier := api.NewTokenVerifier(keyFn, api.WithAudience("accounts"))
token, err := verifier.Verify(rawToken)
if errors.Is(err, api.ErrTokenExpired) {
	// Refresh or reject the credential.
}

jwtOnly := httptransport.MakeHTTPJWTMiddleware(
	keyFn,
	api.WithAudience("accounts"),
)
apiKeyOnly := httptransport.MakeHTTPAPIKeyMiddleware(validateAPIKey)
either := httptransport.MakeHTTPJWTOrAPIKeyMiddleware(
	keyFn,
	validateAPIKey,
	api.WithAudience("accounts"),
)
```

JWT middleware stores the raw token at `api.ContextKeyJWTToken` and verified
claims at `api.ContextKeyAuthClaims`. API-key middleware reads `X-Api-Key`,
stores claims at `api.ContextKeyAuthClaims`, and stores the key at
`api.ContextKeyAPIKey`; retrieve it with `api.GetAPIKeyFromContext`.
The `APIKeyRequestToContext` request hook also reads `X-Api-Key` and stores it
at `api.ContextKeyAPIKey`.

For combined authentication, the presence of an `Authorization` header selects
JWT authentication. An invalid JWT is rejected and never falls back to an API
key.

Authentication middleware writes status 401 with `http.Error`. Missing or
rejected API keys and missing or unusable bearer headers currently use the
generic `not authorized to access this resource` body; a nonempty but
syntactically malformed JWT uses `JWT Token is malformed`. `http.Error` appends
a newline. Treat these strings as diagnostic text rather than a stable client
contract.

## HTTP errors and diagnostics

`DefaultErrorEncoder` writes status 500 and plain text by default. An error that
implements `httptransport.StatusCoder` controls the status; an error that
implements `httptransport.Headerer` supplies response headers; an error that
implements `json.Marshaler` supplies a JSON body. Use
`httptransport.ServerErrorEncoder` when the application needs a custom error
envelope or domain-to-status mapping.

Transport diagnostics are independent of the client response.
`transport.NewNopErrorHandler` is the server default.
Use `transport.NewLogErrorHandler` or
`httptransport.ServerErrorHandler` for explicit reporting, and
`httptransport.ServerFinalizer` for end-of-request telemetry.

## Logging and verification

`logger.Logger` has adapters for logrus, zerolog, apex, and a no-op
implementation. `apikit.MakeEndpointLoggingMiddleware` records endpoint
duration and errors without replacing the endpoint's typed result.

Before contributing, run:

```bash
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
```

See [CHANGELOG.md](CHANGELOG.md) for release-level changes.
