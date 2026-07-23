package api

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestErr2codeMapsJWTVerificationFailuresToUnauthorized(t *testing.T) {
	authErrors := []struct {
		name string
		err  error
	}{
		{name: "unauthorized", err: ErrUnauthorized},
		{name: "token context missing", err: ErrTokenContextMissing},
		{name: "token invalid", err: ErrTokenInvalid},
		{name: "token expired", err: ErrTokenExpired},
		{name: "token malformed", err: ErrTokenMalformed},
		{name: "token not active", err: ErrTokenNotActive},
		{name: "unexpected signing method", err: ErrUnexpectedSigningMethod},
	}

	for _, authError := range authErrors {
		for _, wrapped := range []bool{false, true} {
			name := authError.name + "/direct"
			err := authError.err
			if wrapped {
				name = authError.name + "/wrapped"
				err = fmt.Errorf("verify token: %w", err)
			}

			t.Run(name, func(t *testing.T) {
				if got := Err2code(err); got != http.StatusUnauthorized {
					t.Errorf("Err2code() = %d, want %d", got, http.StatusUnauthorized)
				}
			})
		}
	}
}

func TestErr2codeKeepsMissingJWTKeyFunctionInternal(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "direct", err: ErrJWTKeyFuncMissing},
		{name: "wrapped", err: fmt.Errorf("configure verifier: %w", ErrJWTKeyFuncMissing)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Err2code(tt.err); got != http.StatusInternalServerError {
				t.Errorf("Err2code() = %d, want %d", got, http.StatusInternalServerError)
			}
		})
	}
}

func TestErr2codeKeepsUnknownErrorsInternal(t *testing.T) {
	if got := Err2code(errors.New("database unavailable")); got != http.StatusInternalServerError {
		t.Errorf("Err2code() = %d, want %d", got, http.StatusInternalServerError)
	}
}
