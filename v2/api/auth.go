package api

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strconv"

	"github.com/dgrijalva/jwt-go/v4"
)

var jwtSigningMethod = jwt.SigningMethodHS256

type AuthClaims struct {
	jwt.StandardClaims
	Username string         `json:"username"`
	IsAdmin  bool           `json:"is_admin"`
	Meta     map[string]any `json:"meta"`
}

type jwtOption struct {
	ClaimFactory     ClaimsFactory
	JwtSigningMethod jwt.SigningMethod
	ParserOptions    []jwt.ParserOption
}

func DefaultJwtOptions() *jwtOption {
	return &jwtOption{
		ClaimFactory:     StandardClaimsFactory,
		JwtSigningMethod: jwt.SigningMethodHS256,
		ParserOptions:    []jwt.ParserOption{},
	}
}

type JwtOption func(*jwtOption)

func WithAudience(aud string) JwtOption {
	return func(opt *jwtOption) {
		opt.ParserOptions = append(opt.ParserOptions, jwt.WithAudience(aud))
	}
}

func WithClaimsFactory(claimFactory ClaimsFactory) JwtOption {
	return func(opt *jwtOption) {
		opt.ClaimFactory = claimFactory
	}
}

func WithJwtSigningMethod(method jwt.SigningMethod) JwtOption {
	return func(opt *jwtOption) {
		opt.JwtSigningMethod = method
	}
}

// ClaimsFactory is a factory for jwt.Claims.
// Useful in NewParser middleware.
type ClaimsFactory func() jwt.Claims

// MapClaimsFactory is a ClaimsFactory that returns
// an empty jwt.MapClaims.
func MapClaimsFactory() jwt.Claims {
	return jwt.MapClaims{}
}

// StandardClaimsFactory is a ClaimsFactory that returns
// an empty jwt.StandardClaims.
func StandardClaimsFactory() jwt.Claims {
	return &AuthClaims{}
}

func MakeClaimsFactory[T jwt.Claims](fn func() T) ClaimsFactory {
	return func() jwt.Claims {
		return fn()
	}
}

// CreateToken creates a JWT token with the given claimFactory and keys.
// keys is an array of 32 char long keys to be selected as a key used to sign the token.
//
// the selected key index is added to the token header as "kid".
// make sure to use the same arrays of key for verifying the token.
func CreateToken(claimFactory ClaimsFactory, keys []string) (string, error) {
	if claimFactory == nil {
		return "", fmt.Errorf("claims factory must not be nil")
	}
	if len(keys) == 0 {
		return "", fmt.Errorf("at least one signing key is required")
	}

	claims := claimFactory()
	if claims == nil || isNilClaims(claims) {
		return "", fmt.Errorf("claims factory must return non-nil claims")
	}

	selected, err := rand.Int(rand.Reader, big.NewInt(int64(len(keys))))
	if err != nil {
		return "", fmt.Errorf("failed to select JWT signing key: %w", err)
	}
	n := int(selected.Int64())

	token := jwt.NewWithClaims(jwtSigningMethod, claims)

	kid := strconv.Itoa(n)
	key := []byte(keys[n])

	token.Header["kid"] = kid
	return token.SignedString(key)
}

func isNilClaims(claims jwt.Claims) bool {
	value := reflect.ValueOf(claims)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func JWTMiddleware[I, O any](keyFn jwt.Keyfunc, options ...JwtOption) Middleware[I, O] {
	return func(next Endpoint[I, O]) Endpoint[I, O] {
		return WithJWTAuthEPMiddleware(next, keyFn, options...)
	}
}

func WithJWTAuthEPMiddleware[I, O any](ep Endpoint[I, O], keyFn jwt.Keyfunc, options ...JwtOption) Endpoint[I, O] {
	verifier := NewTokenVerifier(keyFn, options...)

	return func(ctx context.Context, request I) (O, error) {
		var out O

		tokenString, ok := ctx.Value(ContextKeyJWTToken).(string)
		if !ok {
			return out, ErrTokenContextMissing
		}

		token, err := verifier.Verify(tokenString)
		if err != nil {
			return out, err
		}

		ctx = context.WithValue(ctx, ContextKeyAuthClaims, token.Claims)

		return ep(ctx, request)
	}
}

func ParseJwtError(err error) string {
	switch {
	case errors.Is(err, ErrTokenMalformed):
		return ErrTokenMalformed.Error()
	case errors.Is(err, ErrTokenExpired):
		return ErrTokenExpired.Error()
	case errors.Is(err, ErrTokenNotActive):
		return ErrTokenNotActive.Error()
	case errors.Is(err, ErrTokenInvalid):
		return ErrTokenInvalid.Error()
	default:
		return "not authorized to access this resource"
	}
}

// CreateJwtKeyGetterFunc creates a jwt.Keyfunc that uses the given keys. the key will be chosen based on the kid in the token header.
func CreateJwtKeyGetterFunc(keys []string) jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		return getKey(token, keys)
	}
}

func getKey(token *jwt.Token, keys []string) (any, error) {
	if token == nil {
		return nil, fmt.Errorf("token is required")
	}

	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("kid header must be a string")
	}
	if kid == "" {
		return nil, fmt.Errorf("kid header must not be empty")
	}

	n, err := strconv.Atoi(kid)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kid header %q: %w", kid, err)
	}

	if n < 0 || n >= len(keys) {
		return nil, fmt.Errorf("kid index %d is out of range for %d keys", n, len(keys))
	}

	key := keys[n]
	return []byte(key), nil
}
