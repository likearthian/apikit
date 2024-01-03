package api

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
)

var (
	// ErrTokenContextMissing denotes a token was not passed into the parsing
	// middleware's context.
	ErrTokenContextMissing = fmt.Errorf("token up for parsing was not passed through the context")

	// ErrTokenInvalid denotes a token was not able to be validated.
	ErrTokenInvalid = fmt.Errorf("JWT Token was invalid")

	// ErrTokenExpired denotes a token's expire header (exp) has since passed.
	ErrTokenExpired = fmt.Errorf("JWT Token is expired")

	// ErrTokenMalformed denotes a token was not formatted as a JWT token.
	ErrTokenMalformed = fmt.Errorf("JWT Token is malformed")

	// ErrTokenNotActive denotes a token's not before header (nbf) is in the
	// future.
	ErrTokenNotActive = fmt.Errorf("token is not valid yet")

	// ErrUnexpectedSigningMethod denotes a token was signed with an unexpected
	// signing method.
	ErrUnexpectedSigningMethod = fmt.Errorf("unexpected signing method")
)

var jwtSigningMethod = jwt.SigningMethodHS256

type jwtOption struct {
	keyFunc       jwt.Keyfunc
	claimFactory  ClaimsFactory
	parserOptions []jwt.ParserOption
}

type JwtOption func(*jwtOption)

func WithAudience(aud string) JwtOption {
	return func(opt *jwtOption) {
		opt.parserOptions = append(opt.parserOptions, jwt.WithAudience(aud))
	}
}

func WithKeyGetter(keyGetter jwt.Keyfunc) JwtOption {
	return func(opt *jwtOption) {
		opt.keyFunc = keyGetter
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
	return &jwt.StandardClaims{}
}

// CreateToken creates a JWT token with the given claimFactory and keys.
// keys is an array of 32 char long keys to be selected as a key used to sign the token.
//
// the selected key index is added to the token header as "kid".
// make sure to use the same arrays of key for verifying the token.
func CreateToken(claimFactory ClaimsFactory, keys []string) (string, error) {
	claims := claimFactory()

	token := jwt.NewWithClaims(jwtSigningMethod, claims)
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	n := r.Intn(len(keys) - 1)

	// making sure n is between 0 and len(keys)-1, if not then set it to 1
	if n < 0 || n > len(keys)-1 {
		n = 1
	}
	kid := strconv.Itoa(n)
	key := []byte(keys[n])

	token.Header["kid"] = kid
	return token.SignedString(key)
}

func JWTMiddleware[I, O any](options ...JwtOption) Middleware[I, O] {
	return func(next Endpoint[I, O]) Endpoint[I, O] {
		return WithJWTAuthEPMiddleware(next, options...)
	}
}

func WithJWTAuthEPMiddleware[I, O any](ep Endpoint[I, O], options ...JwtOption) Endpoint[I, O] {
	return func(ctx context.Context, request I) (O, error) {
		opt := jwtOption{}
		for _, o := range options {
			o(&opt)
		}

		if opt.claimFactory == nil {
			opt.claimFactory = StandardClaimsFactory
		}

		if opt.keyFunc == nil {
			opt.keyFunc = DefaultJwtKeyGetterFunc
		}

		var out O
		// tokenString is stored in the context from the transport handlers.
		tokenString, ok := ctx.Value(ContextKeyJWTToken).(string)
		if !ok {
			return out, ErrTokenContextMissing
		}

		// Parse takes the token string and a function for looking up the
		// key. The latter is especially useful if you use multiple keys
		// for your application.  The standard is to use 'kid' in the head
		// of the token to identify which key to use, but the parsed token
		// (head and claims) is provided to the callback, providing
		// flexibility.
		token, err := jwt.ParseWithClaims(tokenString, opt.claimFactory(), func(token *jwt.Token) (interface{}, error) {
			// Don't forget to validate the alg is what you expect:
			if token.Method != jwtSigningMethod {
				return nil, ErrUnexpectedSigningMethod
			}

			return opt.keyFunc(token)
		}, opt.parserOptions...)

		if err != nil {
			switch err.(type) {
			case *jwt.MalformedTokenError:
				// Token is malformed
				return out, ErrTokenMalformed
			case *jwt.TokenExpiredError:
				// Token is expired
				return out, ErrTokenExpired
			case *jwt.TokenNotValidYetError:
				// Token is not active yet
				return out, ErrTokenNotActive
			}
			// We have a ValidationError but have no specific Go kit error for it.
			// Fall through to return original error.
			return out, fmt.Errorf("not authorized to access this resource")
		}

		if !token.Valid {
			return out, ErrTokenInvalid
		}

		ctx = context.WithValue(ctx, ContextKeyJWTClaims, token.Claims)

		return ep(ctx, request)
	}
}

func DefaultJwtKeyGetterFunc(token *jwt.Token) (interface{}, error) {
	return getKey(token, DefaultKeys)
}

func CreateJwtKeyGetterFunc(keys []string) jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		return getKey(token, keys)
	}
}

func getKey(token *jwt.Token, keys []string) (any, error) {
	kid := token.Header["kid"].(string)
	n, err := strconv.Atoi(kid)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the kid ID. %w", err)
	}

	if n > len(keys)-1 {
		return nil, fmt.Errorf("kid index is out of range")
	}

	key := keys[n]
	return []byte(key), nil
}

var DefaultKeys = []string{
	"6ai1Vz6dHy9PbLCKUc8QtadUIuOUMuHQ",
	"rUpWCnIwgvHfEpKJpXknmw5ozfBrpzbz",
	"bczvJVnrzXk5WHzSTm5GNMQo5nBfHnyK",
	"PQsc8LdzrV6Mn7Kq71E31N4vRhNpj30q",
	"W9X0Y1Z2A3B4C5D6E7F8G9H0I1J2K3L4",
	"M5N6O7P8Q9R0S1T2U3V4W5X6Y7Z8A9B0",
	"C1D2E3F4G5H6I7J8K9L0M1N2O3P4Q5R6",
	"S7T8U9V0W1X2Y3Z4A5B6C7D8E9F0G1H2",
	"I3J4K5L6M7N8O9P0Q1R2S3T4U5V6W7X8",
	"Y9Z0A1B2C3D4E5F6G7H8I9J0K1L2M3N4",
	"Z0A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5",
	"P6Q7R8S9T0U1V2W3X4Y5Z6A7B8C9D0E1",
	"F2G3H4I5J6K7L8M9N0O1P2Q3R4S5T6U7",
	"V8W9X0Y1Z2A3B4C5D6E7F8G9H0I1J2K3",
	"L4M5N6O7P8Q9R0S1T2U3V4W5X6Y7Z8A9",
	"B0C1D2E3F4G5H6I7J8K9L0M1N2O3P4Q5",
}
