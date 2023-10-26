package middlewares

import (
	"fmt"
	"net/http"
)

func LogMiddleware() Middleware {
	return func(next Handler) Handler {
		return func(r *http.Request) (*http.Response, error) {
			fmt.Println("performing req", r.URL)

			response, err := next(r)
			fmt.Println("ended req", r.URL)

			return response, err
		}
	}
}
