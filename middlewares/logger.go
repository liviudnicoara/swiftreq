package middlewares

import (
	"log/slog"
	"net/http"
)

func LoggerMiddleware(logger *slog.Logger) Middleware {
	return func(next Handler) Handler {
		return func(r *http.Request) (*http.Response, error) {
			logger.Info("Executing request", "URL", r.URL, "Method", r.Method)

			response, err := next(r)

			if err != nil {
				logger.Error("Error on request", "URL", r.URL, "Error", err.Error())
			}

			return response, err
		}
	}
}
