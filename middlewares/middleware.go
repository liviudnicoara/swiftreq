package middlewares

import "net/http"

// Handler represents a function that processes an HTTP request and returns an HTTP response or an error.
type Handler func(req *http.Request) (*http.Response, error)

// Middleware represents a function that takes a Handler and returns a new Handler with additional behavior.
type Middleware func(next Handler) Handler
