package domain

import (
	"errors"
	"fmt"
	"net/http"
)

// FieldError names the input that was wrong and says why, so the UI can put the message beside the
// field rather than in a banner at the top of the form.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error is the one error type the HTTP layer knows how to render. Everything the services reject
// deliberately is one of these; anything else that reaches the handler is a bug and becomes a 500
// with the detail kept in the log rather than sent to the browser.
type Error struct {
	Status  int
	Code    string
	Message string
	Fields  []FieldError
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

func NotFound(entity string, id any) *Error {
	return &Error{
		Status:  http.StatusNotFound,
		Code:    "NOT_FOUND",
		Message: fmt.Sprintf("%s %v was not found.", entity, id),
	}
}

func BadRequest(code, message string) *Error {
	return &Error{Status: http.StatusBadRequest, Code: code, Message: message}
}

func Invalid(fields []FieldError) *Error {
	return &Error{
		Status:  http.StatusUnprocessableEntity,
		Code:    "VALIDATION_FAILED",
		Message: "The area figures are not consistent.",
		Fields:  fields,
	}
}

func Conflict(code, message string) *Error {
	return &Error{Status: http.StatusConflict, Code: code, Message: message}
}

// StaleVersion is what an optimistic-concurrency loss looks like: the row moved under the caller
// between reading it and saving it.
func StaleVersion(entity string, id any) *Error {
	return &Error{
		Status: http.StatusConflict,
		Code:   "CONCURRENCY_CONFLICT",
		Message: fmt.Sprintf(
			"%s %v was changed by someone else after you loaded it. Reload and apply your change again.", entity, id),
	}
}

func Unauthorized(message string) *Error {
	return &Error{Status: http.StatusUnauthorized, Code: "UNAUTHORIZED", Message: message}
}

func Forbidden(message string) *Error {
	return &Error{Status: http.StatusForbidden, Code: "FORBIDDEN", Message: message}
}

func Internal(err error) *Error {
	return &Error{
		Status:  http.StatusInternalServerError,
		Code:    "INTERNAL_ERROR",
		Message: "The request could not be completed.",
		Err:     err,
	}
}

// AsError extracts a *Error from a chain, so a handler can tell a deliberate refusal from a bug.
func AsError(err error) (*Error, bool) {
	var target *Error
	ok := errors.As(err, &target)
	return target, ok
}
