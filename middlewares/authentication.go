package middlewares

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// lifeSpanSafetyMargin defines the safety margin for token lifespan.
var (
	lifeSpanSafetyMargin = 1 * time.Second
)

// tokenInfo represents the information about an access token.
type tokenInfo struct {
	Token string
	Error error
}

// TokenRefresher is a struct responsible for refreshing access tokens.
type TokenRefresher struct {
	accessToken chan tokenInfo
	logger      *slog.Logger
	authorize   AuthorizeFunc

	Schema string
}

// AuthorizeFunc is a function type for obtaining access tokens.
type AuthorizeFunc func() (token string, lifeSpan time.Duration, err error)

// NewTokenRefresher creates a new TokenRefresher with the specified schema, authorization function, and logger.
func NewTokenRefresher(schema string, fn AuthorizeFunc, logger *slog.Logger) *TokenRefresher {
	tr := &TokenRefresher{
		accessToken: make(chan tokenInfo),
		logger:      logger,
		authorize:   fn,

		Schema: schema,
	}

	tr.RefreshToken()

	return tr
}

// RefreshToken refreshes the access token periodically.
func (tr *TokenRefresher) RefreshToken() {
	started := make(chan struct{})

	go func() {
		var err error
		var token string
		var lifeSpan time.Duration
		token, lifeSpan, err = tr.authorize()
		expired := time.After(lifeSpan - lifeSpanSafetyMargin)
		if err != nil {
			tr.logger.Error("Could not retrieve access token", err)
		}

		<-started

		for {
			select {
			case tr.accessToken <- tokenInfo{Token: token, Error: err}:
			case <-expired:
				token, lifeSpan, err = tr.authorize()
				expired = time.After(lifeSpan - lifeSpanSafetyMargin)
				if err != nil {
					tr.logger.Error("Could not retrieve access token", err)
				}
			}

		}
	}()

	started <- struct{}{}
	close(started)
}

// Get retrieves the current access token.
func (tr *TokenRefresher) Get() (string, error) {
	tokenInfo := <-tr.accessToken
	return tokenInfo.Token, tokenInfo.Error
}

// AuthorizeMiddleware creates a middleware that adds the Authorization header to the HTTP request using the TokenRefresher.
func AuthorizeMiddleware(tr *TokenRefresher) Middleware {
	return func(next Handler) Handler {
		return func(req *http.Request) (*http.Response, error) {
			token, err := tr.Get()
			if err != nil {
				tr.logger.Warn("No token will be added to the request", "URL", req.URL, "Method", req.Method, "Error", err)
			} else {
				req.Header.Add("Authorization", fmt.Sprintf("%s %s", tr.Schema, token))
			}

			return next(req)
		}
	}
}
