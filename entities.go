package swiftreq

import (
	"fmt"
)

type Error struct {
	Message    string
	Cause      error
	StatusCode int
}

func (e *Error) Error() string {
	return fmt.Sprintf("message: %s\n cause: %s\n statusCode: %d", e.Message, e.Cause.Error(), e.StatusCode)
}

type Response struct {
	Data       interface{}
	Success    bool
	Error      error
	StatusCode int
}
