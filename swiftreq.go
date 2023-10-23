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

type Error struct {
	Message    string
	Cause      error
	StatusCode int
}

func (e *Error) Error() string {
	return fmt.Sprintf("message: %s\n cause: %s\n statusCode: %d", e.Message, e.Cause.Error(), e.StatusCode)
}

type EmptyRequest struct{}

type Response struct {
	Data       interface{}
	Success    bool
	Error      error
	StatusCode int
}

type RequestExecutor struct {
	client http.Client
}

func NewRequestExecutor(client http.Client) *RequestExecutor {
	return &RequestExecutor{
		client: client,
	}
}

type Request[TReq, TResp any] struct {
	re *RequestExecutor
}

func NewRequest[TReq, TResp any](re *RequestExecutor) *Request[TReq, TResp] {
	return &Request[TReq, TResp]{
		re: re,
	}
}

func (r *Request[TReq, TResp]) Get(
	ctx context.Context,
	url string,
	headers map[string]string,
	queryParameters url.Values,
	request *TReq) (*TResp, error) {

	return r.makeHTTPRequest(ctx, url, "GET", headers, queryParameters, request)
}

func (r *Request[TReq, TResp]) Post(
	ctx context.Context,
	url string,
	headers map[string]string,
	queryParameters url.Values,
	request *TReq) (*TResp, error) {

	return r.makeHTTPRequest(ctx, url, "POST", headers, queryParameters, request)
}

func (r *Request[TReq, TResp]) Put(
	ctx context.Context,
	url string,
	headers map[string]string,
	queryParameters url.Values,
	request *TReq) (*TResp, error) {

	return r.makeHTTPRequest(ctx, url, "PUT", headers, queryParameters, request)
}

func (r *Request[TReq, TResp]) Delete(
	ctx context.Context,
	url string,
	headers map[string]string,
	queryParameters url.Values,
	request *TReq) (*TResp, error) {

	return r.makeHTTPRequest(ctx, url, "DELETE", headers, queryParameters, request)
}

func (r *Request[TReq, TResp]) makeHTTPRequest(
	ctx context.Context,
	fullUrl string,
	httpMethod string,
	headers map[string]string,
	queryParameters url.Values,
	request *TReq) (*TResp, error) {

	ok, u, err := isValidURL(fullUrl)
	if !ok {
		return nil, err
	}

	// if it's a GET, we need to append the query parameters.
	if httpMethod == "GET" {
		q := u.Query()

		for k, v := range queryParameters {
			// this depends on the type of api, you may need to do it for each of v
			q.Set(k, strings.Join(v, ","))
		}
		// set the query to the encoded parameters
		u.RawQuery = q.Encode()
	}

	// regardless of GET or POST, we can safely add the body
	body, err := json.Marshal(request)
	if err != nil {
		return nil, &Error{
			Message: fmt.Sprintf("could not marshal body for request %s. Body:\n %+v", fullUrl, request),
			Cause:   err,
		}
	}

	buff := bytes.NewBuffer(body)

	if err != nil {
		return nil, &Error{
			Message: fmt.Sprintf("could not create body buffer for request %s. Body:\n %+v", fullUrl, request),
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
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// optional: log the request for easier stack tracing
	log.Printf("%s %s\n", httpMethod, req.URL.String())

	// finally, do the request
	res, err := r.re.client.Do(req)
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

	var responseObject TResp
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
