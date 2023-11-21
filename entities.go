package swiftreq

import (
	"fmt"
)

// Error represents an error that may occur during an HTTP request.
type Error struct {
	Message    string
	Cause      error
	StatusCode int
}

// Error returns a formatted error message including the original cause and status code.
func (e *Error) Error() string {
	return fmt.Sprintf("message: %s\n cause: %s\n statusCode: %d", e.Message, e.Cause.Error(), e.StatusCode)
}

// Response represents the result of an HTTP request.
type Response struct {
	Data       interface{}
	Success    bool
	Error      error
	StatusCode int
}
