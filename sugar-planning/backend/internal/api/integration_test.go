package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

// The interface endpoints over real HTTP: what the gate terminal and the
// laboratory system actually talk to.

func TestWeighbridgeTicketsOverHTTP(t *testing.T) {
	ts := newTestServer(t)

	// The demonstration data already carries a fortnight of recorded cane, so
	// what is asserted here is the change the gate made, not the total.
	before := caneRow(t, ts, "2026-12-05")

	body := map[string]any{
		"tickets": []map[string]any{{
			"ticketNo": "WB-8801", "factoryId": ts.seeded.FactoryID,
			"businessDate": "2026-12-05", "vehicleNo": "PP-2245",
			"grossKg": "42000", "tareKg": "14000", "rejectedKg": "0",
		}, {
			"ticketNo": "WB-8802", "factoryId": ts.seeded.FactoryID,
			"businessDate": "2026-12-05", "growerCode": "G-118",
			"grossKg": "38500", "tareKg": "13500", "rejectedKg": "1500",
			"reasonCode": "BURNT",
		}},
	}

	rec := ts.do(t, "interface", http.MethodPost, "/api/v1/integration/weighbridge",
		body, "Idempotency-Key", "gate-batch-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	result := decode(t, rec)
	if result["accepted"] != float64(2) {
		t.Errorf("accepted = %v, want 2", result["accepted"])
	}

	days := result["days"].([]any)
	if len(days) != 1 {
		t.Fatalf("two tickets on one day are one row, got %d", len(days))
	}
	day := days[0].(map[string]any)
	// (42000-14000) + (38500-13500) = 53.000 t delivered, 1.500 t refused.
	if day["caneDelivered"] != "53" && day["caneDelivered"] != "53.000" {
		t.Errorf("caneDelivered = %v, want 53 t", day["caneDelivered"])
	}
	if day["caneAccepted"] != "51.5" && day["caneAccepted"] != "51.500" {
		t.Errorf("caneAccepted = %v, want 51.5 t", day["caneAccepted"])
	}

	// The terminal lost the network and resent the batch. The lorries must not
	// be weighed twice.
	rec = ts.do(t, "interface", http.MethodPost, "/api/v1/integration/weighbridge",
		body, "Idempotency-Key", "gate-batch-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("replay status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Idempotent-Replay") != "true" {
		t.Error("a retried batch must be answered from the first response")
	}

	// The replayed batch must not have added the lorries a second time.
	after := caneRow(t, ts, "2026-12-05")
	if got, want := sub(t, after["caneAccepted"], before["caneAccepted"]), "51.5"; got != want {
		t.Errorf("accepted cane grew by %s, want %s - the replay must not count twice", got, want)
	}
	if got, want := sub(t, after["caneRejected"], before["caneRejected"]), "1.5"; got != want {
		t.Errorf("rejected cane grew by %s, want %s", got, want)
	}
	// The mill reports what it crushed; a gate reading is not evidence about it.
	if after["caneCrushed"] != before["caneCrushed"] {
		t.Errorf("crushed cane changed from %v to %v, and the gate cannot know it",
			before["caneCrushed"], after["caneCrushed"])
	}
}

// caneRow reads the one actual cane row for a date.
func caneRow(t *testing.T, ts *testServer, date string) map[string]any {
	t.Helper()
	rec := ts.do(t, "supervisor", http.MethodGet,
		"/api/v1/versions/"+ts.seeded.ActualID+"/cane?from="+date+"&to="+date+"&series=ACTUAL", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("read cane: %d %s", rec.Code, rec.Body.String())
	}
	rows := decode(t, rec)["value"].([]any)
	if len(rows) != 1 {
		t.Fatalf("want one actual row for %s, got %d", date, len(rows))
	}
	return rows[0].(map[string]any)
}

// sub subtracts two decimal strings exactly, which is the only honest way to
// compare tonnages that crossed the wire as strings.
func sub(t *testing.T, after, before any) string {
	t.Helper()
	a, err := decimal.NewFromString(fmt.Sprint(after))
	if err != nil {
		t.Fatalf("parse %v: %v", after, err)
	}
	b, err := decimal.NewFromString(fmt.Sprint(before))
	if err != nil {
		t.Fatalf("parse %v: %v", before, err)
	}
	return a.Sub(b).String()
}

func TestABadTicketIsRefusedWithTheRowNamed(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "interface", http.MethodPost, "/api/v1/integration/weighbridge", map[string]any{
		"tickets": []map[string]any{{
			"ticketNo": "WB-9001", "factoryId": ts.seeded.FactoryID,
			"businessDate": "2026-12-05",
			// A tare heavier than the gross means the load weighs nothing.
			"grossKg": "12000", "tareKg": "14000", "rejectedKg": "0",
		}},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
		t.Errorf("content type = %q, want a problem document", ct)
	}
}

func TestOnlyTheMachineAccountMayFeedTheInterfaces(t *testing.T) {
	ts := newTestServer(t)

	for _, user := range []string{"planner", "keeper", "lab", "controller"} {
		rec := ts.do(t, user, http.MethodPost, "/api/v1/integration/weighbridge", map[string]any{
			"tickets": []map[string]any{{
				"ticketNo": "WB-1", "factoryId": ts.seeded.FactoryID,
				"businessDate": "2026-12-05", "grossKg": "42000", "tareKg": "14000",
			}},
		})
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s posting gate tickets: status = %d, want 403", user, rec.Code)
		}
		rec = ts.do(t, user, http.MethodGet, "/api/v1/integration/events", nil)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s reading the outbox: status = %d, want 403", user, rec.Code)
		}
	}
}

func TestTheOutboxAndTheDispatcherOverHTTP(t *testing.T) {
	ts := newTestServer(t)

	// Something worth publishing: a goods receipt.
	rec := ts.do(t, "keeper", http.MethodPost, "/api/v1/inventory/documents", map[string]any{
		"docType": "RECEIPT", "businessDate": "2026-12-05", "factoryId": ts.seeded.FactoryID,
		"lines": []map[string]any{{
			"warehouseId": ts.seeded.Warehouses["FG-WH1"],
			"productId":   ts.seeded.Products["REF"], "quantity": "250",
		}},
	})
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("post receipt: %d %s", rec.Code, rec.Body.String())
	}

	rec = ts.do(t, "admin", http.MethodGet, "/api/v1/integration/events?unpublished=true", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list events: %d %s", rec.Code, rec.Body.String())
	}
	page := decode(t, rec)
	events := page["value"].([]any)
	if len(events) == 0 {
		t.Fatal("posting stock must have written an event")
	}
	event := events[0].(map[string]any)
	if event["topic"] != "stock.posted" {
		t.Errorf("topic = %v, want stock.posted", event["topic"])
	}
	if event["id"] == nil || event["payload"] == nil {
		t.Errorf("an event needs an id to deduplicate on and a payload: %#v", event)
	}

	// The default publisher writes to the application log, so a dispatch pass
	// delivers rather than failing.
	rec = ts.do(t, "admin", http.MethodPost, "/api/v1/integration/dispatch", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("dispatch: %d %s", rec.Code, rec.Body.String())
	}
	result := decode(t, rec)
	if result["published"].(float64) < 1 {
		t.Errorf("the pass should have published something: %#v", result)
	}

	rec = ts.do(t, "admin", http.MethodGet, "/api/v1/integration/events?unpublished=true", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list events again: %d %s", rec.Code, rec.Body.String())
	}
	if left := decode(t, rec)["value"].([]any); len(left) != 0 {
		t.Errorf("nothing should still be waiting, got %d", len(left))
	}
}

func TestLabResultsOverHTTP(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, "lab", http.MethodPost, "/api/v1/quality/samples", map[string]any{
		"productId": ts.seeded.Products["REF"], "factoryId": ts.seeded.FactoryID,
		"businessDate": "2026-12-05",
	})
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("create sample: %d %s", rec.Code, rec.Body.String())
	}
	sampleNo := decode(t, rec)["sampleNo"].(string)

	rec = ts.do(t, "interface", http.MethodPost, "/api/v1/integration/lab-results", map[string]any{
		"sampleNo": sampleNo, "instrument": "POLARIMETER-2", "complete": true,
		"readings": []map[string]any{{"parameterCode": "POL", "value": "90"}},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("lab results: %d %s", rec.Code, rec.Body.String())
	}
	outcome := decode(t, rec)
	if outcome["verdict"] != "FAIL" {
		t.Errorf("verdict = %v, want FAIL for a pol of 90 on refined sugar", outcome["verdict"])
	}

	// A number nobody printed judges nothing.
	rec = ts.do(t, "interface", http.MethodPost, "/api/v1/integration/lab-results", map[string]any{
		"sampleNo": "QS-F1-2026-99999",
		"readings": []map[string]any{{"parameterCode": "POL", "value": "99"}},
	})
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404, body = %s", rec.Code, rec.Body.String())
	}
}
