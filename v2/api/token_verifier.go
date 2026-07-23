package api

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/dgrijalva/jwt-go/v4"
)

// TokenVerifier parses and validates signed JWT tokens with a resolved set of
// options.
type TokenVerifier struct {
	keyFn   jwt.Keyfunc
	options *jwtOption
}

// NewTokenVerifier creates a TokenVerifier and resolves its options once.
func NewTokenVerifier(keyFn jwt.Keyfunc, options ...JwtOption) *TokenVerifier {
	resolved := DefaultJwtOptions()
	for _, option := range options {
		if option != nil {
			option(resolved)
		}
	}

	return &TokenVerifier{
		keyFn:   keyFn,
		options: resolved,
	}
}

// Verify parses raw with fresh claims and rejects any token that cannot be
// fully authenticated.
func (v *TokenVerifier) Verify(raw string) (token *jwt.Token, err error) {
	completed := false
	defer func() {
		if !completed {
			recover()
			token = nil
			err = normalizeJWTError(ErrUnauthorized)
		}
	}()

	token, err = v.verify(raw)
	completed = true
	return token, err
}

func (v *TokenVerifier) verify(raw string) (*jwt.Token, error) {
	if raw == "" {
		return nil, ErrTokenContextMissing
	}
	if v == nil || v.keyFn == nil {
		return nil, ErrJWTKeyFuncMissing
	}
	if v.options == nil || v.options.ClaimFactory == nil {
		return nil, wrapJWTError(ErrTokenInvalid)
	}

	claims := v.options.ClaimFactory()
	if claims == nil || isNilClaims(claims) {
		return nil, wrapJWTError(ErrTokenInvalid)
	}

	var keyError error
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token == nil || !sameSigningMethod(token.Method, v.options.JwtSigningMethod) {
			keyError = ErrUnexpectedSigningMethod
			return nil, keyError
		}

		key, err := v.keyFn(token)
		if err != nil {
			keyError = err
		}
		return key, err
	}, v.options.ParserOptions...)
	if err != nil {
		if keyError != nil {
			return nil, normalizeJWTError(keyError)
		}
		return nil, normalizeJWTError(err)
	}
	if token == nil || !token.Valid {
		return nil, wrapJWTError(ErrTokenInvalid)
	}

	return token, nil
}

func sameSigningMethod(actual, expected jwt.SigningMethod) bool {
	if isNilSigningMethod(actual) || isNilSigningMethod(expected) {
		return false
	}

	actualType := reflect.TypeOf(actual)
	if actualType != reflect.TypeOf(expected) || actual.Alg() != expected.Alg() {
		return false
	}

	actualValue := reflect.ValueOf(actual)
	if actualValue.Kind() == reflect.Ptr {
		return actualValue.Pointer() == reflect.ValueOf(expected).Pointer()
	}

	return reflect.DeepEqual(actual, expected)
}

func isNilSigningMethod(method jwt.SigningMethod) bool {
	if method == nil {
		return true
	}

	value := reflect.ValueOf(method)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func normalizeJWTError(err error) error {
	if err == nil {
		return nil
	}

	projectErrors := []error{
		ErrTokenContextMissing,
		ErrTokenInvalid,
		ErrTokenExpired,
		ErrTokenMalformed,
		ErrTokenNotActive,
		ErrUnexpectedSigningMethod,
		ErrJWTKeyFuncMissing,
		ErrUnauthorized,
	}
	for _, projectError := range projectErrors {
		if errors.Is(err, projectError) {
			return wrapJWTError(projectError)
		}
	}

	var malformed *jwt.MalformedTokenError
	if errors.As(err, &malformed) {
		return wrapJWTError(ErrTokenMalformed)
	}

	var expired *jwt.TokenExpiredError
	if errors.As(err, &expired) {
		return wrapJWTError(ErrTokenExpired)
	}

	var notValidYet *jwt.TokenNotValidYetError
	if errors.As(err, &notValidYet) {
		return wrapJWTError(ErrTokenNotActive)
	}

	return wrapJWTError(ErrUnauthorized)
}

func wrapJWTError(err error) error {
	return fmt.Errorf("JWT verification failed: %w", err)
}
