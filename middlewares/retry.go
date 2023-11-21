package middlewares

import (
	"context"
	"crypto/x509"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

// redirectsErrorRe, schemeErrorRe, and notTrustedErrorRe are regular expressions to match specific errors.
var (
	redirectsErrorRe  = regexp.MustCompile(`stopped after \d+ redirects\z`)
	schemeErrorRe     = regexp.MustCompile(`unsupported protocol scheme`)
	notTrustedErrorRe = regexp.MustCompile(`certificate is not trusted`)
)

// RetryHandler defines parameters for retrying HTTP requests.
type RetryHandler struct {
	MinWait    time.Duration
	MaxWait    time.Duration
	RetryCount int
	Backoff    BackoffTime
}

// shouldRetry checks if the HTTP request should be retried based on the response and error.
func (rh *RetryHandler) shouldRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	if err != nil {
		if v, ok := err.(*url.Error); ok {
			if redirectsErrorRe.MatchString(v.Error()) {
				return false, v
			}

			if schemeErrorRe.MatchString(v.Error()) {
				return false, v
			}

			if notTrustedErrorRe.MatchString(v.Error()) {
				return false, v
			}

			if _, ok := v.Err.(x509.UnknownAuthorityError); ok {
				return false, v
			}
		}

		return true, nil
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return true, nil
	}

	if resp.StatusCode == 0 || (resp.StatusCode >= 500 && resp.StatusCode != http.StatusNotImplemented) {
		return true, fmt.Errorf("unexpected HTTP status %s", resp.Status)
	}

	return false, nil
}

// RetryMiddleware creates a middleware that retries HTTP requests based on the RetryHandler configuration.
func RetryMiddleware(rh RetryHandler) Middleware {
	return func(next Handler) Handler {
		return func(req *http.Request) (*http.Response, error) {
			var resp *http.Response
			var shouldRetry bool
			var err error
			var attempt int

			for ; ; attempt++ {
				resp, err = next(req)

				shouldRetry, err = rh.shouldRetry(req.Context(), resp, err)

				if !shouldRetry {
					break
				}

				remain := rh.RetryCount - attempt
				if remain <= 0 {
					break
				}

				wait := rh.Backoff(attempt, rh.MinWait, rh.MaxWait, resp)

				timer := time.NewTimer(wait)
				select {
				case <-req.Context().Done():
					timer.Stop()
					return nil, req.Context().Err()
				case <-timer.C:
				}

			}

			if err == nil && !shouldRetry {
				return resp, nil
			}

			if err == nil {
				return nil, fmt.Errorf("%s %s giving up after %d attempt(s)",
					req.Method, req.URL, attempt)
			}

			return nil, fmt.Errorf("%s %s giving up after %d attempt(s): %w",
				req.Method, req.URL, attempt, err)
		}
	}
}

// BackoffTime calculates how long to wait between retries.
type BackoffTime func(retry int, min, max time.Duration, resp *http.Response) time.Duration

// ExponentialBackoffTime will perform exponential backoff based on the retry
// The time will be between minimum and maximum durations.
// If response contains Retry-After header when a http.StatusTooManyRequests is found in the resp parameter,
// it will return the number of seconds set by the server.
func ExponentialBackoffTime(retry int, min, max time.Duration, resp *http.Response) time.Duration {
	if resp != nil {
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			if s, ok := resp.Header["Retry-After"]; ok {
				if sleep, err := strconv.ParseInt(s[0], 10, 64); err == nil {
					return time.Second * time.Duration(sleep)
				}
			}
		}
	}

	wait := math.Pow(2, float64(retry)) * float64(min)
	duration := time.Duration(int(wait))
	if duration > max {
		duration = max
	}

	return duration
}

// LinearJitterBackoffTime willperform linear backoff based on the retry count with jitter.
// min and max here are *not* absolute values. The number to be multiplied by
// the attempt number will be chosen at random from between them, thus they are
// bounding the jitter.
//
// Examples:
// No jitter: min = max = 1s
// Small jitter: min = 700ms max = 1300 ms
// Big jitter: min = 100 ms max = 10s
func LinearJitterBackoffTime(retry int, min, max time.Duration, resp *http.Response) time.Duration {
	if retry == 0 {
		retry = 1
	}

	if max <= min {
		return min * time.Duration(retry)
	}

	rand := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))

	jitter := rand.Float64() * float64(max-min)
	jitterMin := int64(jitter) + int64(min)
	return time.Duration(jitterMin * int64(retry))
}
