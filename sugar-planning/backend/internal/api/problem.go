// Package api is the HTTP transport. It translates requests into service
// calls, maps domain errors onto RFC 9457 problem details and never lets a
// database message or a stack trace reach a client.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
)

// Problem is an RFC 9457 problem details document, extended with the field
// errors that the SAPUI5 MessagePopover binds to.
type Problem struct {
	Type          string              `json:"type"`
	Title         string              `json:"title"`
	Status        int                 `json:"status"`
	Detail        string              `json:"detail,omitempty"`
	Instance      string              `json:"instance,omitempty"`
	CorrelationID string              `json:"correlationId,omitempty"`
	Errors        []domain.FieldError `json:"errors,omitempty"`
}

// problemTypeBase namespaces the problem type URIs.
const problemTypeBase = "https://sugarplan.example.com/problems/"

// writeProblem maps an error onto the right status code and body.
//
// The mapping is deliberately total: any error that is not a recognised domain
// error becomes a 500 with a generic message, and the real text goes to the
// log rather than to the client.
func writeProblem(w http.ResponseWriter, r *http.Request, err error) {
	p := Problem{
		Instance:      r.URL.Path,
		CorrelationID: service.CorrelationFromContext(r.Context()),
	}

	var verr *domain.ValidationError
	switch {
	case errors.As(err, &verr):
		p.Status, p.Type, p.Title = http.StatusBadRequest, problemTypeBase+"validation", "The request was not valid"
		p.Detail = "One or more fields could not be accepted."
		p.Errors = verr.Errors

	case errors.Is(err, domain.ErrValidation):
		p.Status, p.Type, p.Title = http.StatusBadRequest, problemTypeBase+"validation", "The request was not valid"
		p.Detail = cleanMessage(err)

	case errors.Is(err, domain.ErrNotFound):
		p.Status, p.Type, p.Title = http.StatusNotFound, problemTypeBase+"not-found", "Not found"
		p.Detail = cleanMessage(err)

	case errors.Is(err, domain.ErrForbidden):
		p.Status, p.Type, p.Title = http.StatusForbidden, problemTypeBase+"forbidden", "Not permitted"
		p.Detail = cleanMessage(err)

	case errors.Is(err, domain.ErrConflict):
		// 412 rather than 409: the caller sent a stale If-Match/rowVersion.
		p.Status, p.Type, p.Title = http.StatusPreconditionFailed,
			problemTypeBase+"version-conflict", "The record was changed by somebody else"
		p.Detail = cleanMessage(err)

	case errors.Is(err, domain.ErrDuplicate):
		p.Status, p.Type, p.Title = http.StatusConflict, problemTypeBase+"duplicate", "Already exists"
		p.Detail = cleanMessage(err)

	case errors.Is(err, domain.ErrLocked):
		p.Status, p.Type, p.Title = http.StatusConflict, problemTypeBase+"locked", "The period is locked"
		p.Detail = cleanMessage(err)

	case errors.Is(err, domain.ErrStateTransition):
		p.Status, p.Type, p.Title = http.StatusConflict,
			problemTypeBase+"state-transition", "That action is not available in this status"
		p.Detail = cleanMessage(err)

	case errors.Is(err, domain.ErrCapacity):
		p.Status, p.Type, p.Title = http.StatusConflict, problemTypeBase+"capacity", "Storage capacity exceeded"
		p.Detail = cleanMessage(err)

	default:
		p.Status, p.Type, p.Title = http.StatusInternalServerError,
			problemTypeBase+"internal", "The request could not be completed"
		p.Detail = "An unexpected error occurred. Quote the correlation id when reporting it."
		logError(r, err)
	}

	// RFC 9457 gives problem documents their own media type, and clients are
	// entitled to branch on it: a generic error handler needs to know it has a
	// problem document without first parsing the body to find out.
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(p.Status)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		// The status line is already sent; all that is left is to record it.
		logError(r, err)
	}
}

// cleanMessage strips the sentinel prefix so the client sees the explanation
// rather than "validation failed: ...".
func cleanMessage(err error) string {
	msg := err.Error()
	for _, prefix := range []string{
		"validation failed: ", "not found: ", "forbidden: ", "version conflict: ",
		"duplicate business key: ", "period locked: ", "illegal state transition: ",
		"capacity exceeded: ",
	} {
		msg = strings.TrimPrefix(msg, prefix)
	}
	return capitalise(msg)
}

func capitalise(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ---------------------------------------------------------------------------
// Response helpers
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, v any) { writeJSONStatus(w, http.StatusOK, v) }

func writeJSONStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// The status line is already sent; all that is left is to record it.
		_ = err
	}
}

// decodeJSON reads a request body with a size limit, rejecting unknown fields
// so that a typo in a payload is reported rather than silently ignored.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	const maxBody = 32 << 20 // 32 MB, enough for a full season bulk upsert
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("%w: the request body could not be read: %s", domain.ErrValidation, jsonErrorText(err))
	}
	return nil
}

func jsonErrorText(err error) string {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return fmt.Sprintf("field %q expects a %s", typeErr.Field, typeErr.Type)
	}
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return fmt.Sprintf("malformed JSON at byte %d", syntaxErr.Offset)
	}
	return err.Error()
}

// ---------------------------------------------------------------------------
// Paging and ETag
// ---------------------------------------------------------------------------

// pageResponse is the uniform envelope for list endpoints.
type pageResponse[T any] struct {
	Value []T `json:"value"`
	Count int `json:"count"`
	Skip  int `json:"skip"`
	Top   int `json:"top"`
}

// etagFor builds the entity tag from an optimistic concurrency row version.
func etagFor(rowVersion int64) string { return fmt.Sprintf(`"%d"`, rowVersion) }

// ifMatch reads the If-Match header and returns the row version it carries.
// A missing header returns 0, which the services treat as "no check
// requested"; endpoints that require it say so explicitly.
func ifMatch(r *http.Request) (int64, error) {
	raw := strings.TrimSpace(r.Header.Get("If-Match"))
	if raw == "" || raw == "*" {
		return 0, nil
	}
	raw = strings.Trim(raw, `"`)
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: the If-Match header must carry a row version", domain.ErrValidation)
	}
	return v, nil
}

// requireIfMatch is used by endpoints where a lost update would be damaging.
func requireIfMatch(r *http.Request) (int64, error) {
	v, err := ifMatch(r)
	if err != nil {
		return 0, err
	}
	if v == 0 {
		return 0, fmt.Errorf(
			"%w: this endpoint needs an If-Match header carrying the row version you read", domain.ErrValidation)
	}
	return v, nil
}
