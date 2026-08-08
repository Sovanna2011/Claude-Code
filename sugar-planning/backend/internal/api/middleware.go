package api

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/service"
)

type loggerKey struct{}

func logError(r *http.Request, err error) {
	if l, ok := r.Context().Value(loggerKey{}).(*slog.Logger); ok {
		l.Error("request failed", "error", err)
		return
	}
	slog.Error("request failed", "error", err)
}

// statusRecorder captures the response status for the access log.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

// Observability attaches a correlation id and a request-scoped logger, and
// writes one structured access log line per request.
func Observability(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			correlation := r.Header.Get("X-Correlation-Id")
			if correlation == "" {
				correlation = uuid.NewString()
			}
			w.Header().Set("X-Correlation-Id", correlation)

			ip := clientIP(r)
			logger := base.With(
				"correlationId", correlation,
				"method", r.Method,
				"path", r.URL.Path,
			)
			ctx := context.WithValue(r.Context(), loggerKey{}, logger)
			ctx = service.WithCorrelation(ctx, correlation)
			ctx = service.WithSourceIP(ctx, ip)

			rec := &statusRecorder{ResponseWriter: w}
			started := time.Now()
			next.ServeHTTP(rec, r.WithContext(ctx))

			logger.Info("request",
				"status", rec.status,
				"bytes", rec.bytes,
				"durationMs", time.Since(started).Milliseconds(),
				"user", auth.FromContext(r.Context()).Username,
			)
		})
	}
}

// Recovery turns a panic into a 500 problem document instead of a dropped
// connection, and logs the stack once.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if l, ok := r.Context().Value(loggerKey{}).(*slog.Logger); ok {
					l.Error("panic recovered", "panic", rec, "stack", string(debug.Stack()))
				}
				writeJSONStatus(w, http.StatusInternalServerError, Problem{
					Type: problemTypeBase + "internal", Title: "The request could not be completed",
					Status: http.StatusInternalServerError, Instance: r.URL.Path,
					Detail:        "An unexpected error occurred. Quote the correlation id when reporting it.",
					CorrelationID: service.CorrelationFromContext(r.Context()),
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Timeout bounds how long a request may run, so a slow query cannot pin a
// connection indefinitely.
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Authentication verifies the bearer token and puts the principal in the
// context. `protected` decides which requests need a token: the API does, the
// SAPUI5 application's own files do not, because the browser has to be able to
// load the sign-in page before it has a token to present.
func Authentication(v auth.Verifier, protected func(*http.Request) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !protected(r) {
				next.ServeHTTP(w, r)
				return
			}
			header := r.Header.Get("Authorization")
			token, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || strings.TrimSpace(token) == "" {
				unauthorised(w, r, "A bearer token is required.")
				return
			}
			principal, err := v.Verify(r.Context(), strings.TrimSpace(token))
			if err != nil {
				// The reason is logged, not returned: an attacker learns nothing
				// from the difference between an expired and a forged token.
				logError(r, err)
				unauthorised(w, r, "The access token was not accepted.")
				return
			}
			next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), principal)))
		})
	}
}

func unauthorised(w http.ResponseWriter, r *http.Request, detail string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="sugarplan"`)
	writeJSONStatus(w, http.StatusUnauthorized, Problem{
		Type: problemTypeBase + "unauthenticated", Title: "Authentication required",
		Status: http.StatusUnauthorized, Detail: detail, Instance: r.URL.Path,
		CorrelationID: service.CorrelationFromContext(r.Context()),
	})
}

// SecurityHeaders sets the headers that cost nothing and close off whole
// classes of problem.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// CORS allows the configured origins. The API is normally served from the same
// origin as the SAPUI5 application, so the default list is empty and no
// cross-origin request is permitted.
func CORS(allowed []string) func(http.Handler) http.Handler {
	allow := map[string]bool{}
	for _, o := range allowed {
		allow[strings.TrimSpace(o)] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && allow[origin] {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Vary", "Origin")
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers",
					"Authorization, Content-Type, If-Match, X-Correlation-Id, Idempotency-Key")
				h.Set("Access-Control-Expose-Headers", "ETag, X-Correlation-Id")
				h.Set("Access-Control-Max-Age", "600")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiter is a fixed-window limiter keyed on the caller, which is enough to
// stop a runaway client or a naive scraper without a shared cache.
type RateLimiter struct {
	mu       sync.Mutex
	counts   map[string]int
	window   time.Time
	limit    int
	interval time.Duration
}

// NewRateLimiter builds a limiter allowing `limit` requests per interval.
func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	return &RateLimiter{counts: map[string]int{}, window: time.Now(), limit: limit, interval: interval}
}

// Middleware applies the limit.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rl.limit <= 0 {
			next.ServeHTTP(w, r)
			return
		}
		key := auth.FromContext(r.Context()).Subject
		if key == "" {
			key = clientIP(r)
		}

		rl.mu.Lock()
		if time.Since(rl.window) > rl.interval {
			rl.counts = map[string]int{}
			rl.window = time.Now()
		}
		rl.counts[key]++
		count := rl.counts[key]
		rl.mu.Unlock()

		if count > rl.limit {
			w.Header().Set("Retry-After", strconv.Itoa(max(1, int(rl.interval.Seconds()))))
			writeJSONStatus(w, http.StatusTooManyRequests, Problem{
				Type: problemTypeBase + "rate-limit", Title: "Too many requests",
				Status: http.StatusTooManyRequests, Instance: r.URL.Path,
				Detail:        "The request rate limit was exceeded. Try again shortly.",
				CorrelationID: service.CorrelationFromContext(r.Context()),
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP prefers the proxy header when the deployment sits behind one, and
// falls back to the socket address.
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if first, _, ok := strings.Cut(fwd, ","); ok {
			return strings.TrimSpace(first)
		}
		return strings.TrimSpace(fwd)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// requirePermission is a small guard used by handlers whose whole purpose is
// gated on a single permission.
func requirePermission(r *http.Request, permission string) error {
	return auth.FromContext(r.Context()).Require(permission)
}
