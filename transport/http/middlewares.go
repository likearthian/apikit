package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/likearthian/apikit/api"
)

func MakeHttpJwtMiddleware(keyFn jwt.Keyfunc, options ...api.JwtOption) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := TokenFromHeader(r)
			if tokenString == "" {
				http.Error(w, "Not Authorized", http.StatusUnauthorized)
				return
			}

			opt := api.DefaultJwtOptions()
			for _, o := range options {
				o(opt)
			}

			if opt.ClaimFactory == nil {
				opt.ClaimFactory = api.StandardClaimsFactory
			}

			var jwtSigningMethod jwt.SigningMethod = jwt.SigningMethodHS256
			if opt.JwtSigningMethod != nil {
				jwtSigningMethod = opt.JwtSigningMethod
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
					return nil, api.ErrUnexpectedSigningMethod
				}

				return keyFn(token)
			}, opt.ParserOptions...)

			if err != nil {
				http.Error(w, api.ParseJwtError(err), http.StatusUnauthorized)
				return
			}

			if !token.Valid {
				http.Error(w, api.ErrTokenInvalid.Error(), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), api.ContextKeyJWTToken, tokenString)
			ctx = context.WithValue(ctx, api.ContextKeyAuthClaims, token.Claims)

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

func MakeHttpApikeyMiddleware(validateFn func(apikey string) any) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apikey := ApikeyFromHeader(r)
			if apikey == "" {
				http.Error(w, "Apikey required. Authorized", http.StatusUnauthorized)
				return
			}

			claims := validateFn(apikey)
			if claims == nil {
				http.Error(w, "Apikey Validation failed. Not Authorized", http.StatusUnauthorized)
				return
			}

			r.WithContext(context.WithValue(r.Context(), api.ContextKeyAuthClaims, claims))
			r.WithContext(context.WithValue(r.Context(), api.ContextKeyApikey, apikey))

			next.ServeHTTP(w, r)
		})
	}
}

func MakeHttpJwtAndApikeyMiddleware(jwtKeyFn jwt.Keyfunc, apikeyValidateFn func(apikey string) any, options ...api.JwtOption) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := TokenFromHeader(r)
			apikey := ApikeyFromHeader(r)

			if tokenString == "" && apikey == "" {
				http.Error(w, "Not Authorized", http.StatusUnauthorized)
				return
			}

			opt := api.DefaultJwtOptions()
			for _, o := range options {
				o(opt)
			}

			if opt.ClaimFactory == nil {
				opt.ClaimFactory = api.StandardClaimsFactory
			}

			var jwtSigningMethod jwt.SigningMethod = jwt.SigningMethodHS256
			if opt.JwtSigningMethod != nil {
				jwtSigningMethod = opt.JwtSigningMethod
			}

			if tokenString != "" {
				token, err := jwt.ParseWithClaims(tokenString, opt.ClaimFactory(), func(token *jwt.Token) (interface{}, error) {
					// Don't forget to validate the alg is what you expect:
					if token.Method != jwtSigningMethod {
						return nil, api.ErrUnexpectedSigningMethod
					}

					return jwtKeyFn(token)
				}, opt.ParserOptions...)

				if err != nil {
					http.Error(w, api.ParseJwtError(err), http.StatusUnauthorized)
					return
				}

				if !token.Valid {
					http.Error(w, api.ErrTokenInvalid.Error(), http.StatusUnauthorized)
					return
				}

				r.WithContext(context.WithValue(r.Context(), api.ContextKeyAuthClaims, token.Claims))
				r.WithContext(context.WithValue(r.Context(), api.ContextKeyJWTToken, tokenString))
			} else {
				claims := apikeyValidateFn(apikey)
				if claims == nil {
					http.Error(w, "Apikey Validation failed. Not Authorized", http.StatusUnauthorized)
					return
				}

				r.WithContext(context.WithValue(r.Context(), api.ContextKeyAuthClaims, claims))
				r.WithContext(context.WithValue(r.Context(), api.ContextKeyApikey, apikey))
			}

			next.ServeHTTP(w, r)
		})
	}
}

func TokenFromHeader(r *http.Request) string {
	// Get token from authorization header.
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 7 && strings.ToUpper(bearer[0:7]) == "BEARER " {
		return bearer[7:]
	}
	return ""
}

func ApikeyFromHeader(r *http.Request) string {
	// Get token from authorization header.
	return r.Header.Get("X-Api-Key")
}
