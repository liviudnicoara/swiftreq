package swiftreq

import (
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/liviudnicoara/swiftreq/middlewares"
	"github.com/patrickmn/go-cache"
)

// defaultMinWaitRetry and defaultMaxWaitRetry define default values for minimum and maximum wait time between retries.
var (
	defaultMinWaitRetry = 500 * time.Millisecond
	defaultMaxWaitRetry = 10 * time.Second

	defaultRequestExecutor atomic.Value
)

// init initializes the default RequestExecutor with default settings.
func init() {
	defaultRequestExecutor.Store(newDefaultRequestExecutor())
}

// Default returns the default RequestExecutor.
func Default() *RequestExec { return defaultRequestExecutor.Load().(*RequestExec) }

// SetDefault makes re the default RequestExecutor.
func SetDefault(re *RequestExec) {
	defaultRequestExecutor.Store(re)
}

// RequestExec is a struct representing an HTTP client with middleware support.
type RequestExec struct {
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

// newDefaultRequestExecutor creates a new default RequestExecutor with default settings.
func newDefaultRequestExecutor() *RequestExec {
	client := http.Client{Timeout: 30 * time.Second}
	return NewRequestExecutor(client)
}

// NewRequestExecutor creates a new RequestExecutor with the provided http.Client.
func NewRequestExecutor(client http.Client) *RequestExec {
	re := &RequestExec{
		client: client,

		MinWaitRetry: defaultMinWaitRetry,
		MaxWaitRetry: defaultMaxWaitRetry,
		Logger:       slog.Default(),
	}

	re.pipeline = re.do()

	return re
}

// WithTimeout sets the timeout for the RequestExecutor.
func (re *RequestExec) WithTimeout(timeout time.Duration) *RequestExec {
	re.client.Timeout = timeout
	return re
}

// WithMiddleware adds a single middleware to the RequestExecutor.
func (re *RequestExec) WithMiddleware(handler middlewares.Middleware) *RequestExec {
	re.middlewares = append(re.middlewares, handler)
	re.pipeline = re.do()

	for _, h := range re.middlewares {
		re.pipeline = h(re.pipeline)
	}

	return re
}

// WithMiddlewares adds multiple middlewares to the RequestExecutor.
func (re *RequestExec) WithMiddlewares(handlers ...middlewares.Middleware) *RequestExec {
	re.middlewares = append(re.middlewares, handlers...)
	re.pipeline = re.do()

	for _, h := range re.middlewares {
		re.pipeline = h(re.pipeline)
	}
	return re
}

// AddLogging adds logging middleware to the RequestExecutor.
func (re *RequestExec) AddLogging(logger *slog.Logger) *RequestExec {
	re.Logger = logger
	re.middlewares = append(re.middlewares, middlewares.LoggerMiddleware(logger))
	return re
}

// AddPerformanceMonitor adds performance monitoring middleware to the RequestExecutor.
func (re *RequestExec) AddPerformanceMonitor(threshold time.Duration, logger *slog.Logger) *RequestExec {
	re.Logger = logger
	re.middlewares = append(re.middlewares, middlewares.PerformanceMiddleware(threshold, logger))
	return re
}

// AddCaching adds caching middleware to the RequestExecutor with the specified TTL.
func (re *RequestExec) AddCaching(ttl time.Duration) *RequestExec {
	if re.cacheEnabled {
		return re
	}

	c := cache.New(ttl, 2*ttl)

	re.WithMiddleware(middlewares.CachingMiddleware(c, ttl))
	re.cacheEnabled = true

	return re
}

// WithExponentialRetry adds exponential retry middleware to the RequestExecutor with the specified retry count.
func (re *RequestExec) WithExponentialRetry(retry int) *RequestExec {
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

// WithLinearRetry adds linear retry middleware to the RequestExecutor with the specified retry count.
func (re *RequestExec) WithLinearRetry(retry int) *RequestExec {
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

// WithAuthorization adds authorization middleware to the RequestExecutor with the specified schema and authorization function.
func (re *RequestExec) WithAuthorization(schema string, authorize middlewares.AuthorizeFunc) *RequestExec {
	if re.authEnabled {
		return re
	}

	tr := middlewares.NewTokenRefresher(schema, authorize, re.Logger)

	re.WithMiddleware(middlewares.AuthorizeMiddleware(tr))
	re.retryEnabled = true

	return re
}

// do returns a function that executes the HTTP request using the RequestExecutor's http.Client.
func (re *RequestExec) do() func(req *http.Request) (*http.Response, error) {
	return func(req *http.Request) (*http.Response, error) {
		return re.client.Do(req)
	}
}
