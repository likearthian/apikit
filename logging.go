package apikit

import (
	"context"
	"net/http"
	"time"

	"github.com/likearthian/apikit/api"
	log "github.com/likearthian/apikit/logger"
	"github.com/likearthian/go-http/router"
)

func WithEndpointLogging[I, O any](logger log.Logger, endPointMethod string, next api.Endpoint[I, O]) api.Endpoint[I, O] {
	if logger == nil {
		return next
	}

	return MakeEndpointLoggingMiddleware[I, O](logger, endPointMethod)(next)
}

func MakeEndpointLoggingMiddleware[I, O any](logger log.Logger, endPointMethod string) api.Middleware[I, O] {
	if logger == nil {
		return nil
	}

	return func(next api.Endpoint[I, O]) api.Endpoint[I, O] {
		return func(ctx context.Context, request I) (O, error) {
			reqid, ok := router.ReqIDFromContext(ctx)
			if !ok {
				reqid = ""
			}

			var fields = []interface{}{
				"event", "endpoint return",
				"request-id", reqid,
				"endpoint", endPointMethod,
				"ts", time.Now(),
			}

			var result O
			var err error
			isErrLog := false

			defer func(begin time.Time) {
				fields = append(fields, "duration", time.Since(begin))
				if err != nil {
					fields = append(fields, "error", err.Error())
					code := api.Err2code(err)
					if code == http.StatusInternalServerError {
						isErrLog = true
					}
				}

				if isErrLog {
					logger.Error("request failed", fields...)
					return
				}

				logger.Info("request success", fields...)
			}(time.Now())

			result, err = next(ctx, request)
			return result, err
		}
	}
}
