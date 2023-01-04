package apikit

import (
	"errors"
	"net/http"
)

var ErrBucketNotFound = errors.New("bucket not found")
var ErrKeyAlreadyExists = errors.New("key already exists")
var ErrKeynotFound = errors.New("key not found")
var ErrBadRequest = errors.New("bad request")
var ErrInvalidUserPassword = errors.New("invalid user or password")
var ErrForbidden = errors.New("not authorized to access this resource")
var ErrUnauthorized = errors.New("unauthorized")
var ErrNoRow = errors.New("no row")

var (
	// ErrTokenContextMissing denotes a token was not passed into the parsing
	// middleware's context.
	ErrTokenContextMissing = errors.New("token up for parsing was not passed through the context")

	// ErrTokenInvalid denotes a token was not able to be validated.
	ErrTokenInvalid = errors.New("JWT Token was invalid")

	// ErrTokenExpired denotes a token's expire header (exp) has since passed.
	ErrTokenExpired = errors.New("JWT Token is expired")

	// ErrTokenMalformed denotes a token was not formatted as a JWT token.
	ErrTokenMalformed = errors.New("JWT Token is malformed")

	// ErrTokenNotActive denotes a token's not before header (nbf) is in the
	// future.
	ErrTokenNotActive = errors.New("token is not valid yet")

	// ErrUnexpectedSigningMethod denotes a token was signed with an unexpected
	// signing method.
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
)

func Err2code(err error) int {
	var status = http.StatusInternalServerError

	switch {
	case errors.Is(err, ErrKeynotFound):
		status = http.StatusNotFound
	case errors.Is(err, ErrBadRequest):
		status = http.StatusBadRequest
	case errors.Is(err, ErrInvalidUserPassword):
		status = http.StatusNetworkAuthenticationRequired
	case errors.Is(err, ErrUnauthorized):
		status = http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, ErrTokenExpired),
		errors.Is(err, ErrTokenInvalid),
		errors.Is(err, ErrTokenMalformed),
		errors.Is(err, ErrTokenNotActive):

		status = http.StatusUnauthorized
	}

	return status
}
