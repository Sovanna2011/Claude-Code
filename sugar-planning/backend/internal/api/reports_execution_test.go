package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// decodeInto reads a response into a typed value, for the reports whose shape
// is a table rather than an entity.
func decodeInto(t *testing.T, rec *httptest.ResponseRecorder, into any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), into); err != nil {
		t.Fatalf("decode response (%s): %v", rec.Body.String(), err)
	}
}

// lineID resolves a production line by its code, which is what a test can name.
func lineID(t *testing.T, ts *testServer, code string) string {
	t.Helper()
	rec := ts.do(t, "planner", http.MethodGet,
		"/api/v1/master/production-lines?parentId="+ts.seeded.FactoryID+"&top=100", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("lines: %d %s", rec.Code, rec.Body.String())
	}
	for _, item := range decode(t, rec)["value"].([]any) {
		line := item.(map[string]any)
		if line["code"] == code {
			return line["id"].(string)
		}
	}
	t.Fatalf("no line %q is seeded", code)
	return ""
}

// The reports that reconcile the process. What is asserted here is arithmetic
// somebody could redo on paper: a report that merely returns 200 tells nobody
// whether the figures on it are right.

// runReport fetches a report as JSON and returns its rows, totals and notes.
type table struct {
	Columns []struct {
		Header string `json:"header"`
	} `json:"columns"`
	Rows   [][]string `json:"rows"`
	Totals []string   `json:"totals"`
	Notes  []string   `json:"notes"`
	Title  string     `json:"title"`
}

func runReport(t *testing.T, ts *testServer, user, code, query string) table {
	t.Helper()
	rec := ts.do(t, user, http.MethodGet, "/api/v1/reports/"+code+"?"+query, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("report %s = %d: %s", code, rec.Code, rec.Body.String())
	}
	var out table
	decodeInto(t, rec, &out)
	return out
}

// column finds a column by its heading, so a test does not break when a column
// is inserted beside the one it cares about.
func (tb table) column(t *testing.T, header string) int {
	t.Helper()
	for i, c := range tb.Columns {
		if c.Header == header {
			return i
		}
	}
	t.Fatalf("the report has no %q column; it has %v", header, tb.headers())
	return -1
}

func (tb table) headers() []string {
	out := make([]string, len(tb.Columns))
	for i, c := range tb.Columns {
		out[i] = c.Header
	}
	return out
}

func (tb table) note(fragment string) bool {
	for _, n := range tb.Notes {
		if strings.Contains(n, fragment) {
			return true
		}
	}
	return false
}

func TestTheRecoveryReportReconcilesCaneAgainstRawSugar(t *testing.T) {
	ts := newTestServer(t)
	tb := runReport(t, ts, "planner", "recovery-mass-balance",
		"versionId="+ts.versionID()+"&seasonId="+ts.seeded.SeasonID+
			"&from=2026-12-01&to=2026-12-03")

	if len(tb.Rows) != 3 {
		t.Fatalf("three days asked for, %d rows returned: %v", len(tb.Rows), tb.Rows)
	}

	crushed := tb.column(t, "Cane crushed (t)")
	expected := tb.column(t, "Expected raw (t)")
	produced := tb.column(t, "Raw produced (t)")
	recovery := tb.column(t, "Recovery %")
	difference := tb.column(t, "Balance difference (t)")

	// The seeded actuals run at a recovery that drifts around the 11 %
	// assumption, which is the whole reason the report exists.
	first := tb.Rows[0]
	if first[crushed] != "13766.423" || first[produced] != "1473.007" {
		t.Fatalf("the first day's actuals changed: %v", first)
	}
	// 13766.423 x 11 % = 1514.30653, rounded to three places.
	if first[expected] != "1514.307" {
		t.Errorf("expected raw = %s, want 1514.307 (13766.423 x 11 %%)", first[expected])
	}
	// 1473.007 / 13766.423 = 10.700 %.
	if first[recovery] != "10.700" {
		t.Errorf("recovery = %s, want 10.700", first[recovery])
	}
	// 1514.307 - 1473.007 = 41.300.
	if first[difference] != "41.300" {
		t.Errorf("balance difference = %s, want 41.300", first[difference])
	}

	// 41.3 t on 1514.3 t expected is 2.7 %, well past the seeded 0.5 %
	// tolerance, so the day is an exception however healthy the recovery looks.
	verdict := tb.column(t, "Verdict")
	if first[verdict] != "WARNING" {
		t.Errorf("a day outside the mass balance tolerance must not read %s", first[verdict])
	}
	if !tb.note("3 days are outside the mass balance tolerance") {
		t.Errorf("the exceptions must be counted in the notes: %v", tb.Notes)
	}

	// The plan container holds no raw sugar production row at all, so a report
	// that read it would print a column of zeroes. It must say which figures
	// these are.
	if !tb.note("actuals from ACTUAL") {
		t.Errorf("the report must name the series it reconciled: %v", tb.Notes)
	}

	if tb.Totals[crushed] != "39788.321" {
		t.Errorf("total crushed = %s, want 39788.321", tb.Totals[crushed])
	}
}

func TestThePackingReportConvertsTonsToPackages(t *testing.T) {
	ts := newTestServer(t)
	tb := runReport(t, ts, "planner", "packing",
		"versionId="+ts.versionID()+"&seasonId="+ts.seeded.SeasonID+
			"&from=2026-12-01&to=2026-12-03")

	product := tb.column(t, "Product")
	pack := tb.column(t, "Package")
	series := tb.column(t, "Series")
	qty := tb.column(t, "Quantity (t)")
	count := tb.column(t, "Packages")

	// Both containers are read, so the plan and the actual for a package sit
	// beside each other. A report that could only ever say PLAN would not need
	// the column.
	seen := map[string]bool{}
	for _, row := range tb.Rows {
		seen[row[series]] = true
	}
	if !seen["PLAN"] || !seen["ACTUAL"] {
		t.Errorf("the report must cover both series, saw %v", seen)
	}

	// 2336.496 t of refined sugar in 50 kg bags is 46 729.92 bags, and the
	// column is the exact conversion rather than a purchase requirement, so it
	// truncates rather than rounding up.
	found := false
	for _, row := range tb.Rows {
		if !strings.HasPrefix(row[product], "REF") || row[series] != "PLAN" {
			continue
		}
		found = true
		if !strings.HasPrefix(row[pack], "P50KG") {
			t.Errorf("package = %s", row[pack])
		}
		if row[qty] != "2336.496" || row[count] != "46730" {
			t.Errorf("%s t of refined sugar in 50 kg bags = %s bags, want 2336.496 t and 46730",
				row[qty], row[count])
		}
	}
	if !found {
		t.Fatalf("no planned refined sugar row: %v", tb.Rows)
	}
}

func TestTheRemeltReportShowsYieldAndSkipsUnrefinedProduct(t *testing.T) {
	ts := newTestServer(t)
	tb := runReport(t, ts, "planner", "remelt-refining",
		"versionId="+ts.versionID()+"&seasonId="+ts.seeded.SeasonID+
			"&from=2026-12-01&to=2026-12-01")

	product := tb.column(t, "Product")
	input := tb.column(t, "Remelt input (t)")
	output := tb.column(t, "Output (t)")
	yield := tb.column(t, "Yield %")

	if len(tb.Rows) == 0 {
		t.Fatal("a refining day with no rows")
	}
	for _, row := range tb.Rows {
		// Raw sugar is not refined from anything, so it has no place here; a row
		// for it would carry a remelt input of zero and an unanswerable yield.
		if strings.HasPrefix(row[product], "RAW") {
			t.Errorf("raw sugar must not appear in the refining report: %v", row)
		}
		if row[input] == "0.000" && row[yield] != "" {
			t.Errorf("a day with no remelt input must show no yield, got %q", row[yield])
		}
	}

	// The seeded remelt factor is 1.05, so a tonne in yields 1/1.05 = 95.238 %.
	if tb.Totals[yield] != "95.238" {
		t.Errorf("total yield = %s, want 95.238 (a remelt factor of 1.05)", tb.Totals[yield])
	}
	if tb.Totals[output] == "0.000" || tb.Totals[input] == "0.000" {
		t.Errorf("the totals are empty: %v", tb.Totals)
	}
}

func TestTheOrderVarianceReportNamesUnexplainedShortfalls(t *testing.T) {
	ts := newTestServer(t)

	// An order confirmed short of its plan, with no reason recorded.
	rec := ts.do(t, "supervisor", http.MethodPost, "/api/v1/production-orders", map[string]any{
		"factoryId": ts.seeded.FactoryID, "businessDate": "2026-12-05",
		"productId": ts.seeded.Products["REF"], "plannedQty": "300",
		"versionId": ts.versionID(),
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create order: %d %s", rec.Code, rec.Body.String())
	}
	order := decode(t, rec)
	id := order["id"].(string)

	rec = ts.do(t, "supervisor", http.MethodPost,
		"/api/v1/production-orders/"+id+"/action", map[string]any{"action": "RELEASE"},
		"If-Match", rec.Header().Get("ETag"))
	if rec.Code != http.StatusOK {
		t.Fatalf("release: %d %s", rec.Code, rec.Body.String())
	}

	rec = ts.do(t, "supervisor", http.MethodPost,
		"/api/v1/production-orders/"+id+"/confirm", map[string]any{
			"businessDate": "2026-12-05", "yieldQty": "240",
			"warehouseId": ts.seeded.Warehouses["FG-WH1"],
		})
	if rec.Code != http.StatusOK {
		t.Fatalf("confirm: %d %s", rec.Code, rec.Body.String())
	}

	// An order 20 % short cannot be closed silently, which is the rule this
	// report reads back: the reason on the row is one somebody had to give.
	rec = ts.do(t, "supervisor", http.MethodGet, "/api/v1/production-orders/"+id, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("read order: %d %s", rec.Code, rec.Body.String())
	}
	silent := ts.do(t, "supervisor", http.MethodPost,
		"/api/v1/production-orders/"+id+"/action",
		map[string]any{"action": "TECHNICALLY_CLOSE"},
		"If-Match", rec.Header().Get("ETag"))
	if silent.Code != http.StatusBadRequest {
		t.Fatalf("closing 60 t short with no reason: %d %s", silent.Code, silent.Body.String())
	}

	// Nor may the supervisor authorise their own shortfall: a 20 % variance is
	// an approver's decision.
	unauthorised := ts.do(t, "supervisor", http.MethodPost,
		"/api/v1/production-orders/"+id+"/action",
		map[string]any{"action": "TECHNICALLY_CLOSE",
			"varianceReason": "VAR-CANE"},
		"If-Match", rec.Header().Get("ETag"))
	if unauthorised.Code != http.StatusForbidden {
		t.Fatalf("a supervisor closing their own 20 %% shortfall: %d %s",
			unauthorised.Code, unauthorised.Body.String())
	}

	closed := ts.do(t, "manager", http.MethodPost,
		"/api/v1/production-orders/"+id+"/action",
		map[string]any{"action": "TECHNICALLY_CLOSE",
			"varianceReason": "VAR-CANE"},
		"If-Match", rec.Header().Get("ETag"))
	if closed.Code != http.StatusOK {
		t.Fatalf("close with a reason: %d %s", closed.Code, closed.Body.String())
	}

	tb := runReport(t, ts, "planner", "order-variance",
		"versionId="+ts.versionID()+"&seasonId="+ts.seeded.SeasonID+
			"&from=2026-12-05&to=2026-12-05")

	if len(tb.Rows) != 1 {
		t.Fatalf("one order confirmed, %d rows: %v", len(tb.Rows), tb.Rows)
	}
	row := tb.Rows[0]
	planned := tb.column(t, "Planned (t)")
	confirmed := tb.column(t, "Confirmed (t)")
	variance := tb.column(t, "Variance (t)")
	achievement := tb.column(t, "Achievement %")

	if row[planned] != "300.000" || row[confirmed] != "240.000" {
		t.Fatalf("planned/confirmed = %s/%s", row[planned], row[confirmed])
	}
	if row[variance] != "-60.000" {
		t.Errorf("variance = %s, want -60.000", row[variance])
	}
	if row[achievement] != "80.000" {
		t.Errorf("achievement = %s, want 80.000", row[achievement])
	}

	if row[tb.column(t, "Variance reason")] != "VAR-CANE" {
		t.Errorf("the reason the order was allowed to close must be on the row, got %q",
			row[tb.column(t, "Variance reason")])
	}
	if !tb.note("1 order closed with an authorised variance reason") {
		t.Errorf("the authorised exception must be counted: %v", tb.Notes)
	}
	if !tb.note("0 orders are still open") {
		t.Errorf("the notes must say nothing is still running: %v", tb.Notes)
	}
}

func TestTheQualityReportCarriesTheLimitsAndTheHold(t *testing.T) {
	ts := newTestServer(t)
	product := ts.seeded.Products["REF"]

	// A hold blocks stock that exists, so there has to be some.
	rec := ts.do(t, "keeper", http.MethodPost, "/api/v1/inventory/documents", map[string]any{
		"docType": "RECEIPT", "businessDate": "2026-12-05", "factoryId": ts.seeded.FactoryID,
		"lines": []map[string]any{{
			"warehouseId": ts.seeded.Warehouses["FG-WH1"], "productId": product, "quantity": "500",
		}},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("receipt: %d %s", rec.Code, rec.Body.String())
	}

	// The seeded specification for refined sugar puts polarisation at 99.700
	// degZ or better. Using it rather than inventing one keeps the test on the
	// same data an operator would see.
	rec = ts.do(t, "lab", http.MethodGet, "/api/v1/quality/parameters", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("parameters: %d %s", rec.Code, rec.Body.String())
	}
	var parameterID string
	for _, item := range decode(t, rec)["value"].([]any) {
		if p := item.(map[string]any); p["code"] == "POL" {
			parameterID = p["id"].(string)
		}
	}
	if parameterID == "" {
		t.Fatal("the scenario seeds a polarisation parameter")
	}

	rec = ts.do(t, "lab", http.MethodPost, "/api/v1/quality/samples", map[string]any{
		"productId": product, "factoryId": ts.seeded.FactoryID, "businessDate": "2026-12-06",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("sample: %d %s", rec.Code, rec.Body.String())
	}
	sample := decode(t, rec)

	rec = ts.do(t, "lab", http.MethodPost,
		"/api/v1/quality/samples/"+sample["id"].(string)+"/results", map[string]any{
			"results":  []map[string]any{{"parameterId": parameterID, "value": "98.900"}},
			"complete": true, "holdWarehouse": ts.seeded.Warehouses["FG-WH1"],
			"holdQuantity": "120",
		})
	if rec.Code != http.StatusOK {
		t.Fatalf("results: %d %s", rec.Code, rec.Body.String())
	}

	tb := runReport(t, ts, "planner", "quality-results",
		"versionId="+ts.versionID()+"&seasonId="+ts.seeded.SeasonID+
			"&from=2026-12-06&to=2026-12-06")

	if len(tb.Rows) != 1 {
		t.Fatalf("one result recorded, %d rows: %v", len(tb.Rows), tb.Rows)
	}
	row := tb.Rows[0]
	if row[tb.column(t, "Value")] != "98.900" {
		t.Errorf("value = %s", row[tb.column(t, "Value")])
	}
	// The limits are copied from the result, which is what it was judged
	// against. A specification changed later must not silently re-judge it.
	if row[tb.column(t, "Lower")] != "99.700" {
		t.Errorf("lower limit = %s, want the seeded 99.700", row[tb.column(t, "Lower")])
	}
	if row[tb.column(t, "Upper")] != "" {
		t.Errorf("polarisation has no upper limit, got %q", row[tb.column(t, "Upper")])
	}
	if row[tb.column(t, "Result")] != "FAIL" {
		t.Errorf("98.900 against a lower limit of 99.700 is a FAIL, got %s",
			row[tb.column(t, "Result")])
	}
	if hold := row[tb.column(t, "Hold")]; !strings.HasPrefix(hold, "Held since") {
		t.Errorf("the hold the failure caused must be on the row, got %q", hold)
	}
	if !tb.note("1 result outside specification") || !tb.note("1 hold still open") {
		t.Errorf("the notes must count the failures and the open holds: %v", tb.Notes)
	}
}

func TestTheDowntimeReportPricesEachStoppage(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "supervisor", http.MethodPost, "/api/v1/downtime", map[string]any{
		"factoryId": ts.seeded.FactoryID, "lineId": lineID(t, ts, "MILL-1"),
		"businessDate": "2026-12-04", "startAt": "2026-12-04T02:00:00Z",
		"endAt": "2026-12-04T06:00:00Z", "reasonCode": "DT-BOILER",
		"rootCause": "Tube leak", "correctiveAction": "Tube replaced",
	})
	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("downtime: %d %s", rec.Code, rec.Body.String())
	}

	tb := runReport(t, ts, "planner", "downtime",
		"versionId="+ts.versionID()+"&seasonId="+ts.seeded.SeasonID+
			"&from=2026-12-04&to=2026-12-04")

	if len(tb.Rows) != 1 {
		t.Fatalf("one stoppage recorded, %d rows: %v", len(tb.Rows), tb.Rows)
	}
	row := tb.Rows[0]
	if row[tb.column(t, "Hours")] != "4.00" {
		t.Errorf("hours = %s, want 4.00", row[tb.column(t, "Hours")])
	}
	// The mill is rated at 700 t/h, so four hours down cost 2 800 t.
	if row[tb.column(t, "Lost tons")] != "2800.000" {
		t.Errorf("lost tons = %s, want 2800.000 (4 h at 700 t/h)",
			row[tb.column(t, "Lost tons")])
	}
	if !strings.Contains(row[tb.column(t, "Reason")], "DT-BOILER") {
		t.Errorf("reason = %s", row[tb.column(t, "Reason")])
	}
	if row[tb.column(t, "Type")] != "Unplanned" {
		t.Errorf("type = %s, want Unplanned", row[tb.column(t, "Type")])
	}
	if !tb.note("Lost tons are an estimate") {
		t.Errorf("an estimated figure must be labelled as one: %v", tb.Notes)
	}
}
