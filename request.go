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
	"strconv"
	"strings"
)

type Request[T any] struct {
	re              *RequestExecutor
	headers         map[string]string
	httpMethod      string
	url             string
	payload         interface{}
	queryParameters url.Values
}

func NewGetRequest[T any](url string) *Request[T] {
	return NewDefaultRequest[T]().
		WithMethod("GET").
		WithURL(url)
}

func NewPostRequest[T any](url string, payload interface{}) *Request[T] {
	return NewDefaultRequest[T]().
		WithMethod("POST").
		WithURL(url).
		WithPayload(payload)
}
func NewPutRequest[T any](url string, payload interface{}) *Request[T] {
	return NewDefaultRequest[T]().
		WithMethod("PUT").
		WithURL(url).
		WithPayload(payload)
}

func NewDeleteRequest[T any](url string) *Request[T] {
	return NewDefaultRequest[T]().
		WithMethod("DELETE").
		WithURL(url)
}

func NewDefaultRequest[T any]() *Request[T] {
	return NewRequest[T](Default())
}

func NewRequest[T any](re *RequestExecutor) *Request[T] {
	return &Request[T]{
		re: re,
		headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

func (r *Request[T]) WithMethod(httpMethod string) *Request[T] {
	r.httpMethod = httpMethod
	return r
}

func (r *Request[T]) WithURL(url string) *Request[T] {
	r.url = url
	return r
}

func (r *Request[T]) WithPayload(payload interface{}) *Request[T] {
	r.payload = payload
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

func (r *Request[T]) Do(ctx context.Context) (*T, error) {
	ok, u, err := isValidURL(r.url)
	if !ok {
		return nil, err
	}

	// if it's a GET, we need to append the query parameters.
	if r.httpMethod == "GET" {
		q := u.Query()

		for k, v := range r.queryParameters {
			// this depends on the type of api, you may need to do it for each of v
			q.Set(k, strings.Join(v, ","))
		}
		// set the query to the encoded parameters
		u.RawQuery = q.Encode()
	}

	// regardless of GET or POST, we can safely add the body
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

	// for each header passed, add the header value to the request
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}

	// optional: log the request for easier stack tracing
	log.Printf("%s %s\n", r.httpMethod, req.URL.String())

	// finally, do the request
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
	if res.Header.Get("Content-Type") == "application/json" {
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
			parseErr = fmt.Errorf("unssuported  conversion type: %T", responseObject)
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
