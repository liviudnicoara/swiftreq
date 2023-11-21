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
func Default() *RequestExecutor { return defaultRequestExecutor.Load().(*RequestExecutor) }

// SetDefault makes re the default RequestExecutor.
func SetDefault(re *RequestExecutor) {
	defaultRequestExecutor.Store(re)
}

// RequestExecutor is a struct representing an HTTP client with middleware support.
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

// newDefaultRequestExecutor creates a new default RequestExecutor with default settings.
func newDefaultRequestExecutor() *RequestExecutor {
	client := http.Client{Timeout: 30 * time.Second}
	return NewRequestExecutor(client)
}

// NewRequestExecutor creates a new RequestExecutor with the provided http.Client.
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

// WithTimeout sets the timeout for the RequestExecutor.
func (re *RequestExecutor) WithTimeout(timeout time.Duration) *RequestExecutor {
	re.client.Timeout = timeout
	return re
}

// WithMiddleware adds a single middleware to the RequestExecutor.
func (re *RequestExecutor) WithMiddleware(handler middlewares.Middleware) *RequestExecutor {
	re.middlewares = append(re.middlewares, handler)
	re.pipeline = re.do()

	for _, h := range re.middlewares {
		re.pipeline = h(re.pipeline)
	}

	return re
}

// WithMiddlewares adds multiple middlewares to the RequestExecutor.
func (re *RequestExecutor) WithMiddlewares(handlers ...middlewares.Middleware) *RequestExecutor {
	re.middlewares = append(re.middlewares, handlers...)
	re.pipeline = re.do()

	for _, h := range re.middlewares {
		re.pipeline = h(re.pipeline)
	}
	return re
}

// AddLogging adds logging middleware to the RequestExecutor.
func (re *RequestExecutor) AddLogging(logger *slog.Logger) *RequestExecutor {
	re.Logger = logger
	re.middlewares = append(re.middlewares, middlewares.LoggerMiddleware(logger))
	return re
}

// AddPerformanceMonitor adds performance monitoring middleware to the RequestExecutor.
func (re *RequestExecutor) AddPerformanceMonitor(threshold time.Duration, logger *slog.Logger) *RequestExecutor {
	re.Logger = logger
	re.middlewares = append(re.middlewares, middlewares.PerformanceMiddleware(threshold, logger))
	return re
}

// AddCaching adds caching middleware to the RequestExecutor with the specified TTL.
func (re *RequestExecutor) AddCaching(ttl time.Duration) *RequestExecutor {
	if re.cacheEnabled {
		return re
	}

	c := cache.New(ttl, 2*ttl)

	re.WithMiddleware(middlewares.CachingMiddleware(c, ttl))
	re.cacheEnabled = true

	return re
}

// WithExponentialRetry adds exponential retry middleware to the RequestExecutor with the specified retry count.
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

// WithLinearRetry adds linear retry middleware to the RequestExecutor with the specified retry count.
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

// WithAuthorization adds authorization middleware to the RequestExecutor with the specified schema and authorization function.
func (re *RequestExecutor) WithAuthorization(schema string, authorize middlewares.AuthorizeFunc) *RequestExecutor {
	if re.authEnabled {
		return re
	}

	tr := middlewares.NewTokenRefresher(schema, authorize, re.Logger)

	re.WithMiddleware(middlewares.AuthorizeMiddleware(tr))
	re.retryEnabled = true

	return re
}

// do returns a function that executes the HTTP request using the RequestExecutor's http.Client.
func (re *RequestExecutor) do() func(req *http.Request) (*http.Response, error) {
	return func(req *http.Request) (*http.Response, error) {
		return re.client.Do(req)
	}
}
