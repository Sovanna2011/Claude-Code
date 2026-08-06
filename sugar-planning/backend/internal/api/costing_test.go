package api_test

import (
	"net/http"
	"strings"
	"testing"
)

// The costing endpoints over real HTTP.

func TestCostingARunOverHTTP(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "controller", http.MethodPost, "/api/v1/costing/runs", map[string]any{
		"seasonId": ts.seeded.SeasonID, "from": "2026-12-01", "to": "2026-12-14",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	result := decode(t, rec)

	lines, ok := result["lines"].([]any)
	if !ok || len(lines) == 0 {
		t.Fatalf("the run produced no lines: %s", rec.Body.String())
	}
	// Money crosses the wire as a string, like every other exact figure.
	first := lines[0].(map[string]any)
	if _, isString := first["plannedCost"].(string); !isString {
		t.Errorf("plannedCost = %#v, want a decimal string", first["plannedCost"])
	}
	if first["driverUnit"] == "" {
		t.Error("a line must name the unit its driver is measured in")
	}

	totals := result["totals"].(map[string]any)
	for _, key := range []string{"plannedCost", "actualCost", "rateVariance",
		"usageVariance", "totalVariance", "actualUnitCost", "fixedCost", "variableCost"} {
		if totals[key] == nil {
			t.Errorf("the totals must carry %s", key)
		}
	}

	// The drivers are reported so a reader can check the arithmetic without
	// opening another screen.
	planned := result["plannedDrivers"].(map[string]any)
	if planned["caneTons"] == nil || planned["runHours"] == nil {
		t.Errorf("the run must report the drivers it used: %#v", planned)
	}
	if result["currency"] != "USD" {
		t.Errorf("currency = %v, want the company's USD", result["currency"])
	}
	if result["saved"] != false {
		t.Error("a run without save must not have been stored")
	}
}

func TestCostFiguresAreRefusedWithoutTheCostPermission(t *testing.T) {
	ts := newTestServer(t)

	for _, user := range []string{"planner", "supervisor", "keeper"} {
		rec := ts.do(t, user, http.MethodPost, "/api/v1/costing/runs", map[string]any{
			"seasonId": ts.seeded.SeasonID,
		})
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: status = %d, want 403", user, rec.Code)
		}
		rec = ts.do(t, user, http.MethodGet, "/api/v1/costing/rates", nil)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s reading rates: status = %d, want 403", user, rec.Code)
		}
	}

	// The auditor and the executive may read the cost but not set a rate.
	for _, user := range []string{"auditor", "executive"} {
		rec := ts.do(t, user, http.MethodGet, "/api/v1/costing/rates", nil)
		if rec.Code != http.StatusOK {
			t.Errorf("%s reading rates: status = %d, want 200", user, rec.Code)
		}
		rec = ts.do(t, user, http.MethodPut, "/api/v1/costing/rates", map[string]any{
			"elementId": "3f1d1a1e-0000-4000-8000-000000000000",
			"factoryId": ts.seeded.FactoryID, "rateType": "STANDARD",
			"rate": "1", "currency": "USD", "validFrom": "2026-12-01",
		})
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s setting a rate: status = %d, want 403", user, rec.Code)
		}
	}
}

func TestASavedRunIsReadBackWithItsLines(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "controller", http.MethodPost, "/api/v1/costing/runs", map[string]any{
		"seasonId": ts.seeded.SeasonID, "from": "2026-12-01", "to": "2026-12-14",
		"save": true, "code": "HTTP-1", "note": "First fortnight",
	}, "Idempotency-Key", "cost-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	result := decode(t, rec)
	if result["saved"] != true {
		t.Fatalf("the run was not saved: %s", rec.Body.String())
	}
	run := result["run"].(map[string]any)
	id := run["id"].(string)

	// The retry replays rather than costing and storing again.
	again := ts.do(t, "controller", http.MethodPost, "/api/v1/costing/runs", map[string]any{
		"seasonId": ts.seeded.SeasonID, "from": "2026-12-01", "to": "2026-12-14",
		"save": true, "code": "HTTP-1",
	}, "Idempotency-Key", "cost-1")
	if again.Header().Get("Idempotent-Replay") != "true" {
		t.Error("the retry must be marked as a replay")
	}

	detail := ts.do(t, "controller", http.MethodGet, "/api/v1/costing/runs/"+id, nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("get run status = %d, body = %s", detail.Code, detail.Body.String())
	}
	stored := decode(t, detail)
	if len(stored["lines"].([]any)) == 0 {
		t.Error("a saved run must keep its lines")
	}
	if stored["note"] != "First fortnight" {
		t.Errorf("note = %v", stored["note"])
	}

	list := ts.do(t, "controller", http.MethodGet,
		"/api/v1/costing/runs?seasonId="+ts.seeded.SeasonID, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list runs status = %d", list.Code)
	}
	if count := decode(t, list)["count"]; count != float64(1) {
		t.Errorf("saved runs = %v, want 1", count)
	}
}

func TestARateIsCorrectedRatherThanDuplicated(t *testing.T) {
	ts := newTestServer(t)

	elements := ts.do(t, "controller", http.MethodGet, "/api/v1/costing/elements?$top=100", nil)
	if elements.Code != http.StatusOK {
		t.Fatalf("list elements status = %d, body = %s", elements.Code, elements.Body.String())
	}
	var fuelID string
	for _, e := range decode(t, elements)["value"].([]any) {
		entry := e.(map[string]any)
		if entry["code"] == "FUEL" {
			fuelID = entry["id"].(string)
		}
	}
	if fuelID == "" {
		t.Fatal("the seeded cost structure has no FUEL element")
	}

	before := ts.do(t, "controller", http.MethodGet,
		"/api/v1/costing/rates?factoryId="+ts.seeded.FactoryID+"&elementId="+fuelID, nil)
	countBefore := decode(t, before)["count"]

	rec := ts.do(t, "controller", http.MethodPut, "/api/v1/costing/rates", map[string]any{
		"elementId": fuelID, "factoryId": ts.seeded.FactoryID, "rateType": "STANDARD",
		"rate": "148.000000", "currency": "USD", "validFrom": "2026-12-01",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("save rate status = %d, body = %s", rec.Code, rec.Body.String())
	}

	after := ts.do(t, "controller", http.MethodGet,
		"/api/v1/costing/rates?factoryId="+ts.seeded.FactoryID+"&elementId="+fuelID, nil)
	if decode(t, after)["count"] != countBefore {
		t.Errorf("rates = %v, was %v; the same start date must correct rather than duplicate",
			decode(t, after)["count"], countBefore)
	}

	// A later start date is a price change, not a correction.
	if rec := ts.do(t, "controller", http.MethodPut, "/api/v1/costing/rates", map[string]any{
		"elementId": fuelID, "factoryId": ts.seeded.FactoryID, "rateType": "STANDARD",
		"rate": "160.000000", "currency": "USD", "validFrom": "2027-02-01",
	}); rec.Code != http.StatusOK {
		t.Fatalf("save the price change: %s", rec.Body.String())
	}
	rise := ts.do(t, "controller", http.MethodGet,
		"/api/v1/costing/rates?factoryId="+ts.seeded.FactoryID+"&elementId="+fuelID, nil)
	if decode(t, rise)["count"] == countBefore {
		t.Error("a later start date must be a new rate")
	}
}

func TestAnInvalidRateNamesEveryOffendingField(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "controller", http.MethodPut, "/api/v1/costing/rates", map[string]any{
		"factoryId": ts.seeded.FactoryID, "rateType": "GUESS",
		"rate": "-5", "currency": "DOLLARS", "validFrom": "not-a-date",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/problem+json") {
		t.Errorf("content type = %q, want a problem document", ct)
	}
	errs := decode(t, rec)["errors"].([]any)
	if len(errs) < 5 {
		t.Errorf("field errors = %d, want one per offending field: %s", len(errs), rec.Body.String())
	}
}
