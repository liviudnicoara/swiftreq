package swiftreq

import (
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/liviudnicoara/swiftreq/middlewares"
	"github.com/patrickmn/go-cache"
)

var (
	defaultMinWaitRetry = 500 * time.Millisecond
	defaultMaxWaitRetry = 10 * time.Second

	defaultRequestExecutor atomic.Value
)

func init() {
	defaultRequestExecutor.Store(newDefaultRequestExecutor())
}

// Default returns the default RequestExecutor.
func Default() *RequestExecutor { return defaultRequestExecutor.Load().(*RequestExecutor) }

// SetDefault makes re the default RequestExecutor.
func SetDefault(re *RequestExecutor) {
	defaultRequestExecutor.Store(re)
}

type RequestExecutor struct {
	client       http.Client
	middlewares  []middlewares.Middleware
	pipeline     middlewares.Handler
	cacheEnabled bool
	retryEnabled bool
	authEnabled  bool

	MinWaitRetry time.Duration
	MaxWaitRetry time.Duration

	Logger *slog.Logger
}

func newDefaultRequestExecutor() *RequestExecutor {
	client := http.Client{Timeout: 30 * time.Second}
	return NewRequestExecutor(client)
}

func NewRequestExecutor(client http.Client) *RequestExecutor {
	re := &RequestExecutor{
		client: client,

		MinWaitRetry: defaultMinWaitRetry,
		MaxWaitRetry: defaultMaxWaitRetry,
		Logger:       slog.Default(),
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

func (re *RequestExecutor) AddLogging(logger *slog.Logger) *RequestExecutor {
	re.Logger = logger
	re.middlewares = append(re.middlewares, middlewares.LoggerMiddleware(logger))
	return re
}

func (re *RequestExecutor) AddPerformanceMonitor(threshold time.Duration, logger *slog.Logger) *RequestExecutor {
	re.Logger = logger
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

func (re *RequestExecutor) WithExponentialRetry(retry int) *RequestExecutor {
	if re.retryEnabled {
		return re
	}

	rh := middlewares.RetryHandler{
		MinWait:    re.MinWaitRetry,
		MaxWait:    re.MaxWaitRetry,
		RetryCount: retry,
		Backoff:    middlewares.ExponentialBackoffTime,
	}

	re.WithMiddleware(middlewares.RetryMiddleware(rh))
	re.retryEnabled = true

	return re
}

func (re *RequestExecutor) WithLinearRetry(retry int) *RequestExecutor {
	if re.retryEnabled {
		return re
	}

	rh := middlewares.RetryHandler{
		MinWait:    re.MinWaitRetry,
		MaxWait:    re.MaxWaitRetry,
		RetryCount: retry,
		Backoff:    middlewares.LinearJitterBackoffTime,
	}

	re.WithMiddleware(middlewares.RetryMiddleware(rh))
	re.retryEnabled = true

	return re
}

func (re *RequestExecutor) WithAuthorization(schema string, authorize middlewares.AuthorizeFunc) *RequestExecutor {
	if re.authEnabled {
		return re
	}

	tr := middlewares.NewTokenRefresher(schema, authorize, re.Logger)

	re.WithMiddleware(middlewares.AuthorizeMiddleware(tr))
	re.retryEnabled = true

	return re
}

func (re *RequestExecutor) do() func(req *http.Request) (*http.Response, error) {
	return func(req *http.Request) (*http.Response, error) {
		return re.client.Do(req)
	}
}
