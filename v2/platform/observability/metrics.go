package observability

import (
	"context"
	nethttp "net/http"
	"time"

	httpkit "github.com/likearthian/apikit/v2/transport/http"
)

type contextKey int

const startTimeKey contextKey = iota

// MetricsRecorder receives observed HTTP request metrics. Implementations can
// ship these observations to Prometheus, Datadog, or any other metrics backend.
type MetricsRecorder interface {
	ObserveHTTPRequest(method, path string, code int, duration time.Duration, size int64)
}

// StartTimer stores the current time in ctx. Use it as a ServerBefore hook
// (via http.ServerBefore) so that the ServerFinalizer can compute request
// duration.
func StartTimer(ctx context.Context, _ *nethttp.Request) context.Context {
	return context.WithValue(ctx, startTimeKey, time.Now())
}

// ServerFinalizer returns a ServerFinalizerFunc that reads method, path,
// status code, duration, and response size from the context and passes them
// to rec. Use StartTimer as the corresponding ServerBefore hook.
func ServerFinalizer(rec MetricsRecorder) httpkit.ServerFinalizerFunc {
	return func(ctx context.Context, code int, r *nethttp.Request) {
		start, ok := ctx.Value(startTimeKey).(time.Time)
		if !ok {
			return
		}

		method, _ := ctx.Value(httpkit.ContextKeyRequestMethod).(string)
		path, _ := ctx.Value(httpkit.ContextKeyRequestPath).(string)
		size, _ := ctx.Value(httpkit.ContextKeyResponseSize).(int64)

		rec.ObserveHTTPRequest(method, path, code, time.Since(start), size)
	}
}
