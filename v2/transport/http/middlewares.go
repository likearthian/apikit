package http

import (
	"context"
	"net/http"
	"reflect"
	"strings"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/likearthian/apikit/v2/api"
)

func MakeHTTPJWTMiddleware(keyFn jwt.Keyfunc, options ...api.JwtOption) func(http.Handler) http.Handler {
	verifier := api.NewTokenVerifier(keyFn, options...)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := TokenFromHeader(r)
			token, err := verifier.Verify(tokenString)
			if err != nil {
				writeAuthError(w, err)
				return
			}

			ctx := context.WithValue(r.Context(), api.ContextKeyJWTToken, tokenString)
			ctx = context.WithValue(ctx, api.ContextKeyAuthClaims, token.Claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func MakeHTTPAPIKeyMiddleware(validateFn func(apiKey string) any) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := APIKeyFromHeader(r)
			if apiKey == "" {
				writeAuthError(w, api.ErrUnauthorized)
				return
			}

			claims, ok := validateAPIKey(validateFn, apiKey)
			if !ok {
				writeAuthError(w, api.ErrUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), api.ContextKeyAuthClaims, claims)
			ctx = context.WithValue(ctx, api.ContextKeyAPIKey, apiKey)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func MakeHTTPJWTOrAPIKeyMiddleware(
	jwtKeyFn jwt.Keyfunc,
	apiKeyValidateFn func(apiKey string) any,
	options ...api.JwtOption,
) func(http.Handler) http.Handler {
	jwtMiddleware := MakeHTTPJWTMiddleware(jwtKeyFn, options...)
	apiKeyMiddleware := MakeHTTPAPIKeyMiddleware(apiKeyValidateFn)

	return func(next http.Handler) http.Handler {
		jwtHandler := jwtMiddleware(next)
		apiKeyHandler := apiKeyMiddleware(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(r.Header.Values("Authorization")) != 0 {
				jwtHandler.ServeHTTP(w, r)
				return
			}

			apiKeyHandler.ServeHTTP(w, r)
		})
	}
}

func writeAuthError(w http.ResponseWriter, err error) {
	status := api.Err2code(err)
	message := api.ParseJwtError(err)
	if status >= http.StatusInternalServerError {
		message = http.StatusText(status)
	}
	http.Error(w, message, status)
}

func validateAPIKey(validateFn func(string) any, apiKey string) (claims any, ok bool) {
	if validateFn == nil {
		return nil, false
	}

	completed := false
	defer func() {
		if !completed {
			recover()
			claims = nil
			ok = false
		}
	}()

	claims = validateFn(apiKey)
	completed = true
	return claims, !isNilAPIKeyClaims(claims)
}

func isNilAPIKeyClaims(claims any) bool {
	if claims == nil {
		return true
	}

	value := reflect.ValueOf(claims)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return value.IsNil()
	default:
		return false
	}
}

func TokenFromHeader(r *http.Request) string {
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 7 && strings.ToUpper(bearer[0:7]) == "BEARER " {
		return bearer[7:]
	}
	return ""
}

func APIKeyFromHeader(r *http.Request) string {
	return r.Header.Get("X-Api-Key")
}
