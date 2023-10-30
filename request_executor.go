package swiftreq

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/liviudnicoara/swiftreq/middlewares"
)

var (
	DefaultRequestExecutor = NewDefaultRequestExecutor()
)

type RequestExecutor struct {
	client      http.Client
	middlewares []middlewares.Middleware
	pipeline    middlewares.Handler
}

func NewDefaultRequestExecutor() *RequestExecutor {
	client := http.Client{Timeout: 30 * time.Second}
	return NewRequestExecutor(client)
}

func NewRequestExecutor(client http.Client) *RequestExecutor {
	re := &RequestExecutor{
		client: client,
	}

	re.WithMiddlewares(middlewares.LoggerMiddleware(*slog.Default()), middlewares.PerformanceMiddleware(1*time.Second, *slog.Default()))

	return re
}

func (re *RequestExecutor) WithTimeout(timeout time.Duration) *RequestExecutor {
	re.client.Timeout = timeout
	return re
}

func (re *RequestExecutor) WithMiddleware(handler middlewares.Middleware) *RequestExecutor {
	re.middlewares = append(re.middlewares, handler)
	re.pipeline = func(req *http.Request) (*http.Response, error) {
		return re.client.Do(req)
	}

	for _, h := range re.middlewares {
		re.pipeline = h(re.pipeline)
	}

	return re
}

func (re *RequestExecutor) WithMiddlewares(handlers ...middlewares.Middleware) *RequestExecutor {
	re.middlewares = append(re.middlewares, handlers...)
	re.pipeline = func(req *http.Request) (*http.Response, error) {
		return re.client.Do(req)
	}

	for _, h := range re.middlewares {
		re.pipeline = h(re.pipeline)
	}
	return re
}
