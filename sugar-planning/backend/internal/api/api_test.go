package api_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kss/sugarplan/internal/api"
	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/seed"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store/memory"
)

const devSecret = "test-development-secret"

type testServer struct {
	handler http.Handler
	seeded  seed.Result
	tokens  map[string]string
}

// newTestServer boots the whole stack over an in-memory store with the
// reference scenario loaded, exactly as the demo profile does.
func newTestServer(t *testing.T) *testServer {
	t.Helper()
	ctx := context.Background()

	st := memory.New()
	planning := service.NewPlanning(st, func() time.Time { return time.Now().UTC() })
	res, err := seed.LoadWithActuals(ctx, st, planning, 14)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Development accounts, scoped to the seeded factory as the server does.
	users := map[string]auth.DevUser{
		"planner":  {DisplayName: "Planner", Roles: []string{auth.RoleProductionPlanner}},
		"approver": {DisplayName: "Approver", Roles: []string{auth.RoleApprover}},
		// A factory manager holds two roles. Closing an order that missed its
		// plan needs both the production permission and the approval authority,
		// and no single role carries both - which is the separation of duties
		// working, not a gap. Granting both is a deliberate act.
		"manager": {DisplayName: "Factory manager",
			Roles: []string{auth.RoleApprover, auth.RoleShiftSupervisor}},
		"supervisor": {DisplayName: "Supervisor", Roles: []string{auth.RoleShiftSupervisor}},
		"keeper":     {DisplayName: "Warehouse operator", Roles: []string{auth.RoleWarehouseOperator}},
		"lab":        {DisplayName: "Laboratory", Roles: []string{auth.RoleQualityUser}},
		"masterdata": {DisplayName: "Master data", Roles: []string{auth.RoleMasterDataAdmin}},
		"controller": {DisplayName: "Cost controller", Roles: []string{auth.RoleCostController}},
		"executive":  {DisplayName: "Executive", Roles: []string{auth.RoleExecutiveViewer}},
		"auditor":    {DisplayName: "Auditor", Roles: []string{auth.RoleAuditor}},
		"admin":      {DisplayName: "Administrator", Roles: []string{auth.RoleSystemAdmin}},
		"interface":  {DisplayName: "Gate interface", Roles: []string{auth.RoleIntegration}},
		"outsider":   {DisplayName: "Other factory", Roles: []string{auth.RoleProductionPlanner}},
	}
	for name, u := range users {
		if name != "outsider" {
			u.Companies = []string{res.CompanyID}
			u.Factories = []string{res.FactoryID}
		} else {
			u.Companies = []string{"other-company"}
			u.Factories = []string{"other-factory"}
		}
		users[name] = u
	}

	authCfg := auth.Config{Mode: "dev", DevSecret: devSecret, DevUsers: users}
	verifier, err := auth.NewVerifier(ctx, authCfg)
	if err != nil {
		t.Fatalf("verifier: %v", err)
	}

	handler := api.NewServer(api.Options{
		Store: st, Planning: planning,
		Analytics: service.NewAnalytics(st, planning),
		Materials: service.NewMaterials(st, planning),
		Verifier:  verifier, AuthCfg: authCfg,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Version: "test",
	})

	ts := &testServer{handler: handler, seeded: res, tokens: map[string]string{}}
	for name := range users {
		token, _, err := auth.IssueDevToken(authCfg, name, time.Hour)
		if err != nil {
			t.Fatalf("issue token for %s: %v", name, err)
		}
		ts.tokens[name] = token
	}
	return ts
}

// do issues a request as the named user.
func (ts *testServer) do(t *testing.T, user, method, path string, body any, headers ...string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if user != "" {
		req.Header.Set("Authorization", "Bearer "+ts.tokens[user])
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	ts.handler.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response (%s): %v", rec.Body.String(), err)
	}
	return out
}

// versionID finds the seeded budget version.
func (ts *testServer) versionID() string { return ts.seeded.BudgetID }

// ---------------------------------------------------------------------------
// Authentication and authorisation
// ---------------------------------------------------------------------------

func TestUnauthenticatedRequestsAreRejected(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "", http.MethodGet, "/api/v1/seasons", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if got := rec.Header().Get("WWW-Authenticate"); !strings.Contains(got, "Bearer") {
		t.Errorf("WWW-Authenticate = %q", got)
	}
	body := decode(t, rec)
	if body["title"] != "Authentication required" {
		t.Errorf("problem title = %v", body["title"])
	}

	// A forged token is refused, and the response says nothing about why.
	rec = ts.do(t, "", http.MethodGet, "/api/v1/seasons", nil, "Authorization", "Bearer not-a-token")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("forged token status = %d, want 401", rec.Code)
	}
	if detail, _ := decode(t, rec)["detail"].(string); strings.Contains(strings.ToLower(detail), "signature") {
		t.Errorf("the failure reason must not leak to the client: %q", detail)
	}
}

func TestHealthAndReadinessArePublic(t *testing.T) {
	ts := newTestServer(t)
	for _, path := range []string{"/healthz", "/readyz"} {
		if rec := ts.do(t, "", http.MethodGet, path, nil); rec.Code != http.StatusOK {
			t.Errorf("%s status = %d, want 200", path, rec.Code)
		}
	}
}

func TestPermissionsAreEnforcedPerEndpoint(t *testing.T) {
	ts := newTestServer(t)

	// An executive viewer may read the dashboard but not write a plan.
	rec := ts.do(t, "executive", http.MethodGet,
		"/api/v1/dashboard?seasonId="+ts.seeded.SeasonID, nil)
	if rec.Code != http.StatusOK {
		t.Errorf("executive reading the dashboard = %d, want 200 (%s)", rec.Code, rec.Body)
	}

	rec = ts.do(t, "executive", http.MethodPost,
		"/api/v1/versions/"+ts.versionID()+"/cane",
		map[string]any{"rows": []map[string]any{
			{"businessDate": "2027-03-01", "series": "PLAN", "caneCrushed": 100, "availableHours": 24},
		}})
	if rec.Code != http.StatusForbidden && rec.Code != http.StatusBadRequest {
		t.Errorf("executive writing a plan = %d, want 403 or 400", rec.Code)
	}

	// Only an auditor (or admin) may read the audit trail.
	if rec := ts.do(t, "auditor", http.MethodGet, "/api/v1/audit", nil); rec.Code != http.StatusOK {
		t.Errorf("auditor reading the audit trail = %d, want 200", rec.Code)
	}
	if rec := ts.do(t, "planner", http.MethodGet, "/api/v1/audit", nil); rec.Code != http.StatusForbidden {
		t.Errorf("planner reading the audit trail = %d, want 403", rec.Code)
	}
}

func TestDataScopeIsEnforcedOverHTTP(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "outsider", http.MethodGet, "/api/v1/seasons", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if count := decode(t, rec)["count"]; count != float64(0) {
		t.Errorf("an out-of-scope planner sees %v seasons, want 0", count)
	}

	rec = ts.do(t, "outsider", http.MethodGet, "/api/v1/seasons/"+ts.seeded.SeasonID, nil)
	if rec.Code != http.StatusForbidden {
		t.Errorf("reading an out-of-scope season = %d, want 403", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Problem details
// ---------------------------------------------------------------------------

func TestValidationErrorsCarryRowAndField(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "supervisor", http.MethodPost,
		"/api/v1/versions/"+ts.seeded.ActualID+"/cane",
		map[string]any{"rows": []map[string]any{
			{"businessDate": "2026-12-01", "series": "ACTUAL", "caneCrushed": 15000, "availableHours": 24},
			{"businessDate": "2026-12-02", "series": "ACTUAL", "caneCrushed": -5, "availableHours": 24},
		}})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (%s)", rec.Code, rec.Body)
	}
	body := decode(t, rec)
	errs, ok := body["errors"].([]any)
	if !ok || len(errs) == 0 {
		t.Fatalf("expected field errors, got %s", rec.Body)
	}
	first := errs[0].(map[string]any)
	// Row 1 must be reported as 1, and row 0 would be reported as 0 rather
	// than omitted, so the grid can highlight the right line.
	if first["row"] != float64(1) {
		t.Errorf("row = %v, want 1", first["row"])
	}
	if first["field"] != "caneCrushed" {
		t.Errorf("field = %v, want caneCrushed", first["field"])
	}
	if body["correlationId"] == "" {
		t.Error("a problem document must carry the correlation id")
	}
}

func TestUnknownFieldsAreRejected(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.do(t, "planner", http.MethodPost, "/api/v1/seasons",
		map[string]any{"code": "2027-2028", "factoryId": ts.seeded.FactoryID,
			"companyId": ts.seeded.CompanyID, "startDate": "2027-12-01", "typo": true})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("a payload with an unknown field = %d, want 400", rec.Code)
	}
}

func TestNotFoundIsAProblemDocument(t *testing.T) {
	ts := newTestServer(t)
	rec := ts.do(t, "planner", http.MethodGet,
		"/api/v1/versions/00000000-0000-0000-0000-000000000000", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	body := decode(t, rec)
	if body["status"] != float64(404) || body["type"] == "" {
		t.Errorf("problem document = %s", rec.Body)
	}
	// No internal detail leaks.
	if detail, _ := body["detail"].(string); strings.Contains(strings.ToLower(detail), "select ") {
		t.Errorf("SQL leaked into the response: %q", detail)
	}
}

// ---------------------------------------------------------------------------
// Concurrency
// ---------------------------------------------------------------------------

func TestETagAndIfMatch(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "planner", http.MethodGet, "/api/v1/seasons/"+ts.seeded.SeasonID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("read season: %d", rec.Code)
	}
	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("the season response must carry an ETag")
	}
	season := decode(t, rec)
	season["name"] = "Renamed season"
	delete(season, "rowVersion")

	// A correct If-Match succeeds.
	rec = ts.do(t, "planner", http.MethodPut, "/api/v1/seasons/"+ts.seeded.SeasonID, season,
		"If-Match", etag)
	if rec.Code != http.StatusOK {
		t.Fatalf("update with a matching ETag = %d (%s)", rec.Code, rec.Body)
	}
	newETag := rec.Header().Get("ETag")
	if newETag == etag {
		t.Error("the ETag must change after an update")
	}

	// Re-sending the stale ETag is a precondition failure.
	rec = ts.do(t, "planner", http.MethodPut, "/api/v1/seasons/"+ts.seeded.SeasonID, season,
		"If-Match", etag)
	if rec.Code != http.StatusPreconditionFailed {
		t.Errorf("update with a stale ETag = %d, want 412 (%s)", rec.Code, rec.Body)
	}
}

// ---------------------------------------------------------------------------
// Workflow over HTTP
// ---------------------------------------------------------------------------

func TestWorkflowOverHTTP(t *testing.T) {
	ts := newTestServer(t)
	version := ts.versionID()

	transition := func(user, action, reason string) *httptest.ResponseRecorder {
		payload := map[string]any{"action": action}
		if reason != "" {
			payload["reason"] = reason
		}
		return ts.do(t, user, http.MethodPost, "/api/v1/versions/"+version+"/transition", payload)
	}

	if rec := transition("planner", "SUBMIT", ""); rec.Code != http.StatusOK {
		t.Fatalf("submit = %d (%s)", rec.Code, rec.Body)
	}
	// Separation of duties.
	if rec := transition("planner", "APPROVE", ""); rec.Code != http.StatusForbidden {
		t.Errorf("planner approving = %d, want 403", rec.Code)
	}
	if rec := transition("approver", "APPROVE", ""); rec.Code != http.StatusOK {
		t.Fatalf("approve = %d (%s)", rec.Code, rec.Body)
	}
	// An action that is not available in this status is a conflict.
	if rec := transition("approver", "APPROVE", ""); rec.Code != http.StatusConflict {
		t.Errorf("approving twice = %d, want 409", rec.Code)
	}
	if rec := transition("approver", "RELEASE", ""); rec.Code != http.StatusOK {
		t.Fatalf("release = %d (%s)", rec.Code, rec.Body)
	}
	// Reopen without a reason is a validation failure.
	if rec := transition("approver", "REOPEN", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("reopen without a reason = %d, want 400", rec.Code)
	}
	if rec := transition("approver", "REOPEN", "cane forecast revised"); rec.Code != http.StatusOK {
		t.Errorf("reopen with a reason = %d (%s)", rec.Code, rec.Body)
	}

	// The trail is readable by an auditor.
	rec := ts.do(t, "auditor", http.MethodGet,
		"/api/v1/audit?entity=plan_version&entityId="+version, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("audit = %d", rec.Code)
	}
	if count, _ := decode(t, rec)["count"].(float64); count < 4 {
		t.Errorf("audit events = %v, want at least the four workflow actions", count)
	}
}

// ---------------------------------------------------------------------------
// Reports and exports
// ---------------------------------------------------------------------------

func TestReportCatalogueMatchesTheImplementedReports(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "planner", http.MethodGet, "/api/v1/reports", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("catalogue = %d", rec.Code)
	}
	var body struct {
		Value []struct {
			Code       string   `json:"code"`
			Parameters []string `json:"parameters"`
		} `json:"value"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode catalogue: %v", err)
	}
	if len(body.Value) == 0 {
		t.Fatal("the catalogue is empty")
	}

	// Every advertised report must actually run.
	for _, def := range body.Value {
		params := "versionId=" + ts.versionID() + "&seasonId=" + ts.seeded.SeasonID
		rec := ts.do(t, "planner", http.MethodGet,
			"/api/v1/reports/"+def.Code+"?"+params, nil)
		if rec.Code != http.StatusOK {
			t.Errorf("report %s = %d (%s)", def.Code, rec.Code, rec.Body)
		}
	}
}

// The catalogue is per caller, not one list with a rule for reading it.
//
// This is here because the acceptance harness found the alternative running:
// the packaging requirement report needs materials:read, which only the planner
// holds, and it was on every role's Reports page answering 403 to all of them. A
// menu that offers something it will refuse is worse than one that does not
// offer it.
func TestTheReportCatalogueOffersNothingItWillRefuse(t *testing.T) {
	ts := newTestServer(t)
	params := "?versionId=" + ts.versionID() + "&seasonId=" + ts.seeded.SeasonID

	for _, user := range []string{"planner", "executive", "keeper", "auditor"} {
		rec := ts.do(t, user, http.MethodGet, "/api/v1/reports", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("catalogue as %s = %d", user, rec.Code)
		}
		var body struct {
			Value []struct {
				Code string `json:"code"`
			} `json:"value"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode catalogue as %s: %v", user, err)
		}
		if len(body.Value) == 0 {
			t.Fatalf("%s is offered no reports at all", user)
		}
		for _, def := range body.Value {
			rec := ts.do(t, user, http.MethodGet, "/api/v1/reports/"+def.Code+params, nil)
			if rec.Code == http.StatusForbidden {
				t.Errorf("%s is offered %s and refused it: %s", user, def.Code, rec.Body)
			}
		}
	}

	// And the other half. Leaving a report off somebody's catalogue is a
	// courtesy; the refusal when they name it directly is the control.
	rec := ts.do(t, "executive", http.MethodGet, "/api/v1/reports/material-requirements"+params, nil)
	if rec.Code != http.StatusForbidden {
		t.Errorf("an executive viewer naming the packaging requirement report directly = %d, "+
			"want 403", rec.Code)
	}
}

func TestExportFormats(t *testing.T) {
	ts := newTestServer(t)
	base := "/api/v1/reports/daily-plan?versionId=" + ts.versionID() +
		"&from=2026-12-01&to=2026-12-05&format="

	t.Run("csv carries the metadata and a byte order mark", func(t *testing.T) {
		rec := ts.do(t, "planner", http.MethodGet, base+"csv", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		body := rec.Body.Bytes()
		if !bytes.HasPrefix(body, []byte{0xEF, 0xBB, 0xBF}) {
			t.Error("the CSV must start with a UTF-8 byte order mark for Excel")
		}
		text := string(body)
		for _, want := range []string{"Kampong Speu", "Season: 2026-2027", "Plan version: V1", "Generated by"} {
			if !strings.Contains(text, want) {
				t.Errorf("the export lost its metadata: %q missing", want)
			}
		}
		if !strings.Contains(rec.Header().Get("Content-Disposition"), "attachment") {
			t.Error("an export must be sent as an attachment")
		}
	})

	t.Run("xlsx is a readable workbook with numeric cells", func(t *testing.T) {
		rec := ts.do(t, "planner", http.MethodGet, base+"xlsx", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		body := rec.Body.Bytes()
		zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			t.Fatalf("the workbook is not a readable zip: %v", err)
		}
		found := map[string]bool{}
		var sheet string
		for _, f := range zr.File {
			found[f.Name] = true
			if f.Name == "xl/worksheets/sheet1.xml" {
				rc, err := f.Open()
				if err != nil {
					t.Fatalf("open sheet: %v", err)
				}
				raw, _ := io.ReadAll(rc)
				rc.Close()
				sheet = string(raw)
			}
		}
		for _, part := range []string{
			"[Content_Types].xml", "_rels/.rels", "xl/workbook.xml",
			"xl/styles.xml", "xl/worksheets/sheet1.xml",
		} {
			if !found[part] {
				t.Errorf("the workbook is missing %s", part)
			}
		}
		// A tonnage must be a number, not a string, or the recipient cannot sum it.
		if !strings.Contains(sheet, "<v>16788.321</v>") {
			t.Error("tonnages must be written as numeric cells")
		}
	})

	t.Run("pdf is a well formed document", func(t *testing.T) {
		rec := ts.do(t, "planner", http.MethodGet, base+"pdf", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		body := rec.Body.Bytes()
		if !bytes.HasPrefix(body, []byte("%PDF-1.4")) {
			t.Error("missing the PDF header")
		}
		if !bytes.Contains(body, []byte("startxref")) || !bytes.HasSuffix(bytes.TrimSpace(body), []byte("%%EOF")) {
			t.Error("missing the cross-reference table or the end marker")
		}
		if !bytes.Contains(body, []byte("/Type /Catalog")) {
			t.Error("missing the document catalogue")
		}
		if rec.Header().Get("Content-Type") != "application/pdf" {
			t.Errorf("content type = %q", rec.Header().Get("Content-Type"))
		}
	})

	t.Run("an unknown format is rejected", func(t *testing.T) {
		rec := ts.do(t, "planner", http.MethodGet, base+"docx", nil)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// Idempotency
// ---------------------------------------------------------------------------

func TestIdempotencyKeyReplaysInsteadOfRepeating(t *testing.T) {
	ts := newTestServer(t)

	scenario := ts.do(t, "planner", http.MethodPost,
		"/api/v1/versions/"+ts.versionID()+"/copy",
		map[string]any{"code": "IDEM-TEST", "planType": "WHATIF"})
	if scenario.Code != http.StatusCreated {
		t.Fatalf("copy = %d (%s)", scenario.Code, scenario.Body)
	}
	newID, _ := decode(t, scenario)["id"].(string)

	path := "/api/v1/versions/" + newID + "/generate"
	first := ts.do(t, "planner", http.MethodPost, path, map[string]any{"replace": true},
		"Idempotency-Key", "abc-123")
	if first.Code != http.StatusOK {
		t.Fatalf("first generate = %d (%s)", first.Code, first.Body)
	}

	second := ts.do(t, "planner", http.MethodPost, path, map[string]any{"replace": true},
		"Idempotency-Key", "abc-123")
	if second.Header().Get("Idempotent-Replay") != "true" {
		t.Errorf("the second call with the same key must be flagged as a replay (status %d)", second.Code)
	}
}

// ---------------------------------------------------------------------------
// Specification
// ---------------------------------------------------------------------------

func TestOpenAPIIsServedAndDescribesTheRoutes(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "", http.MethodGet, "/api/v1/openapi.yaml", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	spec := rec.Body.String()
	if !strings.HasPrefix(spec, "openapi: 3.1") {
		t.Errorf("not an OpenAPI 3.1 document: %.40s", spec)
	}

	// Every path the specification advertises must be routed. This is what keeps
	// the document honest as the API grows.
	for _, path := range documentedPaths(spec) {
		concrete := strings.NewReplacer(
			"{id}", ts.versionID(),
			"{mixId}", "00000000-0000-0000-0000-000000000000",
			"{entity}", "products",
			"{code}", "daily-plan",
		).Replace(path)

		rec := ts.do(t, "planner", http.MethodGet, "/api/v1"+concrete, nil)
		if rec.Code == http.StatusNotFound && !strings.Contains(rec.Body.String(), "not-found") {
			t.Errorf("%s is documented but not routed", path)
		}
		if rec.Code == http.StatusMethodNotAllowed {
			continue // documented for POST or PUT only
		}
	}
}

// documentedPaths extracts the top-level path keys from the specification.
func documentedPaths(spec string) []string {
	var out []string
	inPaths := false
	for _, line := range strings.Split(spec, "\n") {
		if strings.HasPrefix(line, "paths:") {
			inPaths = true
			continue
		}
		if inPaths && len(line) > 0 && line[0] != ' ' {
			break // a new top-level key
		}
		if inPaths && strings.HasPrefix(line, "  /") && strings.HasSuffix(strings.TrimSpace(line), ":") {
			out = append(out, strings.TrimSuffix(strings.TrimSpace(line), ":"))
		}
	}
	return out
}

func TestDevLoginIsRefusedOutsideDevMode(t *testing.T) {
	// A server configured for OIDC must not mint tokens, whatever is posted.
	ctx := context.Background()
	st := memory.New()
	planning := service.NewPlanning(st, nil)

	handler := api.NewServer(api.Options{
		Store: st, Planning: planning,
		Analytics: service.NewAnalytics(st, planning),
		Materials: service.NewMaterials(st, planning),
		Verifier:  stubVerifier{},
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	_ = ctx

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/dev-login",
		strings.NewReader(`{"username":"admin"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("dev login in oidc mode = %d, want 400", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "accessToken") {
		t.Error("a token was issued in oidc mode")
	}
}

// stubVerifier stands in for a configured OIDC verifier.
type stubVerifier struct{}

func (stubVerifier) Mode() string { return "oidc" }
func (stubVerifier) Verify(context.Context, string) (auth.Principal, error) {
	return auth.Principal{}, fmt.Errorf("no token accepted in this test")
}
