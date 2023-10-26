package swiftreq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type Request[T any] struct {
	re              *RequestExecutor
	headers         map[string]string
	httpMethod      string
	queryParameters url.Values
}

func NewGetRequest[T any]() *Request[T] {
	return NewDefaultRequest[T]().WithMethod("GET")
}

func NewPostRequest[T any]() *Request[T] {
	return NewDefaultRequest[T]().WithMethod("POST")
}
func NewPutRequest[T any]() *Request[T] {
	return NewDefaultRequest[T]().WithMethod("PUT")
}

func NewDeleteRequest[T any]() *Request[T] {
	return NewDefaultRequest[T]().WithMethod("DELETE")
}

func NewDefaultRequest[T any]() *Request[T] {
	return NewRequest[T](DefaultRequestExecutor)
}

func NewRequest[T any](re *RequestExecutor) *Request[T] {
	return &Request[T]{
		re: re,
	}
}

func (r *Request[T]) WithMethod(httpMethod string) *Request[T] {
	r.httpMethod = httpMethod
	return r
}

func (r *Request[T]) WithRequestExecutor(re *RequestExecutor) *Request[T] {
	r.re = re
	return r
}

func (r *Request[T]) WithHeaders(headers map[string]string) *Request[T] {
	r.headers = headers
	return r
}

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

func (r *Request[T]) Get(ctx context.Context, url string) (*T, error) {
	return r.makeHTTPRequest(ctx, url, "GET", nil)
}

func (r *Request[T]) Post(ctx context.Context, url string, payload interface{}) (*T, error) {
	return r.makeHTTPRequest(ctx, url, "POST", payload)
}

func (r *Request[T]) Put(ctx context.Context, url string, payload interface{}) (*T, error) {
	return r.makeHTTPRequest(ctx, url, "PUT", payload)
}

func (r *Request[T]) Delete(ctx context.Context, url string, payload interface{}) (*T, error) {
	return r.makeHTTPRequest(ctx, url, "DELETE", payload)
}

func (r *Request[T]) makeHTTPRequest(ctx context.Context, fullUrl string, httpMethod string, payload interface{}) (*T, error) {
	ok, u, err := isValidURL(fullUrl)
	if !ok {
		return nil, err
	}

	// if it's a GET, we need to append the query parameters.
	if httpMethod == "GET" {
		q := u.Query()

		for k, v := range r.queryParameters {
			// this depends on the type of api, you may need to do it for each of v
			q.Set(k, strings.Join(v, ","))
		}
		// set the query to the encoded parameters
		u.RawQuery = q.Encode()
	}

	// regardless of GET or POST, we can safely add the body
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, &Error{
			Message: fmt.Sprintf("could not marshal body for request %s. Body:\n %+v", fullUrl, payload),
			Cause:   err,
		}
	}

	buff := bytes.NewBuffer(body)

	if err != nil {
		return nil, &Error{
			Message: fmt.Sprintf("could not create body buffer for request %s. Body:\n %+v", fullUrl, payload),
			Cause:   err,
		}
	}

	req, err := http.NewRequestWithContext(ctx, httpMethod, u.String(), buff)
	if err != nil {
		return nil, &Error{
			Message: "could not create request " + fullUrl,
			Cause:   err,
		}
	}

	// for each header passed, add the header value to the request
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}

	// optional: log the request for easier stack tracing
	log.Printf("%s %s\n", httpMethod, req.URL.String())

	// finally, do the request
	res, err := r.re.pipeline(req)
	// res, err := r.re.client.Do(req)
	if err != nil {
		return nil, &Error{
			Message: "failed to make request " + fullUrl,
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
			Message: "failed to read response body for url request " + fullUrl,
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
	err = json.Unmarshal(responseData, &responseObject)

	if err != nil {
		return nil, &Error{
			Message:    "error unmarshaling response for request " + fullUrl,
			Cause:      err,
			StatusCode: res.StatusCode,
		}
	}

	return &responseObject, nil
}

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
