package middlewares

import (
	"net/http"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

func CachingMiddleware(c *cache.Cache, ttl time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(req *http.Request) (*http.Response, error) {
			if req.Method != "GET" {
				return next(req)
			}

			key := strings.ToLower(req.URL.String())

			if resp, ok := c.Get(key); ok {
				return resp.(*http.Response), nil
			}

			resp, err := next(req)

			if err != nil {
				c.Set(key, resp, ttl)
			}

			return resp, err
		}
	}
}
