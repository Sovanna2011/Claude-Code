package api_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The import over real HTTP: the upload is the body, everything else is a query
// parameter, and nothing reaches the plan until somebody commits.

// upload posts a file to the staging endpoint.
func upload(t *testing.T, ts *testServer, user, query, body string,
	headers ...string,
) *httptest.ResponseRecorder {

	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/imports?"+query,
		bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "text/csv")
	req.Header.Set("Authorization", "Bearer "+ts.tokens[user])
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	ts.handler.ServeHTTP(rec, req)
	return rec
}

const sheet = "Date,Delivered (MT),Accepted (MT),Crushed (MT),Rate (TPH),Remarks\n" +
	"01/12/2026,\"17,000.000\",\"16,900.000\",\"16,788.321\",700,first day\n" +
	"02/12/2026,17000,16900,not a number,700,broken\n"

func TestTheImportRoundTripOverHTTP(t *testing.T) {
	ts := newTestServer(t)
	query := "mapping=CANE-DAILY&versionId=" + ts.seeded.BudgetID + "&series=PLAN&fileName=cane.csv"

	rec := upload(t, ts, "planner", query, sheet, "Idempotency-Key", "upload-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("stage: %d %s", rec.Code, rec.Body.String())
	}
	preview := decode(t, rec)
	job := preview["job"].(map[string]any)
	if job["status"] != "VALIDATED" {
		t.Fatalf("status = %v", job["status"])
	}
	if job["validRows"] != float64(1) || job["errorRows"] != float64(1) {
		t.Errorf("counts = %v valid, %v error", job["validRows"], job["errorRows"])
	}
	id := job["id"].(string)

	// A retried upload must not stage the file twice.
	rec = upload(t, ts, "planner", query, sheet, "Idempotency-Key", "upload-1")
	if rec.Header().Get("Idempotent-Replay") != "true" {
		t.Error("a retried upload must be answered from the first response")
	}

	// The error download is what somebody opens next to their spreadsheet.
	rec = ts.do(t, "planner", http.MethodGet, "/api/v1/imports/"+id+"/errors", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("errors: %d %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.HasPrefix(body, "Row,Field,Problem,Value") {
		t.Errorf("the error file must be a readable CSV, got %q", body[:min(60, len(body))])
	}
	// Line 3 of the file is the broken one, and the row number is the first
	// column because that is the line somebody has to go to.
	if !strings.Contains(body, "\n3,caneCrushed,") {
		t.Errorf("the error file must name the line and the field: %q", body)
	}

	// All or nothing: the sound row is not written while a bad one remains.
	rec = ts.do(t, "planner", http.MethodPost, "/api/v1/imports/"+id+"/commit", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("a file with a bad row must be refused whole: %d %s", rec.Code, rec.Body.String())
	}

	rec = ts.do(t, "planner", http.MethodPost, "/api/v1/imports/"+id+"/commit",
		map[string]any{"partial": true})
	if rec.Code != http.StatusOK {
		t.Fatalf("partial commit: %d %s", rec.Code, rec.Body.String())
	}
	result := decode(t, rec)
	if result["written"] != float64(1) || result["skipped"] != float64(1) {
		t.Errorf("written %v skipped %v", result["written"], result["skipped"])
	}

	// The figure is in the plan.
	rec = ts.do(t, "planner", http.MethodGet,
		"/api/v1/versions/"+ts.seeded.BudgetID+"/cane?from=2026-12-01&to=2026-12-01&series=PLAN", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("read cane: %d %s", rec.Code, rec.Body.String())
	}
	rows := decode(t, rec)["value"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["caneDelivered"] != "17000" {
		t.Errorf("the committed figure must be in the plan: %v", rows)
	}

	// And committing again is refused.
	rec = ts.do(t, "planner", http.MethodPost, "/api/v1/imports/"+id+"/commit",
		map[string]any{"partial": true})
	if rec.Code != http.StatusConflict {
		t.Errorf("a second commit: status = %d, want 409", rec.Code)
	}
}

func TestAnUploadWithNoMappingOrAnUnreadableFileIsRefused(t *testing.T) {
	ts := newTestServer(t)

	rec := upload(t, ts, "planner", "versionId="+ts.seeded.BudgetID, sheet)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("an upload naming no mapping: %d %s", rec.Code, rec.Body.String())
	}

	rec = upload(t, ts, "planner",
		"mapping=NOSUCH&versionId="+ts.seeded.BudgetID, sheet)
	if rec.Code != http.StatusNotFound {
		t.Errorf("an unknown mapping: %d %s", rec.Code, rec.Body.String())
	}

	// A file of headings and nothing else imports nothing, and says so.
	rec = upload(t, ts, "planner",
		"mapping=CANE-DAILY&versionId="+ts.seeded.BudgetID+"&fileName=cane.csv",
		"Date,Crushed (MT)\n")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("a file with no data rows: %d %s", rec.Code, rec.Body.String())
	}

	// A format this system cannot read is named rather than guessed at.
	rec = upload(t, ts, "planner",
		"mapping=CANE-DAILY&versionId="+ts.seeded.BudgetID+"&fileName=plan.pdf", sheet)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("a .pdf: %d %s", rec.Code, rec.Body.String())
	}
}

func TestImportPermissionsAreCheckedBeforeAnybodyReadsAPreview(t *testing.T) {
	ts := newTestServer(t)

	// A keeper may post stock, not plan rows - and is told so at the upload
	// rather than after reading a preview of figures they may not post.
	rec := upload(t, ts, "keeper",
		"mapping=CANE-DAILY&versionId="+ts.seeded.BudgetID+"&fileName=cane.csv", sheet)
	if rec.Code != http.StatusForbidden {
		t.Errorf("a keeper staging plan rows: %d", rec.Code)
	}

	// A planner may not post actuals whatever they upload.
	rec = upload(t, ts, "planner",
		"mapping=CANE-DAILY&versionId="+ts.seeded.ActualID+"&series=ACTUAL&fileName=cane.csv", sheet)
	if rec.Code != http.StatusForbidden {
		t.Errorf("a planner staging actual cane: %d %s", rec.Code, rec.Body.String())
	}

	// Changing what a column means is master data.
	mapping := map[string]any{
		"code": "MINE", "name": "Mine", "kind": "DAILY_CANE", "headerRow": 1, "active": true,
		"columns": []map[string]any{{"field": "businessDate", "header": "Date"}},
	}
	rec = ts.do(t, "planner", http.MethodPut, "/api/v1/import-mappings", mapping)
	if rec.Code != http.StatusForbidden {
		t.Errorf("a planner saving a mapping: %d", rec.Code)
	}
	rec = ts.do(t, "masterdata", http.MethodPut, "/api/v1/import-mappings", mapping)
	if rec.Code != http.StatusOK {
		t.Fatalf("master data saving a mapping: %d %s", rec.Code, rec.Body.String())
	}
}

func TestTheMappingCatalogueDescribesEachKindOfFile(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "planner", http.MethodGet, "/api/v1/import-fields", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("fields: %d %s", rec.Code, rec.Body.String())
	}
	kinds := decode(t, rec)["value"].([]any)
	if len(kinds) != 4 {
		t.Fatalf("four kinds of file, got %d", len(kinds))
	}

	found := false
	for _, k := range kinds {
		kind := k.(map[string]any)
		if kind["kind"] != "DAILY_CANE" {
			continue
		}
		found = true
		fields := kind["fields"].([]any)
		if len(fields) == 0 {
			t.Fatal("a kind with no fields could map nothing")
		}
		first := fields[0].(map[string]any)
		if first["name"] != "businessDate" || first["required"] != true {
			t.Errorf("the date is the required field of a daily sheet: %v", first)
		}
		// The keys are what decide whether two rows are the same row, and the
		// screen shows them so somebody knows what a duplicate means.
		if len(kind["keys"].([]any)) == 0 {
			t.Error("a kind must say what identifies a row")
		}
	}
	if !found {
		t.Error("the catalogue must describe the daily cane file")
	}

	// The seeded templates are there to be used without designing one first.
	rec = ts.do(t, "planner", http.MethodGet, "/api/v1/import-mappings?kind=DAILY_CANE", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("mappings: %d %s", rec.Code, rec.Body.String())
	}
	if len(decode(t, rec)["value"].([]any)) == 0 {
		t.Error("the demonstration scenario ships a cane mapping")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
