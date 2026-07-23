package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
)

const testJWTKey = "01234567890123456789012345678901"

func signAuthToken(t *testing.T, method jwt.SigningMethod, key string, claims *AuthClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(method, claims)
	token.Header["kid"] = "0"

	raw, err := token.SignedString([]byte(key))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	return raw
}

func testTokenVerifier(key string, options ...JwtOption) *TokenVerifier {
	return NewTokenVerifier(func(*jwt.Token) (any, error) {
		return []byte(key), nil
	}, options...)
}

func verifyTokenWithoutPanic(t *testing.T, verifier *TokenVerifier, raw string) (*jwt.Token, error) {
	t.Helper()

	defer func() {
		if recover() != nil {
			t.Fatal("Verify() panicked")
		}
	}()

	return verifier.Verify(raw)
}

func TestDefaultJwtOptions(t *testing.T) {
	options := DefaultJwtOptions()

	if options.ClaimFactory == nil {
		t.Fatal("DefaultJwtOptions().ClaimFactory = nil")
	}
	if claims := options.ClaimFactory(); claims == nil {
		t.Fatal("DefaultJwtOptions().ClaimFactory() = nil")
	} else if _, ok := claims.(*AuthClaims); !ok {
		t.Errorf("DefaultJwtOptions().ClaimFactory() = %T, want *AuthClaims", claims)
	}
	if got, want := options.JwtSigningMethod.Alg(), jwt.SigningMethodHS256.Alg(); got != want {
		t.Errorf("DefaultJwtOptions().JwtSigningMethod.Alg() = %q, want %q", got, want)
	}
	if options.ParserOptions == nil {
		t.Error("DefaultJwtOptions().ParserOptions = nil, want empty non-nil slice")
	} else if len(options.ParserOptions) != 0 {
		t.Errorf("len(DefaultJwtOptions().ParserOptions) = %d, want 0", len(options.ParserOptions))
	}
}

func TestTokenVerifierAcceptsValidHS256Token(t *testing.T) {
	raw := signAuthToken(t, jwt.SigningMethodHS256, testJWTKey, &AuthClaims{
		Username: "alice",
	})

	token, err := testTokenVerifier(testJWTKey).Verify(raw)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !token.Valid {
		t.Fatal("Verify() returned an invalid token")
	}

	claims, ok := token.Claims.(*AuthClaims)
	if !ok {
		t.Fatalf("Verify() claims = %T, want *AuthClaims", token.Claims)
	}
	if got, want := claims.Username, "alice"; got != want {
		t.Errorf("Verify() username = %q, want %q", got, want)
	}
}

func TestTokenVerifierRejectsUnexpectedSigningMethod(t *testing.T) {
	raw := signAuthToken(t, jwt.SigningMethodHS384, testJWTKey, &AuthClaims{
		Username: "alice",
	})

	_, err := testTokenVerifier(testJWTKey).Verify(raw)
	if !errors.Is(err, ErrUnexpectedSigningMethod) {
		t.Fatalf("Verify() error = %v, want ErrUnexpectedSigningMethod", err)
	}
}

type spoofHS256Method struct{}

func (*spoofHS256Method) Alg() string {
	return jwt.SigningMethodHS256.Alg()
}

func (*spoofHS256Method) Sign(signingString string, key any) (string, error) {
	return jwt.SigningMethodHS256.Sign(signingString, key)
}

func (*spoofHS256Method) Verify(signingString, signature string, key any) error {
	return jwt.SigningMethodHS256.Verify(signingString, signature, key)
}

func TestTokenVerifierRejectsDifferentImplementationRegisteredWithExpectedAlgorithm(t *testing.T) {
	raw := signAuthToken(t, jwt.SigningMethodHS256, testJWTKey, &AuthClaims{
		Username: "alice",
	})

	algorithm := jwt.SigningMethodHS256.Alg()
	originalMethod := jwt.GetSigningMethod(algorithm)
	t.Cleanup(func() {
		jwt.RegisterSigningMethod(algorithm, func() jwt.SigningMethod {
			return originalMethod
		})
	})

	spoofMethod := &spoofHS256Method{}
	jwt.RegisterSigningMethod(algorithm, func() jwt.SigningMethod {
		return spoofMethod
	})

	_, err := testTokenVerifier(testJWTKey).Verify(raw)
	if !errors.Is(err, ErrUnexpectedSigningMethod) {
		t.Fatalf("Verify() error = %v, want ErrUnexpectedSigningMethod", err)
	}
}

func TestTokenVerifierRejectsMalformedToken(t *testing.T) {
	_, err := testTokenVerifier(testJWTKey).Verify("not-a-jwt")
	if !errors.Is(err, ErrTokenMalformed) {
		t.Fatalf("Verify() error = %v, want ErrTokenMalformed", err)
	}
}

func TestTokenVerifierRejectsExpiredToken(t *testing.T) {
	raw := signAuthToken(t, jwt.SigningMethodHS256, testJWTKey, &AuthClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: jwt.At(time.Now().Add(-time.Minute)),
		},
		Username: "alice",
	})

	_, err := testTokenVerifier(testJWTKey).Verify(raw)
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("Verify() error = %v, want ErrTokenExpired", err)
	}
}

func TestTokenVerifierRejectsTokenThatIsNotActive(t *testing.T) {
	raw := signAuthToken(t, jwt.SigningMethodHS256, testJWTKey, &AuthClaims{
		StandardClaims: jwt.StandardClaims{
			NotBefore: jwt.At(time.Now().Add(time.Minute)),
		},
		Username: "alice",
	})

	_, err := testTokenVerifier(testJWTKey).Verify(raw)
	if !errors.Is(err, ErrTokenNotActive) {
		t.Fatalf("Verify() error = %v, want ErrTokenNotActive", err)
	}
}

func TestTokenVerifierRejectsBadSignatureWithoutLeakingDetails(t *testing.T) {
	raw := signAuthToken(t, jwt.SigningMethodHS256, "different-signing-key", &AuthClaims{
		Username: "alice",
	})

	_, err := testTokenVerifier(testJWTKey).Verify(raw)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Verify() error = %v, want ErrUnauthorized", err)
	}
	if strings.Contains(strings.ToLower(err.Error()), "signature") {
		t.Errorf("Verify() error leaks signature details: %q", err)
	}
}

func TestTokenVerifierRejectsKeyFuncErrorWithoutLeakingDetails(t *testing.T) {
	const sensitiveDetail = "secret key backend unavailable"
	raw := signAuthToken(t, jwt.SigningMethodHS256, testJWTKey, &AuthClaims{
		Username: "alice",
	})
	verifier := NewTokenVerifier(func(*jwt.Token) (any, error) {
		return nil, errors.New(sensitiveDetail)
	})

	_, err := verifier.Verify(raw)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Verify() error = %v, want ErrUnauthorized", err)
	}
	if strings.Contains(err.Error(), sensitiveDetail) {
		t.Errorf("Verify() error leaks key function details: %q", err)
	}
}

func TestTokenVerifierRejectsEmptyToken(t *testing.T) {
	_, err := testTokenVerifier(testJWTKey).Verify("")
	if !errors.Is(err, ErrTokenContextMissing) {
		t.Fatalf("Verify() error = %v, want ErrTokenContextMissing", err)
	}
}

func TestTokenVerifierRejectsNilKeyFuncWithoutPanicking(t *testing.T) {
	raw := signAuthToken(t, jwt.SigningMethodHS256, testJWTKey, &AuthClaims{
		Username: "alice",
	})

	_, err := NewTokenVerifier(nil).Verify(raw)
	if !errors.Is(err, ErrJWTKeyFuncMissing) {
		t.Fatalf("Verify() error = %v, want ErrJWTKeyFuncMissing", err)
	}
}

func TestTokenVerifierRejectsNilClaimsWithoutPanicking(t *testing.T) {
	raw := signAuthToken(t, jwt.SigningMethodHS256, testJWTKey, &AuthClaims{
		Username: "alice",
	})
	verifier := testTokenVerifier(testJWTKey, WithClaimsFactory(func() jwt.Claims {
		return nil
	}))

	_, err := verifier.Verify(raw)
	if !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("Verify() error = %v, want ErrTokenInvalid", err)
	}
}

func TestTokenVerifierRejectsClaimsFactoryPanicWithoutLeakingDetails(t *testing.T) {
	const sensitiveDetail = "sensitive claims factory panic"
	raw := signAuthToken(t, jwt.SigningMethodHS256, testJWTKey, &AuthClaims{
		Username: "alice",
	})
	verifier := testTokenVerifier(testJWTKey, WithClaimsFactory(func() jwt.Claims {
		panic(sensitiveDetail)
	}))

	_, err := verifyTokenWithoutPanic(t, verifier, raw)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Verify() error = %v, want ErrUnauthorized", err)
	}
	if strings.Contains(err.Error(), sensitiveDetail) {
		t.Errorf("Verify() error leaks panic details: %q", err)
	}
}

func TestTokenVerifierRejectsNilClaimsFactoryPanic(t *testing.T) {
	raw := signAuthToken(t, jwt.SigningMethodHS256, testJWTKey, &AuthClaims{
		Username: "alice",
	})
	verifier := testTokenVerifier(testJWTKey, WithClaimsFactory(func() jwt.Claims {
		panic(nil)
	}))

	token, err := verifyTokenWithoutPanic(t, verifier, raw)
	if token != nil {
		t.Errorf("Verify() token = %#v, want nil", token)
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Verify() error = %v, want ErrUnauthorized", err)
	}
}

func TestTokenVerifierRejectsKeyFuncPanicWithoutLeakingDetails(t *testing.T) {
	const sensitiveDetail = "sensitive key function panic"
	raw := signAuthToken(t, jwt.SigningMethodHS256, testJWTKey, &AuthClaims{
		Username: "alice",
	})
	verifier := NewTokenVerifier(func(*jwt.Token) (any, error) {
		panic(sensitiveDetail)
	})

	_, err := verifyTokenWithoutPanic(t, verifier, raw)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Verify() error = %v, want ErrUnauthorized", err)
	}
	if strings.Contains(err.Error(), sensitiveDetail) {
		t.Errorf("Verify() error leaks panic details: %q", err)
	}
}

func TestTokenVerifierRejectsNilKeyFuncPanic(t *testing.T) {
	raw := signAuthToken(t, jwt.SigningMethodHS256, testJWTKey, &AuthClaims{
		Username: "alice",
	})
	verifier := NewTokenVerifier(func(*jwt.Token) (any, error) {
		panic(nil)
	})

	token, err := verifyTokenWithoutPanic(t, verifier, raw)
	if token != nil {
		t.Errorf("Verify() token = %#v, want nil", token)
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Verify() error = %v, want ErrUnauthorized", err)
	}
}

func TestTokenVerifierRejectsNilSigningMethod(t *testing.T) {
	raw := signAuthToken(t, jwt.SigningMethodHS256, testJWTKey, &AuthClaims{
		Username: "alice",
	})
	verifier := testTokenVerifier(testJWTKey, WithJwtSigningMethod(nil))

	_, err := verifyTokenWithoutPanic(t, verifier, raw)
	if !errors.Is(err, ErrUnexpectedSigningMethod) {
		t.Fatalf("Verify() error = %v, want ErrUnexpectedSigningMethod", err)
	}
}

func TestTokenVerifierAppliesOptionsOnlyWhenConstructed(t *testing.T) {
	applied := 0
	verifier := testTokenVerifier(testJWTKey, func(*jwtOption) {
		applied++
	})
	raw := signAuthToken(t, jwt.SigningMethodHS256, testJWTKey, &AuthClaims{
		Username: "alice",
	})

	for i := 0; i < 2; i++ {
		if _, err := verifier.Verify(raw); err != nil {
			t.Fatalf("Verify() call %d error = %v", i+1, err)
		}
	}
	if got, want := applied, 1; got != want {
		t.Errorf("option applied %d times, want %d", got, want)
	}
}

func TestWithJWTAuthEPMiddlewarePropagatesClaims(t *testing.T) {
	raw := signAuthToken(t, jwt.SigningMethodHS256, testJWTKey, &AuthClaims{
		Username: "alice",
	})
	keyFn := func(*jwt.Token) (any, error) {
		return []byte(testJWTKey), nil
	}
	secured := WithJWTAuthEPMiddleware(
		func(ctx context.Context, _ struct{}) (string, error) {
			claims, ok := ctx.Value(ContextKeyAuthClaims).(*AuthClaims)
			if !ok {
				t.Fatalf("auth claims = %T, want *AuthClaims", ctx.Value(ContextKeyAuthClaims))
			}
			return claims.Username, nil
		},
		keyFn,
	)

	ctx := context.WithValue(context.Background(), ContextKeyJWTToken, raw)
	username, err := secured(ctx, struct{}{})
	if err != nil {
		t.Fatalf("secured endpoint error = %v", err)
	}
	if got, want := username, "alice"; got != want {
		t.Errorf("secured endpoint username = %q, want %q", got, want)
	}
}

func TestWithJWTAuthEPMiddlewareRejectsMissingAndMalformedTokens(t *testing.T) {
	keyFn := func(*jwt.Token) (any, error) {
		return []byte(testJWTKey), nil
	}
	secured := WithJWTAuthEPMiddleware(
		func(context.Context, struct{}) (struct{}, error) {
			t.Fatal("secured endpoint called")
			return struct{}{}, nil
		},
		keyFn,
	)

	tests := []struct {
		name string
		ctx  context.Context
		want error
	}{
		{
			name: "missing",
			ctx:  context.Background(),
			want: ErrTokenContextMissing,
		},
		{
			name: "malformed",
			ctx:  context.WithValue(context.Background(), ContextKeyJWTToken, "not-a-jwt"),
			want: ErrTokenMalformed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := secured(tt.ctx, struct{}{}); !errors.Is(err, tt.want) {
				t.Fatalf("secured endpoint error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestWithJWTAuthEPMiddlewareFailuresMapToExpectedStatus(t *testing.T) {
	unexpectedMethodToken := signAuthToken(t, jwt.SigningMethodHS384, testJWTKey, &AuthClaims{
		Username: "alice",
	})
	validToken := signAuthToken(t, jwt.SigningMethodHS256, testJWTKey, &AuthClaims{
		Username: "alice",
	})

	tests := []struct {
		name       string
		keyFn      jwt.Keyfunc
		ctx        context.Context
		wantStatus int
	}{
		{
			name: "missing token context",
			keyFn: func(*jwt.Token) (any, error) {
				return []byte(testJWTKey), nil
			},
			ctx:        context.Background(),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "unexpected signing method",
			keyFn: func(*jwt.Token) (any, error) {
				return []byte(testJWTKey), nil
			},
			ctx:        context.WithValue(context.Background(), ContextKeyJWTToken, unexpectedMethodToken),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing key function",
			keyFn:      nil,
			ctx:        context.WithValue(context.Background(), ContextKeyJWTToken, validToken),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secured := WithJWTAuthEPMiddleware(
				func(context.Context, struct{}) (struct{}, error) {
					t.Fatal("secured endpoint called")
					return struct{}{}, nil
				},
				tt.keyFn,
			)

			_, err := secured(tt.ctx, struct{}{})
			if err == nil {
				t.Fatal("secured endpoint error = nil")
			}
			if got := Err2code(err); got != tt.wantStatus {
				t.Errorf("Err2code(secured endpoint error) = %d, want %d", got, tt.wantStatus)
			}
		})
	}
}

func TestParseJwtErrorUsesProjectSentinels(t *testing.T) {
	const generic = "not authorized to access this resource"
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "malformed", err: fmt.Errorf("wrapped: %w", ErrTokenMalformed), want: ErrTokenMalformed.Error()},
		{name: "expired", err: fmt.Errorf("wrapped: %w", ErrTokenExpired), want: ErrTokenExpired.Error()},
		{name: "not active", err: fmt.Errorf("wrapped: %w", ErrTokenNotActive), want: ErrTokenNotActive.Error()},
		{name: "invalid", err: fmt.Errorf("wrapped: %w", ErrTokenInvalid), want: ErrTokenInvalid.Error()},
		{name: "unexpected signing method", err: fmt.Errorf("wrapped: %w", ErrUnexpectedSigningMethod), want: generic},
		{name: "unknown", err: errors.New("sensitive internal detail"), want: generic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseJwtError(tt.err); got != tt.want {
				t.Errorf("ParseJwtError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreateTokenSupportsOneKey(t *testing.T) {
	keys := []string{testJWTKey}
	tokenString, err := CreateToken(StandardClaimsFactory, keys)
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}
	if tokenString == "" {
		t.Fatal("CreateToken() returned an empty token")
	}

	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, CreateJwtKeyGetterFunc(keys))
	if err != nil {
		t.Fatalf("ParseWithClaims() error = %v", err)
	}
	if !token.Valid {
		t.Fatal("parsed token is invalid")
	}
	if token.Method != jwt.SigningMethodHS256 {
		t.Errorf("token method = %v, want HS256", token.Method)
	}
	if got := token.Header["kid"]; got != "0" {
		t.Errorf("token kid = %#v, want %q", got, "0")
	}
}

func TestCreateTokenRejectsEmptyKeys(t *testing.T) {
	if _, err := CreateToken(StandardClaimsFactory, nil); err == nil {
		t.Fatal("CreateToken() error = nil, want an error")
	}
}

func TestCreateTokenRejectsNilClaimsFactory(t *testing.T) {
	if _, err := CreateToken(nil, []string{testJWTKey}); err == nil {
		t.Fatal("CreateToken() error = nil, want an error")
	}
}

func TestCreateTokenRejectsNilClaims(t *testing.T) {
	tests := []struct {
		name    string
		factory ClaimsFactory
	}{
		{
			name: "nil interface",
			factory: func() jwt.Claims {
				return nil
			},
		},
		{
			name: "typed nil",
			factory: func() jwt.Claims {
				var claims *AuthClaims
				return claims
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := CreateToken(tt.factory, []string{testJWTKey}); err == nil {
				t.Fatal("CreateToken() error = nil, want an error")
			}
		})
	}
}

func TestKeyGetterRejectsNilTokenWithoutPanicking(t *testing.T) {
	if _, err := getKey(nil, []string{testJWTKey}); err == nil {
		t.Fatal("getKey() error = nil, want an error")
	} else if got, want := err.Error(), "token is required"; got != want {
		t.Errorf("getKey() error = %q, want %q", got, want)
	}
}

func TestKeyGetterRejectsMissingKidWithoutPanicking(t *testing.T) {
	token := jwt.New(jwt.SigningMethodHS256)
	delete(token.Header, "kid")

	if _, err := getKey(token, []string{testJWTKey}); err == nil {
		t.Fatal("getKey() error = nil, want an error")
	}
}

func TestKeyGetterRejectsInvalidKidWithoutPanicking(t *testing.T) {
	tests := []struct {
		name         string
		kid          any
		keys         []string
		wantNumError bool
	}{
		{name: "non-string", kid: 1, keys: []string{testJWTKey}},
		{name: "empty", kid: "", keys: []string{testJWTKey}},
		{name: "negative", kid: "-1", keys: []string{testJWTKey}},
		{name: "non-integer", kid: "not-an-integer", keys: []string{testJWTKey}, wantNumError: true},
		{name: "overflowing integer", kid: "999999999999999999999999999999999", keys: []string{testJWTKey}, wantNumError: true},
		{name: "out-of-range", kid: "1", keys: []string{testJWTKey}},
		{name: "empty keys", kid: "0", keys: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := jwt.New(jwt.SigningMethodHS256)
			token.Header["kid"] = tt.kid

			_, err := getKey(token, tt.keys)
			if err == nil {
				t.Fatal("getKey() error = nil, want an error")
			}
			if tt.wantNumError {
				var numError *strconv.NumError
				if !errors.As(err, &numError) {
					t.Errorf("getKey() error = %v, want wrapped *strconv.NumError", err)
				}
			}
		})
	}
}
