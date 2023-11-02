package middlewares

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

var (
	lifeSpanSaftyMargin = 1 * time.Second
)

type tokenInfo struct {
	Token string
	Error error
}

type TokenRefresher struct {
	accessToken chan tokenInfo
	logger      *slog.Logger
	authorize   AuthorizeFunc

	Schema string
}

type AuthorizeFunc func() (token string, lifeSpan time.Duration, err error)

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

func (tr *TokenRefresher) RefreshToken() {
	go func() {
		var err error
		var token string
		var lifeSpan time.Duration
		token, lifeSpan, err = tr.authorize()
		expired := time.After(lifeSpan - lifeSpanSaftyMargin)
		if err != nil {
			tr.logger.Error("Could not retrieve access token", err)
		}

		for {
			select {
			case tr.accessToken <- tokenInfo{Token: token, Error: err}:
			case <-expired:
				token, lifeSpan, err = tr.authorize()
				expired = time.After(lifeSpan - lifeSpanSaftyMargin)
				if err != nil {
					tr.logger.Error("Could not retrieve access token", err)
				}
			}

		}
	}()
}

func (tr *TokenRefresher) Get() (string, error) {
	tokenInfo := <-tr.accessToken
	return tokenInfo.Token, tokenInfo.Error
}

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
