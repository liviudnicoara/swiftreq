package swiftreq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Request represents an HTTP request with fluent methods for customization.
type Request[T any] struct {
	re              *RequestExecutor
	headers         map[string]string
	httpMethod      string
	url             string
	payload         interface{}
	queryParameters url.Values
}

// Get creates a new HTTP GET request.
func Get[T any](url string) *Request[T] {
	return newDefaultRequest[T]().
		WithMethod("GET").
		WithURL(url)
}

// Post creates a new HTTP POST request with the specified payload.
func Post[T any](url string, payload interface{}) *Request[T] {
	return newDefaultRequest[T]().
		WithMethod("POST").
		WithURL(url).
		WithPayload(payload)
}

// Put creates a new HTTP PUT request with the specified payload.
func Put[T any](url string, payload interface{}) *Request[T] {
	return newDefaultRequest[T]().
		WithMethod("PUT").
		WithURL(url).
		WithPayload(payload)
}

// Delete creates a new HTTP DELETE request.
func Delete[T any](url string) *Request[T] {
	return newDefaultRequest[T]().
		WithMethod("DELETE").
		WithURL(url)
}

// newDefaultRequest creates a new default Request with default settings.
func newDefaultRequest[T any]() *Request[T] {
	return newRequest[T](Default())
}

// newRequest creates a new Request with the specified RequestExecutor.
func newRequest[T any](re *RequestExecutor) *Request[T] {
	return &Request[T]{
		re: re,
		headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// WithMethod sets the HTTP method for the request.
func (r *Request[T]) WithMethod(httpMethod string) *Request[T] {
	r.httpMethod = httpMethod
	return r
}

// WithURL sets the URL for the request.
func (r *Request[T]) WithURL(url string) *Request[T] {
	r.url = url
	return r
}

// WithPayload sets the payload for the request.
func (r *Request[T]) WithPayload(payload interface{}) *Request[T] {
	r.payload = payload
	return r
}

// WithRequestExecutor sets the RequestExecutor for the request.
func (r *Request[T]) WithRequestExecutor(re *RequestExecutor) *Request[T] {
	r.re = re
	return r
}

// WithHeaders sets the headers for the request.
func (r *Request[T]) WithHeaders(headers map[string]string) *Request[T] {
	r.headers = headers
	return r
}

// WithQueryParameters sets the query parameters for the request.
func (r *Request[T]) WithQueryParameters(params map[string]string) *Request[T] {
	if len(params) == 0 {
		return r
	}

	queryParams := url.Values{}
	for k, v := range params {
		queryParams.Add(k, v)
	}

	r.queryParameters = queryParams

	return r
}

// Do executes the HTTP request and returns the response.
func (r *Request[T]) Do(ctx context.Context) (*T, error) {
	ok, u, err := isValidURL(r.url)
	if !ok {
		return nil, err
	}

	if r.httpMethod == "GET" {
		q := u.Query()

		for k, v := range r.queryParameters {
			q.Set(k, strings.Join(v, ","))
		}

		u.RawQuery = q.Encode()
	}

	var body []byte
	if r.payload != nil {
		body, err = json.Marshal(r.payload)
		if err != nil {
			return nil, &Error{
				Message: fmt.Sprintf("could not marshal body for request %s. Body:\n %+v", r.url, r.payload),
				Cause:   err,
			}
		}
	}

	buff := bytes.NewBuffer(body)

	if err != nil {
		return nil, &Error{
			Message: fmt.Sprintf("could not create body buffer for request %s. Body:\n %+v", r.url, r.payload),
			Cause:   err,
		}
	}

	req, err := http.NewRequestWithContext(ctx, r.httpMethod, u.String(), buff)
	if err != nil {
		return nil, &Error{
			Message: "could not create request " + r.url,
			Cause:   err,
		}
	}

	for k, v := range r.headers {
		req.Header.Set(k, v)
	}

	res, err := r.re.pipeline(req)
	if err != nil {
		return nil, &Error{
			Message: "failed to make request " + r.url,
			Cause:   err,
		}
	}

	if res == nil {
		return nil, &Error{
			Message: fmt.Sprintf("calling %s returned empty response", u.String()),
		}
	}

	responseData, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, &Error{
			Message: "failed to read response body for url request " + r.url,
			Cause:   err,
		}
	}

	defer res.Body.Close()

	if res.StatusCode >= http.StatusBadRequest {
		return nil, &Error{
			Message:    fmt.Sprintf("error calling %s", u.String()),
			Cause:      fmt.Errorf("%s", responseData),
			StatusCode: res.StatusCode,
		}
	}

	var responseObject T
	contentType := res.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") || contentType == "" {
		err = json.Unmarshal(responseData, &responseObject)

		if err != nil {
			return nil, &Error{
				Message:    "error unmarshaling response for request " + r.url,
				Cause:      err,
				StatusCode: res.StatusCode,
			}
		}
	} else {
		dataAsString := string(responseData)
		var parseErr error
		switch any(responseObject).(type) {
		case string:
			responseObject = any(dataAsString).(T)

		case int:
			data, err := strconv.Atoi(dataAsString)
			responseObject = any(data).(T)
			parseErr = err
		case float64:
			data, err := strconv.ParseFloat(dataAsString, 64)
			responseObject = any(data).(T)
			parseErr = err
		case float32:
			data, err := strconv.ParseFloat(dataAsString, 32)
			responseObject = any(data).(T)
			parseErr = err
		default:
			parseErr = fmt.Errorf("unsupported conversion type: %T", responseObject)
		}

		if parseErr != nil {
			return nil, &Error{
				Message:    "error converting response for request " + r.url,
				Cause:      parseErr,
				StatusCode: res.StatusCode,
			}
		}
	}

	return &responseObject, nil
}

// isValidURL checks if the given URL is valid and parses it.
func isValidURL(u string) (bool, *url.URL, error) {
	parsedURL, err := url.Parse(u)

	if err != nil {
		return false, parsedURL, &Error{
			Message: "could not parse url " + u,
			Cause:   err,
		}
	}

	if parsedURL.Host == "" {
		return false, parsedURL, &Error{
			Message: "invalid url host " + u,
			Cause:   err,
		}
	}

	return true, parsedURL, nil
}
