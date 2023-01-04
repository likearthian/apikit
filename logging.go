package apikit

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	log "github.com/likearthian/apikit/logger"
	"github.com/likearthian/go-http/router"
)

func MakeEndpointLoggingMiddleware(logger log.Logger, endPointMethod string) endpoint.Middleware {
	if logger == nil {
		return nil
	}

	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
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

			var result interface{}
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
