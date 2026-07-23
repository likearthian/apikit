# Migrating from apikit v1 to v2

Version 2 is a breaking cleanup. It adopts Go semantic import versioning,
removes the duplicate root response model, makes HTTP server construction
fallible, and consolidates authentication and binding behavior.

## 1. Update the module path

```bash
go get github.com/likearthian/apikit/v2
```

Before:

```go
import (
	"github.com/likearthian/apikit"
	"github.com/likearthian/apikit/api"
	httptransport "github.com/likearthian/apikit/transport/http"
)
```

After:

```go
import (
	"github.com/likearthian/apikit/v2"
	"github.com/likearthian/apikit/v2/api"
	httptransport "github.com/likearthian/apikit/v2/transport/http"
)
```

## API mapping

| v1 API | v2 replacement | Migration note |
| --- | --- | --- |
| Module/import path `github.com/likearthian/apikit` | `github.com/likearthian/apikit/v2` | Add `/v2` to every apikit import. |
| Root `apikit.BaseResponse` | `api.BaseResponse[T]` | The canonical envelope is generic and no longer contains `status_code` or `status_text`. |
| Root `apikit.PaginationDTO` | `api.PaginationDTO` | `Page` and `Total` change from `int` to `uint`. Validate before converting negative values. |
| Root `apikit.PagedResponse` | `api.PagedData[T]` plus `httptransport.DefaultPagedJSONResponseEncoder` | Endpoint data and pagination are typed; the encoder creates the canonical envelope. |
| Root `apikit.SuccessResponse` | `api.SuccessResponse` | Type inference normally supplies `T`. |
| Root `apikit.ErrorResponse(requestID, code, err)` | `api.ErrorResponse(requestID, err)` | Select HTTP status through an `httptransport.ErrorEncoder` or an error implementing `httptransport.StatusCoder`. |
| Root `apikit.ResponseType` | `net/http.StatusText` or application-owned messages | The mutable package-level response text map was removed. |
| `MakeHttpJwtMiddleware` | `MakeHTTPJWTMiddleware` | JWT verification now uses `api.TokenVerifier`. |
| `MakeHttpApikeyMiddleware` | `MakeHTTPAPIKeyMiddleware` | API-key context propagation is fixed and validators fail closed. |
| `MakeHttpJwtAndApikeyMiddleware` | `MakeHTTPJWTOrAPIKeyMiddleware` | The name reflects either credential being accepted. Authorization-header precedence is described below. |
| `ApikeyFromHeader` | `APIKeyFromHeader` | Reads `X-Api-Key`. |
| `api.ContextKeyApikey` | `api.ContextKeyAPIKey` | Update direct context access to the renamed key. |
| `api.GetApikeyFromContext` | `api.GetAPIKeyFromContext` | Returns the API key string or `""`. |
| `APIKeyRequestToContext` reading `api_key` | `APIKeyRequestToContext` reading `X-Api-Key` | The function name is unchanged, but the stored key changes to `api.ContextKeyAPIKey`. |
| `api.DefaultKeys` and `api.DefaultJwtKeyGetterFunc` | `api.CreateJwtKeyGetterFunc(applicationKeys)` | The package-owned HS256 secret was removed. Supply keys from application configuration or a secret store. |
| `httptransport.NewServer(...) *Server[I, O]` | `httptransport.NewServer(...) (*Server[I, O], error)` | Handle constructor validation errors. |

The old symbols are removed rather than retained as deprecated aliases.

## 2. Migrate response construction

Before:

```go
payload := apikit.SuccessResponse(
	requestID,
	users,
	apikit.PaginationDTO{Page: page, Total: total},
)

failure := apikit.ErrorResponse(requestID, http.StatusBadRequest, err)
```

After:

```go
payload := api.SuccessResponse(
	requestID,
	users,
	api.PaginationDTO{Page: uint(page), Total: uint(total)},
)

failure := api.ErrorResponse(requestID, err)
```

Validate signed pagination values before converting them to `uint`.

The v2 JSON envelope is intentionally smaller:

```json
{
  "request_id": "request-42",
  "message": "success",
  "data": []
}
```

`status_code` and `status_text` were removed from the body. HTTP status belongs
to the HTTP response, while `message` is an application-level value.

For paged endpoints, return `api.PagedData[T]`:

```go
endpoint := func(ctx context.Context, request listRequest) (api.PagedData[[]user], error) {
	return api.PagedData[[]user]{
		Data: users,
		Pagination: api.PaginationDTO{
			Page:  1,
			Total: 42,
		},
	}, nil
}

server, err := httptransport.NewServer(
	endpoint,
	decodeListRequest,
	httptransport.DefaultPagedJSONResponseEncoder[[]user],
)
```

`api.PagedData[T]` now has explicit JSON tags. When marshaled directly, its keys
are lowercase `data` and `pagination`, replacing the v1-style `Data` and
`Pagination` keys.

## 3. Handle server construction errors

Before:

```go
server := httptransport.NewServer(endpoint, decoder, encoder)
```

After:

```go
server, err := httptransport.NewServer(endpoint, decoder, encoder)
if err != nil {
	return fmt.Errorf("construct users HTTP server: %w", err)
}
```

`NewServer` rejects nil required dependencies with `ErrNilEndpoint`,
`ErrNilRequestDecoder`, or `ErrNilResponseEncoder`. Update constructor helpers
and tests to return or assert the new error.

## 4. Move status selection out of response DTOs

`api.ErrorResponse` only constructs the canonical body. Configure an HTTP error
encoder when the body and status must be written together:

```go
func encodeError(ctx context.Context, err error, writer http.ResponseWriter) {
	status := api.Err2code(err)
	requestID, _ := httptransport.ReqIDFromContext(ctx)
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(
		api.ErrorResponse(requestID, err),
	)
}

server, err := httptransport.NewServer(
	endpoint,
	decoder,
	encoder,
	httptransport.ServerErrorEncoder(encodeError),
)
```

When the default plain-text body is sufficient, an error value can implement:

```go
type statusError struct {
	error
	code int
}

func (err statusError) StatusCode() int { return err.code }
```

`httptransport.DefaultErrorEncoder` recognizes
`httptransport.StatusCoder`, uses status 500 otherwise, and does not use
`api.ErrorResponse` automatically.

## 5. Update binding error handling

Existing calls to `BindURLQuery` and `BindFormData` keep their signatures.
Conversion failures now contain structured field context:

```go
err := httptransport.BindURLQuery(&request, values)
if err != nil {
	var bindingErr *httptransport.BindingError
	if errors.As(err, &bindingErr) {
		log.Printf("invalid %s=%q: %v", bindingErr.Field, bindingErr.Value, bindingErr.Err)
	}
}
```

`BindingError.Unwrap` preserves `errors.Is` and `errors.As` checks for the
original conversion error. Exact input keys take precedence. If multiple
case-insensitive keys match a field and no exact key exists, binding now returns
a deterministic ambiguity error. Map destinations are limited to
`*map[string]string`; nil maps are initialized, and empty value slices are
ignored.

## 6. Update authentication

Rename HTTP helpers according to the mapping table and use `api.TokenVerifier`
when application code verifies raw JWTs:

```go
verifier := api.NewTokenVerifier(
	keyFn,
	api.WithAudience("accounts"),
	api.WithClaimsFactory(api.StandardClaimsFactory),
)

token, err := verifier.Verify(rawToken)
if err != nil {
	return fmt.Errorf("verify access token: %w", err)
}
```

`TokenVerifier` resolves options once, creates fresh claims for each call,
requires the configured signing method, verifies the signature, and normalizes
JWT errors for `errors.Is`. Missing or panicking key/claims callbacks fail
closed.

The exported `api.DefaultKeys` and `api.DefaultJwtKeyGetterFunc` were removed
because a package-owned public HS256 secret lets anyone forge tokens accepted
by consumers of that getter. Load keys from application configuration or a
secret store and construct the key function explicitly. Generate at least 32
random bytes with a cryptographically secure generator; do not use a
human-chosen password. This example expects that key material to be base64
encoded in configuration and fails closed unless it decodes to at least 256
bits:

```go
import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/likearthian/apikit/v2/api"
)

func loadJWTKeyFunc() (jwt.Keyfunc, error) {
	encodedKey, ok := os.LookupEnv("APIKIT_JWT_KEY_BASE64")
	if !ok || encodedKey == "" {
		return nil, fmt.Errorf("APIKIT_JWT_KEY_BASE64 is required")
	}

	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, fmt.Errorf("decode APIKIT_JWT_KEY_BASE64: %w", err)
	}
	if len(key) < 32 {
		return nil, fmt.Errorf("APIKIT_JWT_KEY_BASE64 must contain at least 32 random bytes")
	}

	return api.CreateJwtKeyGetterFunc([]string{string(key)}), nil
}
```

The v1 default key is public and must be treated as compromised. Rotate it
immediately, remove it from every accepted key set, and invalidate or revoke
all tokens signed with it. Do not retain the old key for a migration grace
period.

Verification errors may return project JWT sentinels directly or wrap them.
In either case, replace direct equality checks with `errors.Is`. Credential and
token failures map to HTTP 401 through `api.Err2code`;
`api.ErrJWTKeyFuncMissing` instead identifies a server configuration failure
and maps to HTTP 500. The HTTP JWT middleware writes the generic
`Internal Server Error` body for server failures and does not expose verifier
configuration details:

```go
// v1 pattern:
if err == api.ErrTokenExpired {
	// ...
}

// v2: inspect the error chain.
if errors.Is(err, api.ErrTokenExpired) {
	// ...
}
```

Combined middleware has an intentional precedence rule:

```go
auth := httptransport.MakeHTTPJWTOrAPIKeyMiddleware(keyFn, validateAPIKey)
```

If any `Authorization` header is present, the request takes the JWT path. A
malformed, expired, invalid, or otherwise unverifiable JWT returns 401 and
never falls back to `X-Api-Key`. The API-key path is used only when the
`Authorization` header is absent.

API-key middleware now propagates values through the request context correctly:

```go
apiKey := api.GetAPIKeyFromContext(request.Context())
claims := request.Context().Value(api.ContextKeyAuthClaims)
```

If a route uses `APIKeyRequestToContext` as a `ServerBefore` hook, update the
request header as well as the context key:

```go
server, err := httptransport.NewServer(
	endpoint,
	decoder,
	encoder,
	httptransport.ServerBefore(httptransport.APIKeyRequestToContext),
)

// v1 clients sent: api_key: secret
// v2 clients send: X-Api-Key: secret
```

The v2 auth middleware does not preserve v1's distinct literal error bodies
such as `Not Authorized`, `Apikey required. Authorized`, and
`Apikey Validation failed. Not Authorized`. Its current 401 responses are:

- Missing or rejected API key: `not authorized to access this resource`.
- Missing bearer token or an `Authorization` value that cannot yield a bearer
  token: `not authorized to access this resource`.
- A nonempty bearer token that is syntactically malformed:
  `JWT Token is malformed`.
- Other recognized JWT failures use their normalized sentinel message.

These bodies are written with `http.Error`, which appends a newline. Treat them
as diagnostic text, not a stable client contract; use the 401 status for client
control flow. Server-side JWT configuration failures use status 500 and the
generic `Internal Server Error` body. In combined middleware, an
`Authorization` header still prevents API-key fallback when that server error
occurs.

## 7. Choose transport diagnostics explicitly

Servers use `transport.NewNopErrorHandler()` by default, so transport errors are
not logged implicitly. To log them:

```go
server, err := httptransport.NewServer(
	endpoint,
	decoder,
	encoder,
	httptransport.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
)
```

Use `transport.NewNopErrorHandler` explicitly in tests or composition code that
needs a concrete no-op handler.

## Upgrade checklist

1. Add `/v2` to imports and module requirements.
2. Replace root response types and helpers with the `api` package.
3. Validate signed pagination values before converting to `uint`.
4. Handle the error returned by every `NewServer` call.
5. Rename HTTP and API-key authentication symbols.
6. Update `APIKeyRequestToContext` clients from `api_key` to `X-Api-Key` and
   use `api.ContextKeyAPIKey`.
7. Replace direct JWT sentinel equality checks with `errors.Is`.
8. Replace `DefaultJwtKeyGetterFunc` and `DefaultKeys` with
   `CreateJwtKeyGetterFunc` using validated, high-entropy,
   application-managed secrets. Remove the public v1 key and invalidate or
   revoke every token signed with it.
9. Stop depending on legacy authentication error-body strings.
10. Decide whether combined auth's Authorization-header precedence matches each
   route.
11. Update binding-error handling to use `*BindingError` where field context is
   useful.
12. Configure an error encoder, status-carrying errors, and diagnostic error
   handler intentionally.
13. Run `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...`.
