// Package httpapi is the delivery layer: it parses requests, calls a service and renders the
// result. It contains no business logic, and no SQL.
package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/sovanna2011/farm-area/backend/internal/auth"
	"github.com/sovanna2011/farm-area/backend/internal/domain"
)

type ctxKey int

const (
	ctxUser ctxKey = iota
	ctxRequestID
)

// Recover turns a panic into a 500 with a logged stack, so one bad request cannot take the
// process down and no stack trace is ever sent to a browser.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if p := recover(); p != nil {
					log.Error("panic serving request",
						"path", r.URL.Path, "panic", p, "stack", string(debug.Stack()))
					writeError(w, r, domain.Internal(nil), log)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestLog records one line per request with its outcome and duration.
func RequestLog(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			log.Info("request",
				"method", r.Method, "path", r.URL.Path, "status", rec.status,
				"ms", time.Since(started).Milliseconds(), "user", usernameOf(r))
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// CORS answers the browser's preflight for the SAPUI5 app, which is served from its own origin.
func CORS(allowed []string) func(http.Handler) http.Handler {
	allowedSet := map[string]bool{}
	for _, o := range allowed {
		allowedSet[strings.ToLower(strings.TrimRight(o, "/"))] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.ToLower(strings.TrimRight(r.Header.Get("Origin"), "/"))
			if origin != "" && allowedSet[origin] {
				w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, If-Match")
				// Without this the browser hides the header and every export downloads as the
				// URL's last segment instead of its real file name.
				w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
				w.Header().Set("Access-Control-Max-Age", "600")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Authenticate accepts a bearer token and puts the caller in the context. Every route below /api
// except the login itself passes through here.
func Authenticate(tokens *auth.Tokens, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
				writeError(w, r, domain.Unauthorized("Sign in to use this endpoint."), log)
				return
			}
			user, err := tokens.Parse(strings.TrimSpace(header[len("bearer "):]))
			if err != nil {
				writeError(w, r, domain.Unauthorized("Your session has expired. Sign in again."), log)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxUser, user)))
		})
	}
}

func userOf(r *http.Request) domain.User {
	if u, ok := r.Context().Value(ctxUser).(domain.User); ok {
		return u
	}
	return domain.User{}
}

func usernameOf(r *http.Request) string {
	if u, ok := r.Context().Value(ctxUser).(domain.User); ok {
		return u.Username
	}
	return "-"
}

func remoteOf(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ---------------------------------------------------------------- rendering

type errorBody struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Fields  []domain.FieldError `json:"fields,omitempty"`
}

// writeError is the one place an error becomes a response. A deliberate refusal keeps its code and
// message; anything else becomes a plain 500 and the detail goes to the log, not to the caller.
func writeError(w http.ResponseWriter, r *http.Request, err error, log *slog.Logger) {
	appErr, ok := domain.AsError(err)
	if !ok {
		log.Error("unhandled error", "path", r.URL.Path, "error", err)
		appErr = domain.Internal(err)
	} else if appErr.Status >= 500 {
		log.Error("server error", "path", r.URL.Path, "code", appErr.Code, "error", appErr.Err)
	}

	writeJSON(w, appErr.Status, errorBody{Code: appErr.Code, Message: appErr.Message, Fields: appErr.Fields})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

// ---------------------------------------------------------------- query parsing

func intParam(r *http.Request, name string) (int, error) {
	raw := r.PathValue(name)
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0, domain.BadRequest("INVALID_ID", "The identifier in the path is not a positive number.")
	}
	return id, nil
}

func optionalInt(r *http.Request, name string) *int {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return nil
	}
	return &v
}

func optionalEnum(r *http.Request, name string, allowed []string) (*string, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return nil, nil
	}
	canonical, ok := domain.Canonical(raw, allowed)
	if !ok {
		return nil, domain.BadRequest("INVALID_FILTER",
			name+" must be one of: "+strings.Join(allowed, ", ")+".")
	}
	return &canonical, nil
}

// filterFrom turns the query string into the one filter every read path takes. An unknown value
// for an enumerated filter is refused here rather than passed to Postgres, which would answer
// with a type error the caller cannot act on.
func filterFrom(r *http.Request) (domain.Filter, error) {
	f := domain.Filter{
		CompanyID:    optionalInt(r, "companyId"),
		PlantationID: optionalInt(r, "plantationId"),
		FarmID:       optionalInt(r, "farmId"),
		ZoneID:       optionalInt(r, "zoneId"),
		BlockID:      optionalInt(r, "blockId"),
		CropYear:     optionalInt(r, "cropYear"),
		PlantingYear: optionalInt(r, "plantingYear"),
		SeasonID:     optionalInt(r, "seasonId"),
		VarietyID:    optionalInt(r, "varietyId"),
		Search:       strings.TrimSpace(r.URL.Query().Get("search")),
	}
	f.IncludeInactive, _ = strconv.ParseBool(r.URL.Query().Get("includeInactive"))

	var err error
	if f.PlantingType, err = optionalEnum(r, "plantingType", domain.ValidPlantingTypes); err != nil {
		return f, err
	}
	if f.LandStatus, err = optionalEnum(r, "landStatus", domain.ValidLandStatuses); err != nil {
		return f, err
	}
	if f.CaneStatus, err = optionalEnum(r, "caneStatus", domain.ValidCaneStatuses); err != nil {
		return f, err
	}
	return f, nil
}

func pageFrom(r *http.Request) domain.Page {
	p := domain.Page{
		Number:   1,
		Size:     0,
		SortBy:   strings.TrimSpace(r.URL.Query().Get("sortBy")),
		SortDesc: r.URL.Query().Get("sortDesc") == "true",
	}
	if v := optionalInt(r, "page"); v != nil {
		p.Number = *v
	}
	if v := optionalInt(r, "pageSize"); v != nil {
		p.Size = *v
	}
	return p.Normalise()
}

func decodeBody(r *http.Request, target any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 4<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return domain.BadRequest("INVALID_BODY", "The request body is not the JSON this endpoint expects: "+err.Error())
	}
	return nil
}
