package middlewares

import "net/http"

type Handler func(req *http.Request) (*http.Response, error)

type Middleware func(next Handler) Handler
