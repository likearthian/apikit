package apikit

import (
	"context"
	"time"

	"github.com/likearthian/apikit/api"
	log "github.com/likearthian/apikit/logger"
	"github.com/likearthian/go-http/router"
)

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
					code := Err2code(err)
					if code == 500 {
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
			if err != nil {
				result = ErrorResponse(reqid, 500, err)
			}
			return result, err
		}
	}
}
