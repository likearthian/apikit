package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"unsafe"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/likearthian/apikit/v2/api"
)

const middlewareTestJWTKey = "01234567890123456789012345678901"

type middlewareAPIKeyClaims struct {
	Username string
}

func signMiddlewareToken(t *testing.T, username string) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &api.AuthClaims{
		Username: username,
	})
	token.Header["kid"] = "0"

	raw, err := token.SignedString([]byte(middlewareTestJWTKey))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	return raw
}

func middlewareTestKeyFunc(*jwt.Token) (any, error) {
	return []byte(middlewareTestJWTKey), nil
}

func TestMakeHTTPAPIKeyMiddlewarePropagatesAPIKeyAndClaims(t *testing.T) {
	wantClaims := &middlewareAPIKeyClaims{Username: "alice"}
	called := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		called = true
		if got := api.GetAPIKeyFromContext(r.Context()); got != "secret-key" {
			t.Errorf("API key = %q, want %q", got, "secret-key")
		}
		if got := r.Context().Value(api.ContextKeyAuthClaims); got != wantClaims {
			t.Errorf("auth claims = %#v, want %#v", got, wantClaims)
		}
	})
	handler := MakeHTTPAPIKeyMiddleware(func(apiKey string) any {
		if apiKey != "secret-key" {
			return nil
		}
		return wantClaims
	})(next)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Api-Key", "secret-key")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if !called {
		t.Fatal("next handler was not called")
	}
	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestMakeHTTPAPIKeyMiddlewareRejectsInvalidMissingAndUnsafeValidators(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		validator func(string) any
	}{
		{
			name:   "missing API key",
			apiKey: "",
			validator: func(string) any {
				return struct{}{}
			},
		},
		{
			name:   "invalid API key",
			apiKey: "invalid",
			validator: func(string) any {
				return nil
			},
		},
		{
			name:   "typed nil claims",
			apiKey: "secret-key",
			validator: func(string) any {
				var claims *middlewareAPIKeyClaims
				return claims
			},
		},
		{
			name:   "nil unsafe pointer claims",
			apiKey: "secret-key",
			validator: func(string) any {
				return unsafe.Pointer(nil)
			},
		},
		{
			name:      "nil validator",
			apiKey:    "secret-key",
			validator: nil,
		},
		{
			name:   "panicking validator",
			apiKey: "secret-key",
			validator: func(string) any {
				panic("sensitive validator failure")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			handler := MakeHTTPAPIKeyMiddleware(tt.validator)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				called = true
			}))
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.apiKey != "" {
				request.Header.Set("X-Api-Key", tt.apiKey)
			}
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if called {
				t.Error("next handler was called")
			}
			if recorder.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestMakeHTTPJWTMiddlewarePropagatesRawTokenAndClaims(t *testing.T) {
	raw := signMiddlewareToken(t, "alice")
	called := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		called = true
		if got := r.Context().Value(api.ContextKeyJWTToken); got != raw {
			t.Errorf("raw token = %#v, want %q", got, raw)
		}
		claims, ok := r.Context().Value(api.ContextKeyAuthClaims).(*api.AuthClaims)
		if !ok {
			t.Fatalf("auth claims = %T, want *api.AuthClaims", r.Context().Value(api.ContextKeyAuthClaims))
		}
		if got, want := claims.Username, "alice"; got != want {
			t.Errorf("username = %q, want %q", got, want)
		}
	})
	handler := MakeHTTPJWTMiddleware(middlewareTestKeyFunc)(next)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+raw)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if !called {
		t.Fatal("next handler was not called")
	}
	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestMakeHTTPJWTMiddlewareRejectsMalformedAndMissingTokens(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
		wantBody      string
	}{
		{
			name:     "missing",
			wantBody: "not authorized to access this resource\n",
		},
		{
			name:          "malformed",
			authorization: "Bearer not-a-jwt",
			wantBody:      api.ErrTokenMalformed.Error() + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			handler := MakeHTTPJWTMiddleware(middlewareTestKeyFunc)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				called = true
			}))
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set("Authorization", tt.authorization)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if called {
				t.Error("next handler was called")
			}
			if recorder.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
			if got := recorder.Body.String(); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
		})
	}
}

func TestMakeHTTPJWTMiddlewareReportsMissingKeyFunctionAsInternalServerError(t *testing.T) {
	called := false
	handler := MakeHTTPJWTMiddleware(nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+signMiddlewareToken(t, "alice"))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if called {
		t.Error("next handler was called")
	}
	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if got, want := recorder.Body.String(), http.StatusText(http.StatusInternalServerError)+"\n"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
	if got := recorder.Body.String(); got == api.ErrJWTKeyFuncMissing.Error()+"\n" {
		t.Error("response exposed the missing key-function error")
	}
}

func TestMakeHTTPJWTOrAPIKeyMiddlewareAcceptsJWTBranch(t *testing.T) {
	raw := signMiddlewareToken(t, "jwt-user")
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if got := r.Context().Value(api.ContextKeyJWTToken); got != raw {
			t.Errorf("raw token = %#v, want %q", got, raw)
		}
		claims, ok := r.Context().Value(api.ContextKeyAuthClaims).(*api.AuthClaims)
		if !ok || claims.Username != "jwt-user" {
			t.Errorf("auth claims = %#v, want JWT claims for jwt-user", r.Context().Value(api.ContextKeyAuthClaims))
		}
		if got := api.GetAPIKeyFromContext(r.Context()); got != "" {
			t.Errorf("API key = %q, want empty", got)
		}
	})
	handler := MakeHTTPJWTOrAPIKeyMiddleware(middlewareTestKeyFunc, func(string) any {
		t.Fatal("API-key validator called for JWT request")
		return nil
	})(next)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+raw)
	request.Header.Set("X-Api-Key", "also-present")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestMakeHTTPJWTOrAPIKeyMiddlewareAcceptsAPIKeyBranch(t *testing.T) {
	wantClaims := &middlewareAPIKeyClaims{Username: "api-key-user"}
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if got := api.GetAPIKeyFromContext(r.Context()); got != "secret-key" {
			t.Errorf("API key = %q, want %q", got, "secret-key")
		}
		if got := r.Context().Value(api.ContextKeyAuthClaims); got != wantClaims {
			t.Errorf("auth claims = %#v, want %#v", got, wantClaims)
		}
	})
	handler := MakeHTTPJWTOrAPIKeyMiddleware(middlewareTestKeyFunc, func(apiKey string) any {
		if apiKey != "secret-key" {
			return nil
		}
		return wantClaims
	})(next)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Api-Key", "secret-key")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestMakeHTTPJWTOrAPIKeyMiddlewareRejectsAbsentAndInvalidJWT(t *testing.T) {
	tests := []struct {
		name             string
		setAuthorization bool
		authorization    string
		apiKey           string
	}{
		{name: "credentials absent"},
		{
			name:             "invalid JWT does not fall back to API key",
			setAuthorization: true,
			authorization:    "Bearer not-a-jwt",
			apiKey:           "valid-api-key",
		},
		{
			name:             "malformed authorization does not fall back to API key",
			setAuthorization: true,
			authorization:    "Bearer",
			apiKey:           "valid-api-key",
		},
		{
			name:             "empty authorization header does not fall back to API key",
			setAuthorization: true,
			apiKey:           "valid-api-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			validatorCalled := false
			handler := MakeHTTPJWTOrAPIKeyMiddleware(middlewareTestKeyFunc, func(string) any {
				validatorCalled = true
				return struct{}{}
			})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				called = true
			}))
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.setAuthorization {
				request.Header.Set("Authorization", tt.authorization)
			}
			if tt.apiKey != "" {
				request.Header.Set("X-Api-Key", tt.apiKey)
			}
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if called {
				t.Error("next handler was called")
			}
			if recorder.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
			if tt.setAuthorization && validatorCalled {
				t.Error("API-key validator called after invalid JWT")
			}
		})
	}
}

func TestMakeHTTPJWTOrAPIKeyMiddlewareReportsJWTConfigurationErrorWithoutFallback(t *testing.T) {
	called := false
	validatorCalled := false
	handler := MakeHTTPJWTOrAPIKeyMiddleware(nil, func(string) any {
		validatorCalled = true
		return struct{}{}
	})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+signMiddlewareToken(t, "alice"))
	request.Header.Set("X-Api-Key", "valid-api-key")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if called {
		t.Error("next handler was called")
	}
	if validatorCalled {
		t.Error("API-key validator called after JWT configuration error")
	}
	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if got, want := recorder.Body.String(), http.StatusText(http.StatusInternalServerError)+"\n"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
	if got := recorder.Body.String(); got == api.ErrJWTKeyFuncMissing.Error()+"\n" {
		t.Error("response exposed the missing key-function error")
	}
}

func TestAPIKeyRequestToContextUsesCanonicalHeader(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("x-api-key", "secret-key")

	ctx := APIKeyRequestToContext(context.Background(), request)

	if got := api.GetAPIKeyFromContext(ctx); got != "secret-key" {
		t.Errorf("API key = %q, want %q", got, "secret-key")
	}
}
