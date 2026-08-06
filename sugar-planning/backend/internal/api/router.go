package api

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
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
	execution *service.Execution
	verifier  auth.Verifier
	authCfg   auth.Config
	logger    *slog.Logger
	version   string
	// now is injected so that a default business date in a request is
	// deterministic in tests.
	now func() time.Time
	// staticDir serves the SAPUI5 application when the API also hosts the UI,
	// which is the on-premises single-container deployment.
	staticDir string
	// patterns is every route registered, in registration order.
	patterns []string
}

// Options configures the server.
type Options struct {
	Store     store.Store
	Planning  *service.Planning
	Analytics *service.Analytics
	Materials *service.Materials
	Execution *service.Execution
	Verifier  auth.Verifier
	AuthCfg   auth.Config
	Logger    *slog.Logger
	Version   string
	StaticDir string
	// Now overrides the clock. Leave it nil outside tests.
	Now func() time.Time
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
		execution: o.Execution, verifier: o.Verifier, authCfg: o.AuthCfg, logger: o.Logger,
		version: o.Version, staticDir: o.StaticDir, now: o.Now,
	}
	if s.now == nil {
		s.now = func() time.Time { return time.Now().UTC() }
	}
	if s.execution == nil {
		s.execution = service.NewExecution(o.Store, s.now)
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
		"/api/v1/auth/dev-users": true,
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

// handle registers one endpoint and remembers its pattern, so that a test can
// hold the OpenAPI document to the routing table rather than to a list somebody
// has to keep up to date by hand.
func (s *Server) handle(mux *http.ServeMux, pattern string, h http.HandlerFunc) {
	s.patterns = append(s.patterns, pattern)
	mux.HandleFunc(pattern, h)
}

// routes registers every endpoint. Go 1.22 method patterns keep the routing
// table readable and make the API surface visible in one place.
func (s *Server) routes(mux *http.ServeMux) {
	// --- operations ---------------------------------------------------------
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /readyz", s.handleReady)
	s.handle(mux, "GET /api/v1/openapi.yaml", s.handleOpenAPI)

	// --- session ------------------------------------------------------------
	s.handle(mux, "GET /api/v1/auth/dev-users", s.handleDevUsers)
	s.handle(mux, "POST /api/v1/auth/dev-login", s.handleDevLogin)
	s.handle(mux, "GET /api/v1/session", s.handleSession)

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
	s.handle(mux, "GET /api/v1/seasons", s.handleListSeasons)
	s.handle(mux, "POST /api/v1/seasons", s.handleCreateSeason)
	s.handle(mux, "GET /api/v1/seasons/{id}", s.handleGetSeason)
	s.handle(mux, "PUT /api/v1/seasons/{id}", s.handleUpdateSeason)
	s.handle(mux, "GET /api/v1/seasons/{id}/versions", s.handleListVersions)
	s.handle(mux, "POST /api/v1/seasons/{id}/versions", s.handleCreateVersion)

	s.handle(mux, "GET /api/v1/versions/{id}", s.handleGetVersion)
	s.handle(mux, "PUT /api/v1/versions/{id}", s.handleUpdateVersion)
	s.handle(mux, "POST /api/v1/versions/{id}/copy", s.handleCopyVersion)
	s.handle(mux, "POST /api/v1/versions/{id}/generate", s.handleGenerate)
	s.handle(mux, "POST /api/v1/versions/{id}/transition", s.handleTransition)
	s.handle(mux, "POST /api/v1/versions/compare", s.handleCompare)

	s.handle(mux, "PUT /api/v1/versions/{id}/assumptions", s.handleSaveAssumption)
	s.handle(mux, "PUT /api/v1/versions/{id}/product-mix", s.handleSaveMix)
	s.handle(mux, "DELETE /api/v1/versions/{id}/product-mix/{mixId}", s.handleDeleteMix)

	// --- daily plan rows ----------------------------------------------------
	s.handle(mux, "GET /api/v1/versions/{id}/cane", s.handleListCane)
	s.handle(mux, "POST /api/v1/versions/{id}/cane", s.handleUpsertCane)
	s.handle(mux, "GET /api/v1/versions/{id}/production", s.handleListProduction)
	s.handle(mux, "POST /api/v1/versions/{id}/production", s.handleUpsertProduction)
	s.handle(mux, "GET /api/v1/versions/{id}/storage", s.handleListStorage)
	s.handle(mux, "POST /api/v1/versions/{id}/storage", s.handleUpsertStorage)
	s.handle(mux, "GET /api/v1/versions/{id}/shipments", s.handleListShipments)
	s.handle(mux, "POST /api/v1/versions/{id}/shipments", s.handleUpsertShipments)

	// --- analytics and materials -------------------------------------------
	s.handle(mux, "GET /api/v1/dashboard", s.handleDashboard)
	s.handle(mux, "GET /api/v1/versions/{id}/material-requirements", s.handleMaterialRequirements)

	// --- downtime and maintenance -------------------------------------------
	s.handle(mux, "GET /api/v1/downtime", s.handleListDowntime)
	s.handle(mux, "POST /api/v1/downtime", s.handleSaveDowntime)
	s.handle(mux, "GET /api/v1/maintenance", s.handleListMaintenance)
	s.handle(mux, "PUT /api/v1/maintenance", s.handleSaveMaintenance)

	// --- stock and inventory postings ---------------------------------------
	s.handle(mux, "GET /api/v1/stock", s.handleStock)
	s.handle(mux, "GET /api/v1/inventory/documents", s.handleListDocuments)
	s.handle(mux, "POST /api/v1/inventory/documents", s.handlePostDocument)
	s.handle(mux, "GET /api/v1/inventory/documents/{id}", s.handleGetDocument)
	s.handle(mux, "POST /api/v1/inventory/documents/{id}/reverse", s.handleReverseDocument)

	// --- production orders --------------------------------------------------
	s.handle(mux, "GET /api/v1/production-orders", s.handleListOrders)
	s.handle(mux, "POST /api/v1/production-orders", s.handleCreateOrder)
	s.handle(mux, "GET /api/v1/production-orders/{id}", s.handleGetOrder)
	s.handle(mux, "POST /api/v1/production-orders/{id}/action", s.handleOrderAction)
	s.handle(mux, "POST /api/v1/production-orders/{id}/confirm", s.handleConfirmOrder)
	s.handle(mux, "POST /api/v1/versions/{id}/production-orders", s.handleOrdersFromPlan)
	s.handle(mux, "POST /api/v1/confirmations/{id}/reverse", s.handleReverseConfirmation)

	// --- quality ------------------------------------------------------------
	s.handle(mux, "GET /api/v1/quality/parameters", s.handleListQualityParameters)
	s.handle(mux, "PUT /api/v1/quality/parameters", s.handleSaveQualityParameter)
	s.handle(mux, "GET /api/v1/quality/specs", s.handleListQualitySpecs)
	s.handle(mux, "PUT /api/v1/quality/specs", s.handleSaveQualitySpec)
	s.handle(mux, "GET /api/v1/quality/samples", s.handleListSamples)
	s.handle(mux, "POST /api/v1/quality/samples", s.handleCreateSample)
	s.handle(mux, "GET /api/v1/quality/samples/{id}", s.handleGetSample)
	s.handle(mux, "POST /api/v1/quality/samples/{id}/results", s.handleRecordResults)
	s.handle(mux, "GET /api/v1/quality/holds", s.handleListHolds)
	s.handle(mux, "POST /api/v1/quality/holds", s.handlePlaceHold)
	s.handle(mux, "POST /api/v1/quality/holds/{id}/release", s.handleReleaseHold)

	// --- reports ------------------------------------------------------------
	s.handle(mux, "GET /api/v1/reports", s.handleListReports)
	s.handle(mux, "GET /api/v1/reports/{code}", s.handleRunReport)

	// --- audit --------------------------------------------------------------
	s.handle(mux, "GET /api/v1/audit", s.handleAudit)

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
// handleDevUsers lists the demonstration accounts for the sign-in page.
//
// The list comes from the server's own configuration rather than from a copy
// kept in the browser, so an account added to one and forgotten in the other
// cannot happen. In oidc mode it is empty, and the sign-in page redirects to
// the identity provider instead.
func (s *Server) handleDevUsers(w http.ResponseWriter, r *http.Request) {
	type devUser struct {
		Username    string   `json:"username"`
		DisplayName string   `json:"displayName"`
		Roles       []string `json:"roles"`
	}
	out := []devUser{}
	if s.verifier.Mode() == "dev" {
		for name, u := range s.authCfg.DevUsers {
			out = append(out, devUser{Username: name, DisplayName: u.DisplayName, Roles: u.Roles})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Username < out[j].Username })
	}
	writeJSON(w, pageResponse[devUser]{Value: out, Count: len(out)})
}

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
