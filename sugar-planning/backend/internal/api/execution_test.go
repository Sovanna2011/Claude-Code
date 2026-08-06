package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// These tests drive the execution endpoints over real HTTP, so they cover the
// things only the transport can get wrong: routing, status codes, the problem
// document, ETags and the idempotency key.

func TestPostingMovesStockOverHTTP(t *testing.T) {
	ts := newTestServer(t)
	warehouse := ts.seeded.Warehouses["FG-WH1"]
	product := ts.seeded.Products["REF"]

	rec := ts.do(t, "keeper", http.MethodPost, "/api/v1/inventory/documents", map[string]any{
		"docType": "RECEIPT", "businessDate": "2026-12-05", "factoryId": ts.seeded.FactoryID,
		"lines": []map[string]any{{
			"warehouseId": warehouse, "productId": product, "quantity": "500",
		}},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	document := decode(t, rec)
	if document["documentNo"] == nil || !strings.HasPrefix(document["documentNo"].(string), "MD-") {
		t.Errorf("document number = %v", document["documentNo"])
	}

	rec = ts.do(t, "keeper", http.MethodGet,
		"/api/v1/stock?warehouseId="+warehouse+"&productId="+product, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("stock status = %d, body = %s", rec.Code, rec.Body.String())
	}
	stock := decode(t, rec)
	lines := stock["value"].([]any)
	if len(lines) != 1 {
		t.Fatalf("stock lines = %d, want 1", len(lines))
	}
	line := lines[0].(map[string]any)
	if line["quantity"] != "500" {
		t.Errorf("quantity = %v, want the string \"500\"; quantities must stay exact on the wire",
			line["quantity"])
	}
	if line["warehouseCode"] != "FG-WH1" {
		t.Errorf("the stock line must name its warehouse, got %v", line["warehouseCode"])
	}
}

func TestARefusedPostingReturnsTheOffendingLine(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "keeper", http.MethodPost, "/api/v1/inventory/documents", map[string]any{
		"docType": "ISSUE", "businessDate": "2026-12-05", "factoryId": ts.seeded.FactoryID,
		"lines": []map[string]any{{
			"warehouseId": ts.seeded.Warehouses["FG-WH1"],
			"productId":   ts.seeded.Products["REF"], "quantity": "10",
		}},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/problem+json") {
		t.Errorf("content type = %q, want a problem document", ct)
	}
	problem := decode(t, rec)
	errs, ok := problem["errors"].([]any)
	if !ok || len(errs) == 0 {
		t.Fatalf("the problem must carry the field errors: %s", rec.Body.String())
	}
	first := errs[0].(map[string]any)
	if first["code"] != "NEGATIVE_STOCK" {
		t.Errorf("code = %v, want NEGATIVE_STOCK", first["code"])
	}
	if first["row"] == nil {
		t.Error("the error must address a line so the table can highlight it")
	}
}

func TestARetriedPostingReplaysInsteadOfPostingTwice(t *testing.T) {
	ts := newTestServer(t)
	warehouse := ts.seeded.Warehouses["FG-WH1"]
	product := ts.seeded.Products["REF"]

	body := map[string]any{
		"docType": "RECEIPT", "businessDate": "2026-12-05", "factoryId": ts.seeded.FactoryID,
		"lines": []map[string]any{{
			"warehouseId": warehouse, "productId": product, "quantity": "250",
		}},
	}

	first := ts.do(t, "keeper", http.MethodPost, "/api/v1/inventory/documents", body,
		"Idempotency-Key", "posting-1")
	if first.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", first.Code, first.Body.String())
	}

	second := ts.do(t, "keeper", http.MethodPost, "/api/v1/inventory/documents", body,
		"Idempotency-Key", "posting-1")
	if second.Code != http.StatusOK {
		t.Fatalf("replay status = %d, body = %s", second.Code, second.Body.String())
	}
	if second.Header().Get("Idempotent-Replay") != "true" {
		t.Error("the retry must be marked as a replay")
	}
	// The retry gets the document the first request produced, not an
	// acknowledgement: a client that lost the first response still ends up with
	// the posting it needs to show.
	if second.Body.String() != first.Body.String() {
		t.Errorf("the replay must repeat the first response\nfirst:  %s\nsecond: %s",
			first.Body.String(), second.Body.String())
	}

	rec := ts.do(t, "keeper", http.MethodGet,
		"/api/v1/stock?warehouseId="+warehouse+"&productId="+product, nil)
	line := decode(t, rec)["value"].([]any)[0].(map[string]any)
	if line["quantity"] != "250" {
		t.Errorf("balance = %v, want 250; the retry must not have posted again", line["quantity"])
	}
}

func TestPostingIsRefusedWithoutTheStockPermission(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "planner", http.MethodPost, "/api/v1/inventory/documents", map[string]any{
		"docType": "RECEIPT", "businessDate": "2026-12-05", "factoryId": ts.seeded.FactoryID,
		"lines": []map[string]any{{
			"warehouseId": ts.seeded.Warehouses["FG-WH1"],
			"productId":   ts.seeded.Products["REF"], "quantity": "1",
		}},
	})
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403; body = %s", rec.Code, rec.Body.String())
	}
}

func TestTheOrderLifeCycleOverHTTP(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "supervisor", http.MethodPost, "/api/v1/production-orders", map[string]any{
		"factoryId": ts.seeded.FactoryID, "businessDate": "2026-12-05",
		"productId": ts.seeded.Products["REF"], "plannedQty": "300",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	order := decode(t, rec)
	id := order["id"].(string)
	if rec.Header().Get("ETag") == "" {
		t.Error("a created order must carry an ETag")
	}

	// The If-Match header is the row version, so a stale tab cannot act.
	stale := ts.do(t, "supervisor", http.MethodPost,
		"/api/v1/production-orders/"+id+"/action", map[string]any{"action": "RELEASE"},
		"If-Match", `"99"`)
	if stale.Code != http.StatusPreconditionFailed {
		t.Errorf("stale action status = %d, want 412; body = %s", stale.Code, stale.Body.String())
	}

	etag := rec.Header().Get("ETag")
	released := ts.do(t, "supervisor", http.MethodPost,
		"/api/v1/production-orders/"+id+"/action", map[string]any{"action": "RELEASE"},
		"If-Match", etag)
	if released.Code != http.StatusOK {
		t.Fatalf("release status = %d, body = %s", released.Code, released.Body.String())
	}
	if decode(t, released)["status"] != "RELEASED" {
		t.Errorf("status = %v, want RELEASED", decode(t, released)["status"])
	}

	confirmed := ts.do(t, "supervisor", http.MethodPost,
		"/api/v1/production-orders/"+id+"/confirm", map[string]any{
			"businessDate": "2026-12-05", "yieldQty": "300",
			"warehouseId": ts.seeded.Warehouses["FG-WH1"],
		})
	if confirmed.Code != http.StatusOK {
		t.Fatalf("confirm status = %d, body = %s", confirmed.Code, confirmed.Body.String())
	}
	result := decode(t, confirmed)
	if result["order"].(map[string]any)["status"] != "COMPLETED" {
		t.Errorf("order status = %v, want COMPLETED", result["order"].(map[string]any)["status"])
	}
	if result["document"] == nil {
		t.Error("the confirmation must return the goods receipt it posted")
	}

	detail := ts.do(t, "supervisor", http.MethodGet, "/api/v1/production-orders/"+id, nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("get order status = %d", detail.Code)
	}
	body := decode(t, detail)
	if len(body["confirmations"].([]any)) != 1 {
		t.Errorf("confirmations = %v", body["confirmations"])
	}
	if actions, ok := body["allowedActions"].([]any); !ok || len(actions) == 0 {
		t.Error("the detail must say what may be done next")
	}
}

func TestTheLaboratoryEndpointsBlockAndFreeStock(t *testing.T) {
	ts := newTestServer(t)
	warehouse := ts.seeded.Warehouses["FG-WH1"]
	product := ts.seeded.Products["REF"]

	if rec := ts.do(t, "keeper", http.MethodPost, "/api/v1/inventory/documents", map[string]any{
		"docType": "RECEIPT", "businessDate": "2026-12-05", "factoryId": ts.seeded.FactoryID,
		"lines": []map[string]any{{
			"warehouseId": warehouse, "productId": product, "quantity": "500",
		}},
	}); rec.Code != http.StatusOK {
		t.Fatalf("receipt status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec := ts.do(t, "planner", http.MethodPut, "/api/v1/quality/parameters", map[string]any{
		"code": "POL", "name": "Polarisation", "uom": "PCT", "active": true,
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("a planner may not define quality parameters, got %d", rec.Code)
	}

	rec = ts.do(t, "masterdata", http.MethodPut, "/api/v1/quality/parameters", map[string]any{
		"code": "POL", "name": "Polarisation", "uom": "PCT", "active": true,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("save parameter status = %d, body = %s", rec.Code, rec.Body.String())
	}
	parameter := decode(t, rec)

	rec = ts.do(t, "masterdata", http.MethodPut, "/api/v1/quality/specs", map[string]any{
		"productId": product, "parameterId": parameter["id"], "lowerLimit": "99.500",
		"validFrom": "2026-10-01",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("save spec status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec = ts.do(t, "lab", http.MethodPost, "/api/v1/quality/samples", map[string]any{
		"productId": product, "factoryId": ts.seeded.FactoryID, "businessDate": "2026-12-06",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create sample status = %d, body = %s", rec.Code, rec.Body.String())
	}
	sample := decode(t, rec)
	if sample["status"] != "OPEN" {
		t.Errorf("a new sample is OPEN, got %v", sample["status"])
	}

	// A failing sheet fails the sample and blocks the material named on it.
	rec = ts.do(t, "lab", http.MethodPost,
		"/api/v1/quality/samples/"+sample["id"].(string)+"/results", map[string]any{
			"results":  []map[string]any{{"parameterId": parameter["id"], "value": "98.900"}},
			"complete": true, "holdWarehouse": warehouse, "holdQuantity": "120",
		})
	if rec.Code != http.StatusOK {
		t.Fatalf("record results status = %d, body = %s", rec.Code, rec.Body.String())
	}
	outcome := decode(t, rec)
	if outcome["verdict"] != "FAIL" {
		t.Fatalf("verdict = %v, want FAIL", outcome["verdict"])
	}
	if outcome["hold"] == nil {
		t.Fatal("a failed sample must block the material")
	}
	hold := outcome["hold"].(map[string]any)

	rec = ts.do(t, "keeper", http.MethodGet,
		"/api/v1/stock?warehouseId="+warehouse+"&productId="+product, nil)
	line := decode(t, rec)["value"].([]any)[0].(map[string]any)
	if line["holdQuantity"] != "120" || line["available"] != "380" {
		t.Errorf("held = %v available = %v, want 120 and 380", line["holdQuantity"], line["available"])
	}

	rec = ts.do(t, "keeper", http.MethodGet, "/api/v1/quality/holds?openOnly=true", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list holds status = %d", rec.Code)
	}
	if count := decode(t, rec)["count"]; count != float64(1) {
		t.Errorf("open holds = %v, want 1", count)
	}

	// Releasing needs the quality release permission, which the keeper lacks.
	rec = ts.do(t, "keeper", http.MethodPost,
		"/api/v1/quality/holds/"+hold["id"].(string)+"/release",
		map[string]any{"releasedOn": "2026-12-08", "reason": "retested"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("a keeper may not release a hold, got %d", rec.Code)
	}

	rec = ts.do(t, "lab", http.MethodPost,
		"/api/v1/quality/holds/"+hold["id"].(string)+"/release",
		map[string]any{"releasedOn": "2026-12-08", "reason": "retested within limits"},
		"If-Match", fmt.Sprintf(`"%v"`, hold["rowVersion"]))
	if rec.Code != http.StatusOK {
		t.Fatalf("release status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec = ts.do(t, "keeper", http.MethodGet,
		"/api/v1/stock?warehouseId="+warehouse+"&productId="+product, nil)
	line = decode(t, rec)["value"].([]any)[0].(map[string]any)
	if line["holdQuantity"] != "0" {
		t.Errorf("held after the release = %v, want 0", line["holdQuantity"])
	}
}

func TestMaintenanceIsApprovedOnlyByAnApprover(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "supervisor", http.MethodPut, "/api/v1/maintenance", map[string]any{
		"factoryId": ts.seeded.FactoryID, "startDate": "2027-01-10", "endDate": "2027-01-12",
		"description": "Mill roller change",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	window := decode(t, rec)
	if window["status"] != "PLANNED" {
		t.Errorf("a new window is PLANNED, got %v", window["status"])
	}

	window["status"] = "APPROVED"
	rec = ts.do(t, "supervisor", http.MethodPut, "/api/v1/maintenance", window)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("a supervisor may not approve an outage, got %d; body = %s", rec.Code, rec.Body.String())
	}

	rec = ts.do(t, "approver", http.MethodPut, "/api/v1/maintenance", window)
	if rec.Code != http.StatusOK {
		t.Fatalf("approve status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec = ts.do(t, "planner", http.MethodGet,
		"/api/v1/maintenance?factoryId="+ts.seeded.FactoryID+"&status=approved", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	if count := decode(t, rec)["count"]; count != float64(1) {
		t.Errorf("approved windows = %v, want 1", count)
	}
}

func TestExecutionEndpointsRefuseAnotherFactory(t *testing.T) {
	ts := newTestServer(t)

	for _, path := range []string{
		"/api/v1/production-orders?factoryId=" + ts.seeded.FactoryID,
		"/api/v1/inventory/documents?factoryId=" + ts.seeded.FactoryID,
		"/api/v1/quality/samples?factoryId=" + ts.seeded.FactoryID,
		"/api/v1/maintenance?factoryId=" + ts.seeded.FactoryID,
	} {
		rec := ts.do(t, "outsider", http.MethodGet, path, nil)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: status = %d, want 403", path, rec.Code)
		}
	}
}

// TestTheSignInListComesFromTheServer pins the reason the endpoint exists: the
// application used to keep its own copy of the demonstration accounts, and the
// copy drifted - the quality user was configured on the server and missing from
// the sign-in page, so the laboratory role could not be demonstrated at all.
func TestTheSignInListComesFromTheServer(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "", http.MethodGet, "/api/v1/auth/dev-users", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; the sign-in page has no token yet", rec.Code)
	}
	body := decode(t, rec)
	users := body["value"].([]any)
	if len(users) == 0 {
		t.Fatal("no accounts were listed")
	}

	names := map[string]bool{}
	for _, u := range users {
		entry := u.(map[string]any)
		names[entry["username"].(string)] = true
		if entry["displayName"] == "" {
			t.Errorf("%v has no display name", entry["username"])
		}
		if roles, ok := entry["roles"].([]any); !ok || len(roles) == 0 {
			t.Errorf("%v has no roles", entry["username"])
		}
	}
	// Every account the server can issue a token for must be offered.
	for _, want := range []string{"planner", "approver", "supervisor", "keeper", "lab", "masterdata"} {
		if !names[want] {
			t.Errorf("%q is configured on the server but not offered on the sign-in page", want)
		}
	}
}

func TestTheCertificateOfAnalysisOverHTTP(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "lab", http.MethodPost, "/api/v1/quality/samples", map[string]any{
		"productId": ts.seeded.Products["REF"], "factoryId": ts.seeded.FactoryID,
		"businessDate": "2026-12-05",
	})
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("create sample: %d %s", rec.Code, rec.Body.String())
	}
	sample := decode(t, rec)
	sampleID := sample["id"].(string)

	rec = ts.do(t, "lab", http.MethodGet, "/api/v1/quality/parameters", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("parameters: %d %s", rec.Code, rec.Body.String())
	}
	var polID string
	for _, p := range decode(t, rec)["value"].([]any) {
		if p.(map[string]any)["code"] == "POL" {
			polID = p.(map[string]any)["id"].(string)
		}
	}
	if polID == "" {
		t.Fatal("the seeded catalogue has no POL parameter")
	}

	// An open sample has nothing to certify.
	rec = ts.do(t, "lab", http.MethodGet, "/api/v1/quality/samples/"+sampleID+"/certificate", nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("an open sample: status = %d, want 400", rec.Code)
	}

	rec = ts.do(t, "lab", http.MethodPost, "/api/v1/quality/samples/"+sampleID+"/results",
		map[string]any{
			"results":  []map[string]any{{"parameterId": polID, "value": "99.9"}},
			"complete": true,
		})
	if rec.Code != http.StatusOK {
		t.Fatalf("results: %d %s", rec.Code, rec.Body.String())
	}

	rec = ts.do(t, "lab", http.MethodGet, "/api/v1/quality/samples/"+sampleID+"/certificate", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("certificate: %d %s", rec.Code, rec.Body.String())
	}
	cert := decode(t, rec)
	if cert["verdict"] != "PASS" {
		t.Errorf("verdict = %v, want PASS", cert["verdict"])
	}
	lines := cert["lines"].([]any)
	if len(lines) != 1 {
		t.Fatalf("one measurement, one line, got %d", len(lines))
	}
	line := lines[0].(map[string]any)
	if line["lowerLimit"] == nil {
		t.Error("the limit is what makes the result mean anything")
	}
	if _, isString := line["value"].(string); !isString {
		t.Errorf("value = %#v, want a decimal string", line["value"])
	}

	// The PDF is what goes with the consignment.
	rec = ts.do(t, "lab", http.MethodGet,
		"/api/v1/quality/samples/"+sampleID+"/certificate?format=pdf", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("pdf: %d %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("content type = %q, want application/pdf", ct)
	}
	if !strings.HasPrefix(rec.Body.String(), "%PDF-") {
		t.Error("the body is not a PDF")
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "COA-") {
		t.Errorf("the download must be named after the certificate, got %q", cd)
	}
}

func TestThePackagingBillOfMaterialsOverHTTP(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "masterdata", http.MethodGet, "/api/v1/master/packaging-bom", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, rec.Body.String())
	}
	if len(decode(t, rec)["value"].([]any)) == 0 {
		t.Fatal("the seeded scenario has a bill of materials")
	}

	// A component consumed in no quantity is not a component.
	packaging := ts.seeded.Packaging["PJUMBO"]
	rec = ts.do(t, "masterdata", http.MethodPut, "/api/v1/master/packaging-bom", map[string]any{
		"packagingId": packaging, "materialId": ts.seeded.Materials["PALLET"],
		"qtyPerPackage": "0", "active": true,
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("a zero quantity: status = %d, want 400", rec.Code)
	}

	rec = ts.do(t, "masterdata", http.MethodPut, "/api/v1/master/packaging-bom", map[string]any{
		"packagingId": packaging, "materialId": ts.seeded.Materials["PALLET"],
		"qtyPerPackage": "0.033333", "active": true,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("save: %d %s", rec.Code, rec.Body.String())
	}
	saved := decode(t, rec)
	if saved["qtyPerPackage"] != "0.033333" {
		t.Errorf("qtyPerPackage = %v; six decimals must survive the round trip",
			saved["qtyPerPackage"])
	}

	// The same material twice on one package is the one thing the key forbids.
	rec = ts.do(t, "masterdata", http.MethodPut, "/api/v1/master/packaging-bom", map[string]any{
		"packagingId": packaging, "materialId": ts.seeded.Materials["PALLET"],
		"qtyPerPackage": "0.05", "active": true,
	})
	if rec.Code != http.StatusConflict {
		t.Errorf("a duplicate: status = %d, want 409, body = %s", rec.Code, rec.Body.String())
	}

	rec = ts.do(t, "masterdata", http.MethodDelete,
		"/api/v1/master/packaging-bom/"+saved["id"].(string), nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("delete: status = %d, want 204", rec.Code)
	}

	// Reading is master data; writing needs the write permission.
	rec = ts.do(t, "planner", http.MethodPut, "/api/v1/master/packaging-bom", map[string]any{
		"packagingId": packaging, "materialId": ts.seeded.Materials["PALLET"],
		"qtyPerPackage": "0.05",
	})
	if rec.Code != http.StatusForbidden {
		t.Errorf("a planner writing the bill of materials: status = %d, want 403", rec.Code)
	}
}
