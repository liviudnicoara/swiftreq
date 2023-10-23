package swiftreq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Response struct {
	Data       interface{}
	Success    bool
	Error      error
	StatusCode int
}

type ReqExec struct {
}

func (r *ReqExec) Get(url string) (*Response, error) {
	if !isValidURL(url) {
		return nil, fmt.Errorf("invalid URL: %s", url)
	}

	resp, err := http.Get(url)

	if err != nil {
		return nil, fmt.Errorf("GET request failed: %v", url)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)

	if err != nil {
		return nil, fmt.Errorf("error decoding response body: %v", err)
	}

	if resp.StatusCode >= 400 {
		return &Response{
			Success:    false,
			Error:      fmt.Errorf("error received: %v", result),
			StatusCode: resp.StatusCode,
		}, nil
	}

	return &Response{
		Success:    true,
		Data:       result,
		StatusCode: resp.StatusCode,
	}, nil
}

func (r *ReqExec) Post(url string, data interface{}) (*Response, error) {
	if !isValidURL(url) {
		return nil, fmt.Errorf("invalid URL: %s", url)

	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON data: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("POST request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)

	if err != nil {
		return nil, fmt.Errorf("error decoding response body: %v", err)
	}

	if resp.StatusCode >= 400 {
		return &Response{
			Success:    false,
			Error:      fmt.Errorf("error received: %v", result),
			StatusCode: resp.StatusCode,
		}, nil
	}

	return &Response{
		Success:    true,
		Data:       result,
		StatusCode: resp.StatusCode,
	}, nil
}

func (r *ReqExec) Put(url string, data interface{}) (*Response, error) {
	if !isValidURL(url) {
		return nil, fmt.Errorf("invalid URL: %s", url)
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON data: %v", err)
	}

	request, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating PUT request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("PUT request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)

	if err != nil {
		return nil, fmt.Errorf("error decoding response body: %v", err)
	}

	if resp.StatusCode >= 400 {
		return &Response{
			Success:    false,
			Error:      fmt.Errorf("error received: %v", result),
			StatusCode: resp.StatusCode,
		}, nil
	}

	return &Response{
		Success:    true,
		Data:       result,
		StatusCode: resp.StatusCode,
	}, nil
}

func (r *ReqExec) Delete(url string) (*Response, error) {
	if !isValidURL(url) {
		return nil, fmt.Errorf("invalid URL: %s", url)
	}

	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating DELETE request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("DELETE request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)

	if err != nil {
		return nil, fmt.Errorf("error decoding response body: %v", err)
	}

	if resp.StatusCode >= 400 {
		return &Response{
			Success:    false,
			Error:      fmt.Errorf("error received: %v", result),
			StatusCode: resp.StatusCode,
		}, nil
	}

	return &Response{
		Success:    true,
		Data:       result,
		StatusCode: resp.StatusCode,
	}, nil
}

func isValidURL(u string) bool {
	parsedURL, err := url.Parse(u)
	return err == nil && parsedURL.Scheme != "" && parsedURL.Host != ""
}
