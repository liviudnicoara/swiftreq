package middlewares

import (
	"log/slog"
	"net/http"
	"time"
)

func PerformanceMiddleware(threshold time.Duration, logger *slog.Logger) Middleware {
	return func(next Handler) Handler {
		return func(req *http.Request) (*http.Response, error) {
			start := time.Now()

			resp, err := next(req)

			elapsed := time.Since(start)

			if elapsed > threshold {
				logger.Warn("Slow request", "URL", req.URL, "Elapsed", elapsed)
			}

			return resp, err
		}
	}
}
