package middlewares

import (
	"fmt"
	"net/http"
	"time"
)

func PerformanceMiddleware(threshold time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(req *http.Request) (*http.Response, error) {
			start := time.Now()

			resp, err := next(req)

			elapsed := time.Since(start)

			if elapsed > threshold {
				fmt.Println("request lasted", elapsed)
			}

			return resp, err
		}
	}
}
