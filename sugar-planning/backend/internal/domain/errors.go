package domain

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors. The HTTP layer maps these onto RFC 9457 problem details so
// that no internal SQL text or stack trace ever reaches a client.
var (
	// ErrValidation marks input that failed syntactic or domain validation.
	ErrValidation = errors.New("validation failed")
	// ErrNotFound marks a missing entity.
	ErrNotFound = errors.New("not found")
	// ErrConflict marks an optimistic concurrency failure (row_version/ETag).
	ErrConflict = errors.New("version conflict")
	// ErrForbidden marks an authorisation failure inside the service layer.
	ErrForbidden = errors.New("forbidden")
	// ErrStateTransition marks an illegal workflow transition.
	ErrStateTransition = errors.New("illegal state transition")
	// ErrLocked marks an attempt to change a released or closed period.
	ErrLocked = errors.New("period locked")
	// ErrCapacity marks a storage capacity violation that was not overridden.
	ErrCapacity = errors.New("capacity exceeded")
	// ErrDuplicate marks a unique business key violation.
	ErrDuplicate = errors.New("duplicate business key")
)

// FieldError carries a message that the SAPUI5 MessagePopover can bind to a
// specific control, plus the source row of a bulk request.
//
// Row is a pointer so that "row 0" and "not about a particular row" are
// different things on the wire: the first row of a bulk upload is a perfectly
// ordinary place for an error to be, and a client must not have to guess.
type FieldError struct {
	Row     *int   `json:"row"`
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ValidationError aggregates field errors for a request or an imported file.
type ValidationError struct {
	Errors []FieldError
}

// Error summarises the field errors.
//
// The API does not use this text - it puts the errors themselves into the
// problem document, addressed by row and field - but a log line, a test failure
// and a wrapped error all do, and "validation failed" on its own tells whoever
// is reading it nothing at all. The first few messages are enough to recognise
// the problem; the rest are counted rather than printed so that a rejected
// import of ten thousand rows does not become a ten-thousand-line log entry.
func (v *ValidationError) Error() string {
	if len(v.Errors) == 0 {
		return "validation failed"
	}
	const shown = 3
	parts := make([]string, 0, shown+1)
	for i, e := range v.Errors {
		if i == shown {
			parts = append(parts, fmt.Sprintf("and %d more", len(v.Errors)-shown))
			break
		}
		if e.Row != nil {
			parts = append(parts, fmt.Sprintf("row %d %s: %s", *e.Row, e.Field, e.Message))
			continue
		}
		parts = append(parts, e.Field+": "+e.Message)
	}
	return "validation failed: " + strings.Join(parts, "; ")
}

// Unwrap lets errors.Is(err, ErrValidation) succeed for aggregated errors.
func (v *ValidationError) Unwrap() error { return ErrValidation }

// Add appends a field error that is not tied to a particular row.
func (v *ValidationError) Add(field, code, msg string) {
	v.Errors = append(v.Errors, FieldError{Field: field, Code: code, Message: msg})
}

// AddRow appends a field error for a specific row of a bulk request.
func (v *ValidationError) AddRow(row int, field, code, msg string) {
	v.Errors = append(v.Errors, FieldError{Row: &row, Field: field, Code: code, Message: msg})
}

// HasErrors reports whether anything was collected.
func (v *ValidationError) HasErrors() bool { return len(v.Errors) > 0 }

// OrNil returns nil when no field errors were collected, so callers can
// `return v.OrNil()` unconditionally.
func (v *ValidationError) OrNil() error {
	if v.HasErrors() {
		return v
	}
	return nil
}
