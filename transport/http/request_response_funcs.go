package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/likearthian/apikit/api"
)

const (
	bearer       string = "bearer"
	bearerFormat string = "Bearer %s"
)

// RequestFunc may take information from an HTTP request and put it into a
// request context. In Servers, RequestFuncs are executed prior to invoking the
// endpoint. In Clients, RequestFuncs are executed after creating the request
// but prior to invoking the HTTP client.
type RequestFunc func(context.Context, *http.Request) context.Context

// ServerResponseFunc may take information from a request context and use it to
// manipulate a ResponseWriter. ServerResponseFuncs are only executed in
// servers, after invoking the endpoint but prior to writing a response.
type ServerResponseFunc func(context.Context, http.ResponseWriter) context.Context

// ClientResponseFunc may take information from an HTTP request and make the
// response available for consumption. ClientResponseFuncs are only executed in
// clients, after a request has been made, but prior to it being decoded.
type ClientResponseFunc func(context.Context, *http.Response) context.Context

// SetContentType returns a ServerResponseFunc that sets the Content-Type header
// to the provided value.
func SetContentType(contentType string) ServerResponseFunc {
	return SetResponseHeader("Content-Type", contentType)
}

// SetResponseHeader returns a ServerResponseFunc that sets the given header.
func SetResponseHeader(key, val string) ServerResponseFunc {
	return func(ctx context.Context, w http.ResponseWriter) context.Context {
		w.Header().Set(key, val)
		return ctx
	}
}

// SetRequestHeader returns a RequestFunc that sets the given header.
func SetRequestHeader(key, val string) RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		r.Header.Set(key, val)
		return ctx
	}
}

// PopulateRequestContext is a RequestFunc that populates several values into
// the context from the HTTP request. Those values may be extracted using the
// corresponding ContextKey type in this package.
func PopulateRequestContext(ctx context.Context, r *http.Request) context.Context {
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}

	for k, v := range map[api.ContextKey]string{
		api.ContextKeyRequestMethod:          r.Method,
		api.ContextKeyRequestURI:             r.RequestURI,
		api.ContextKeyRequestPath:            r.URL.Path,
		api.ContextKeyRequestProto:           r.Proto,
		api.ContextKeyRequestHost:            r.Host,
		api.ContextKeyRequestRemoteAddr:      r.RemoteAddr,
		api.ContextKeyRequestXForwardedFor:   r.Header.Get("X-Forwarded-For"),
		api.ContextKeyRequestXForwardedProto: r.Header.Get("X-Forwarded-Proto"),
		api.ContextKeyRequestAuthorization:   r.Header.Get("Authorization"),
		api.ContextKeyRequestReferer:         r.Header.Get("Referer"),
		api.ContextKeyRequestUserAgent:       r.Header.Get("User-Agent"),
		api.ContextKeyRequestXRequestID:      r.Header.Get("X-Request-Id"),
		api.ContextKeyRequestAccept:          r.Header.Get("Accept"),
		api.ContextKeyRequestAcceptEncoding:  r.Header.Get("Accept-Encoding"),
		api.ContextKeyRequestXTraceID:        r.Header.Get("X-Trace-Id"),
		api.ContextKeyRequestDatetime:        r.Header.Get("datetime"),
		api.ContextKeyRequestSignature:       r.Header.Get("signature"),
		api.ContextKeyRequestScheme:          scheme,
	} {
		ctx = context.WithValue(ctx, k, v)
	}
	return ctx
}

func JWTHTTPRequestToContext(ctx context.Context, r *http.Request) context.Context {
	token, ok := extractTokenFromAuthHeader(r.Header.Get("authorization"))
	if !ok {
		return ctx
	}

	return context.WithValue(ctx, api.ContextKeyJWTToken, token)
}

func extractTokenFromAuthHeader(val string) (token string, ok bool) {
	authHeaderParts := strings.Split(val, " ")
	if len(authHeaderParts) != 2 || !strings.EqualFold(authHeaderParts[0], bearer) {
		return "", false
	}

	return authHeaderParts[1], true
}
