package api

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

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
		ClaimFactory:  StandardClaimsFactory,
		ParserOptions: []jwt.ParserOption{},
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

func JWTMiddleware[I, O any](keyFn jwt.Keyfunc, options ...JwtOption) Middleware[I, O] {
	return func(next Endpoint[I, O]) Endpoint[I, O] {
		return WithJWTAuthEPMiddleware(next, keyFn, options...)
	}
}

func WithJWTAuthEPMiddleware[I, O any](ep Endpoint[I, O], keyFn jwt.Keyfunc, options ...JwtOption) Endpoint[I, O] {
	return func(ctx context.Context, request I) (O, error) {
		opt := jwtOption{}
		for _, o := range options {
			o(&opt)
		}

		if opt.ClaimFactory == nil {
			opt.ClaimFactory = StandardClaimsFactory
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
		token, err := jwt.ParseWithClaims(tokenString, opt.ClaimFactory(), func(token *jwt.Token) (interface{}, error) {
			// Don't forget to validate the alg is what you expect:
			if token.Method != jwtSigningMethod {
				return nil, ErrUnexpectedSigningMethod
			}

			return keyFn(token)
		}, opt.ParserOptions...)

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

		ctx = context.WithValue(ctx, ContextKeyAuthClaims, token.Claims)

		return ep(ctx, request)
	}
}

func ParseJwtError(err error) string {
	parsed := "not authorized to access this resource"
	// fmt.Println("jwt error:", err)
	switch err.(type) {
	case *jwt.MalformedTokenError:
		// Token is malformed
		parsed = ErrTokenMalformed.Error()
	case *jwt.TokenExpiredError:
		// Token is expired
		parsed = ErrTokenExpired.Error()
	case *jwt.TokenNotValidYetError:
		// Token is not active yet
		parsed = ErrTokenNotActive.Error()
	}

	return parsed
}

func DefaultJwtKeyGetterFunc(token *jwt.Token) (interface{}, error) {
	return getKey(token, DefaultKeys)
}

// CreateJwtKeyGetterFunc creates a jwt.Keyfunc that uses the given keys. the key will be chosen based on the kid in the token header.
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
}
