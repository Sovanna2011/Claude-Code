package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
)

// Server holds the dependencies of the HTTP layer. Everything is injected
// through the constructor; nothing is reached through a package-level variable.
type Server struct {
	store     store.Store
	planning  *service.Planning
	analytics *service.Analytics
	materials *service.Materials
	verifier  auth.Verifier
	authCfg   auth.Config
	logger    *slog.Logger
	version   string
	// staticDir serves the SAPUI5 application when the API also hosts the UI,
	// which is the on-premises single-container deployment.
	staticDir string
}

// Options configures the server.
type Options struct {
	Store     store.Store
	Planning  *service.Planning
	Analytics *service.Analytics
	Materials *service.Materials
	Verifier  auth.Verifier
	AuthCfg   auth.Config
	Logger    *slog.Logger
	Version   string
	StaticDir string
	// AllowedOrigins is empty for a same-origin deployment.
	AllowedOrigins []string
	RequestTimeout time.Duration
	RateLimit      int
	RateInterval   time.Duration
}

// NewServer builds the HTTP handler.
func NewServer(o Options) http.Handler {
	s := &Server{
		store: o.Store, planning: o.Planning, analytics: o.Analytics, materials: o.Materials,
		verifier: o.Verifier, authCfg: o.AuthCfg, logger: o.Logger,
		version: o.Version, staticDir: o.StaticDir,
	}

	mux := http.NewServeMux()
	s.routes(mux)

	timeout := o.RequestTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	limiter := NewRateLimiter(o.RateLimit, orDefault(o.RateInterval, time.Minute))

	// Only the API needs a token. The SAPUI5 files are served unauthenticated
	// because the browser must be able to load the sign-in page before it has
	// one; the application shows nothing until the session endpoint answers.
	openEndpoints := map[string]bool{
		"/api/v1/auth/dev-login": true,
		"/api/v1/openapi.yaml":   true,
	}
	protected := func(r *http.Request) bool {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			return false
		}
		return !openEndpoints[r.URL.Path]
	}

	var h http.Handler = mux
	h = limiter.Middleware(h)
	h = Authentication(s.verifier, protected)(h)
	h = Timeout(timeout)(h)
	h = SecurityHeaders(h)
	h = CORS(o.AllowedOrigins)(h)
	h = Recovery(h)
	h = Observability(o.Logger)(h)
	return h
}

func orDefault(d, fallback time.Duration) time.Duration {
	if d <= 0 {
		return fallback
	}
	return d
}

// routes registers every endpoint. Go 1.22 method patterns keep the routing
// table readable and make the API surface visible in one place.
func (s *Server) routes(mux *http.ServeMux) {
	// --- operations ---------------------------------------------------------
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /readyz", s.handleReady)
	mux.HandleFunc("GET /api/v1/openapi.yaml", s.handleOpenAPI)

	// --- session ------------------------------------------------------------
	mux.HandleFunc("POST /api/v1/auth/dev-login", s.handleDevLogin)
	mux.HandleFunc("GET /api/v1/session", s.handleSession)

	// --- master data --------------------------------------------------------
	// Every master entity is exposed through the same four routes, generated
	// from one table so a new entity cannot arrive with an inconsistent API.
	md := s.store.MasterData()
	registerMasterData(mux, "companies", md.Companies())
	registerMasterData(mux, "factories", md.Factories())
	registerMasterData(mux, "production-lines", md.Lines())
	registerMasterData(mux, "shifts", md.Shifts())
	registerMasterData(mux, "product-categories", md.ProductCategories())
	registerMasterData(mux, "products", md.Products())
	registerMasterData(mux, "units", md.UOMs())
	registerMasterData(mux, "unit-conversions", md.UOMConversions())
	registerMasterData(mux, "packaging-types", md.PackagingTypes())
	registerMasterData(mux, "warehouses", md.Warehouses())
	registerMasterData(mux, "customers", md.Customers())
	registerMasterData(mux, "shipment-channels", md.Channels())
	registerMasterData(mux, "materials", md.Materials())
	registerMasterData(mux, "reason-codes", md.ReasonCodes())

	// --- seasons and versions ----------------------------------------------
	mux.HandleFunc("GET /api/v1/seasons", s.handleListSeasons)
	mux.HandleFunc("POST /api/v1/seasons", s.handleCreateSeason)
	mux.HandleFunc("GET /api/v1/seasons/{id}", s.handleGetSeason)
	mux.HandleFunc("PUT /api/v1/seasons/{id}", s.handleUpdateSeason)
	mux.HandleFunc("GET /api/v1/seasons/{id}/versions", s.handleListVersions)
	mux.HandleFunc("POST /api/v1/seasons/{id}/versions", s.handleCreateVersion)

	mux.HandleFunc("GET /api/v1/versions/{id}", s.handleGetVersion)
	mux.HandleFunc("PUT /api/v1/versions/{id}", s.handleUpdateVersion)
	mux.HandleFunc("POST /api/v1/versions/{id}/copy", s.handleCopyVersion)
	mux.HandleFunc("POST /api/v1/versions/{id}/generate", s.handleGenerate)
	mux.HandleFunc("POST /api/v1/versions/{id}/transition", s.handleTransition)
	mux.HandleFunc("POST /api/v1/versions/compare", s.handleCompare)

	mux.HandleFunc("PUT /api/v1/versions/{id}/assumptions", s.handleSaveAssumption)
	mux.HandleFunc("PUT /api/v1/versions/{id}/product-mix", s.handleSaveMix)
	mux.HandleFunc("DELETE /api/v1/versions/{id}/product-mix/{mixId}", s.handleDeleteMix)

	// --- daily plan rows ----------------------------------------------------
	mux.HandleFunc("GET /api/v1/versions/{id}/cane", s.handleListCane)
	mux.HandleFunc("POST /api/v1/versions/{id}/cane", s.handleUpsertCane)
	mux.HandleFunc("GET /api/v1/versions/{id}/production", s.handleListProduction)
	mux.HandleFunc("POST /api/v1/versions/{id}/production", s.handleUpsertProduction)
	mux.HandleFunc("GET /api/v1/versions/{id}/storage", s.handleListStorage)
	mux.HandleFunc("POST /api/v1/versions/{id}/storage", s.handleUpsertStorage)
	mux.HandleFunc("GET /api/v1/versions/{id}/shipments", s.handleListShipments)
	mux.HandleFunc("POST /api/v1/versions/{id}/shipments", s.handleUpsertShipments)

	// --- analytics and materials -------------------------------------------
	mux.HandleFunc("GET /api/v1/dashboard", s.handleDashboard)
	mux.HandleFunc("GET /api/v1/versions/{id}/material-requirements", s.handleMaterialRequirements)

	// --- downtime -----------------------------------------------------------
	mux.HandleFunc("GET /api/v1/downtime", s.handleListDowntime)
	mux.HandleFunc("POST /api/v1/downtime", s.handleSaveDowntime)

	// --- reports ------------------------------------------------------------
	mux.HandleFunc("GET /api/v1/reports", s.handleListReports)
	mux.HandleFunc("GET /api/v1/reports/{code}", s.handleRunReport)

	// --- audit --------------------------------------------------------------
	mux.HandleFunc("GET /api/v1/audit", s.handleAudit)

	// --- static SAPUI5 application -----------------------------------------
	if s.staticDir != "" {
		fs := http.FileServer(http.Dir(s.staticDir))
		mux.Handle("GET /", spaHandler{root: s.staticDir, fs: fs})
	}
}

// spaHandler serves the SAPUI5 application, falling back to index.html so that
// a deep link into a client-side route still loads the shell.
type spaHandler struct {
	root string
	fs   http.Handler
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Static assets may be cached; the shell may not, so a redeploy is picked up.
	if strings.Contains(r.URL.Path, ".") {
		w.Header().Set("Cache-Control", "public, max-age=300")
		h.fs.ServeHTTP(w, r)
		return
	}
	http.ServeFile(w, r, h.root+"/index.html")
}

// ---------------------------------------------------------------------------
// Operations endpoints
// ---------------------------------------------------------------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"status": "ok", "version": s.version, "authMode": s.verifier.Mode(),
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		logError(r, err)
		writeJSONStatus(w, http.StatusServiceUnavailable, map[string]any{
			"status": "unavailable", "detail": "the database is not reachable",
		})
		return
	}
	writeJSON(w, map[string]any{"status": "ready", "version": s.version})
}

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(openAPISpec)
}

// ---------------------------------------------------------------------------
// Session
// ---------------------------------------------------------------------------

// handleDevLogin issues a token for one of the configured development
// accounts. It exists only when authentication runs in dev mode; in oidc mode
// it refuses, so an accidental production deployment cannot mint tokens.
func (s *Server) handleDevLogin(w http.ResponseWriter, r *http.Request) {
	if s.verifier.Mode() != "dev" {
		writeProblem(w, r, wrapValidation(
			"development login is disabled; sign in through the identity provider"))
		return
	}
	var body struct {
		Username string `json:"username"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeProblem(w, r, err)
		return
	}
	token, principal, err := auth.IssueDevToken(s.authCfg, body.Username, 8*time.Hour)
	if err != nil {
		writeProblem(w, r, wrapValidation(err.Error()))
		return
	}
	writeJSON(w, map[string]any{
		"accessToken": token,
		"tokenType":   "Bearer",
		"expiresIn":   int((8 * time.Hour).Seconds()),
		"profile":     sessionProfile(principal),
	})
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, sessionProfile(auth.FromContext(r.Context())))
}

func sessionProfile(p auth.Principal) map[string]any {
	return map[string]any{
		"subject":     p.Subject,
		"username":    p.Username,
		"displayName": p.DisplayName,
		"email":       p.Email,
		"roles":       p.Roles,
		"permissions": p.Permissions(),
		"companies":   p.Companies,
		"factories":   p.Factories,
	}
}

// ---------------------------------------------------------------------------
// Query helpers
// ---------------------------------------------------------------------------

// listOptions reads the paging and filter parameters that every list endpoint
// accepts. Unknown parameters are ignored rather than trusted.
func listOptions(r *http.Request) store.ListOptions {
	q := r.URL.Query()
	opts := store.ListOptions{
		Skip:     atoiOr(q.Get("$skip"), 0),
		Top:      atoiOr(q.Get("$top"), 100),
		Search:   q.Get("$search"),
		ParentID: q.Get("parentId"),
		OrderBy:  q.Get("$orderby"),
	}
	switch strings.ToLower(q.Get("active")) {
	case "true":
		t := true
		opts.Active = &t
	case "false":
		f := false
		opts.Active = &f
	}
	return opts.Normalise()
}

// planFilter reads the allow-listed filters for daily plan rows.
func planFilter(r *http.Request, versionID string) store.PlanFilter {
	q := r.URL.Query()
	f := store.PlanFilter{
		VersionIDs:   []string{versionID},
		FactoryID:    q.Get("factoryId"),
		From:         domain.BusinessDate(q.Get("from")),
		To:           domain.BusinessDate(q.Get("to")),
		ProductIDs:   splitCSV(q.Get("productId")),
		WarehouseIDs: splitCSV(q.Get("warehouseId")),
		ChannelIDs:   splitCSV(q.Get("channelId")),
		LineIDs:      splitCSV(q.Get("lineId")),
		Skip:         atoiOr(q.Get("$skip"), 0),
		Top:          atoiOr(q.Get("$top"), 0),
	}
	if series := strings.ToUpper(q.Get("series")); series != "" {
		f.Series = domain.Series(series)
	}
	return f
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func atoiOr(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return v
}

func wrapValidation(msg string) error {
	return &domain.ValidationError{Errors: []domain.FieldError{
		{Field: "request", Code: "INVALID", Message: msg},
	}}
}
