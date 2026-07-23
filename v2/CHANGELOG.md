# Changelog

## Unreleased

### Added

- Go semantic-import-version module path
  `github.com/likearthian/apikit/v2`.
- `api.TokenVerifier` for centralized, reusable JWT verification.
- `transport/http.BindingError` with field, value, and wrapped conversion
  cause.
- `transport.NewNopErrorHandler` for explicit no-op transport diagnostics.
- Constructor errors `ErrNilEndpoint`, `ErrNilRequestDecoder`, and
  `ErrNilResponseEncoder`.
- Contract and regression coverage for responses, binding, authentication,
  HTTP server lifecycle, logging adapters, and transport error handling.

### Changed

- `transport/http.NewServer` now returns `(*Server[I, O], error)` and rejects
  nil endpoint, decoder, and encoder dependencies.
- HTTP auth helpers use initialism-correct names:
  `MakeHTTPJWTMiddleware`, `MakeHTTPAPIKeyMiddleware`,
  `MakeHTTPJWTOrAPIKeyMiddleware`, and `APIKeyFromHeader`.
- API-key context symbols are now `api.ContextKeyAPIKey` and
  `api.GetAPIKeyFromContext`.
- `APIKeyRequestToContext` now reads `X-Api-Key` instead of `api_key` and
  stores it at `api.ContextKeyAPIKey`.
- Endpoint and HTTP JWT middleware share `api.TokenVerifier`.
- Combined JWT-or-API-key authentication gives any `Authorization` header
  precedence and never falls back to an API key after JWT failure.
- Missing or rejected API keys and unusable bearer headers now use the generic
  401 body `not authorized to access this resource`; recognized JWT parse
  failures use normalized sentinel messages. Callers should not depend on the
  legacy literal bodies.
- `api.PagedData[T]` marshals with lowercase `data` and `pagination` keys.
- Binding lookup prefers exact keys, accepts a unique case-insensitive match,
  and rejects ambiguous matches deterministically.
- Query/form binding and URL-query encoding are separated into focused
  implementation files.
- The default HTTP server error handler is an explicit no-op; applications opt
  into logging or other diagnostics.
- The root endpoint logging middleware preserves the endpoint's typed result
  when an error is returned.

### Fixed

- Root-package build failures caused by stale response references.
- Nil-map initialization, empty map values, nil pointer map values, interface
  values, and string-key validation in reflection-based binding.
- API-key middleware request-context propagation and typed-nil validator
  results.
- JWT signing-key selection to use `crypto/rand`, including safe single-key
  operation.
- JWT key lookup for missing, malformed, negative, and out-of-range `kid`
  headers.
- JWT algorithm validation, signature verification, nil callbacks, typed-nil
  claims, and panicking key/claims callbacks.
- JWT credential and token sentinel errors now map consistently to HTTP 401
  through `api.Err2code`; the `ErrJWTKeyFuncMissing` verifier-configuration
  failure remains HTTP 500. HTTP JWT middleware preserves that server-error
  status while redacting the response body to `Internal Server Error`.

### Removed

- The duplicate root response API: `apikit.BaseResponse`,
  `apikit.PaginationDTO`, `apikit.PagedResponse`, `apikit.SuccessResponse`,
  `apikit.ErrorResponse`, and `apikit.ResponseType`.
- Legacy auth spellings `MakeHttpJwtMiddleware`,
  `MakeHttpApikeyMiddleware`, `MakeHttpJwtAndApikeyMiddleware`,
  `ApikeyFromHeader`, `api.ContextKeyApikey`, and
  `api.GetApikeyFromContext`.
- `api.DefaultKeys` and `api.DefaultJwtKeyGetterFunc`. The exported,
  package-owned HS256 secret was unsafe because it allowed forged tokens;
  applications must supply their own keys through
  `api.CreateJwtKeyGetterFunc`. The public v1 key must be rotated out of every
  accepted key set, and tokens signed with it must be invalidated or revoked.
