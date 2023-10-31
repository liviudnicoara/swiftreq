package swiftreq

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/liviudnicoara/swiftreq/middlewares"
	"github.com/patrickmn/go-cache"
)

var (
	DefaultRequestExecutor = NewDefaultRequestExecutor()
)

type RequestExecutor struct {
	client       http.Client
	middlewares  []middlewares.Middleware
	pipeline     middlewares.Handler
	cacheEnabled bool
}

func NewDefaultRequestExecutor() *RequestExecutor {
	client := http.Client{Timeout: 30 * time.Second}
	return NewRequestExecutor(client)
}

func NewRequestExecutor(client http.Client) *RequestExecutor {
	re := &RequestExecutor{
		client: client,
	}

	re.pipeline = re.do()

	return re
}

func (re *RequestExecutor) WithTimeout(timeout time.Duration) *RequestExecutor {
	re.client.Timeout = timeout
	return re
}

func (re *RequestExecutor) WithMiddleware(handler middlewares.Middleware) *RequestExecutor {
	re.middlewares = append(re.middlewares, handler)
	re.pipeline = re.do()

	for _, h := range re.middlewares {
		re.pipeline = h(re.pipeline)
	}

	return re
}

func (re *RequestExecutor) WithMiddlewares(handlers ...middlewares.Middleware) *RequestExecutor {
	re.middlewares = append(re.middlewares, handlers...)
	re.pipeline = re.do()

	for _, h := range re.middlewares {
		re.pipeline = h(re.pipeline)
	}
	return re
}

func (re *RequestExecutor) AddLogging(logger slog.Logger) *RequestExecutor {
	re.middlewares = append(re.middlewares, middlewares.LoggerMiddleware(logger))
	return re
}

func (re *RequestExecutor) AddPerformanceMonitor(threshold time.Duration, logger slog.Logger) *RequestExecutor {
	re.middlewares = append(re.middlewares, middlewares.PerformanceMiddleware(threshold, logger))
	return re
}

func (re *RequestExecutor) AddCaching(ttl time.Duration) *RequestExecutor {
	if re.cacheEnabled {
		return re
	}

	c := cache.New(ttl, 2*ttl)

	re.WithMiddleware(middlewares.CachingMiddleware(c, ttl))
	re.cacheEnabled = true

	return re
}

func (re *RequestExecutor) do() func(req *http.Request) (*http.Response, error) {
	return func(req *http.Request) (*http.Response, error) {
		return re.client.Do(req)
	}
}
