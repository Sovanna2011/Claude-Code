package api_test

import (
	"net/http"
	"strings"
	"testing"
)

// Metrics. The failure that matters here is not a wrong count: it is a label
// that grows a series per request, which is the standard way to take down a
// Prometheus server.

func TestMetricsAreLabelledByRouteNotByPath(t *testing.T) {
	ts := newTestServer(t)

	// Three requests for three different seasons. One route, three ids.
	for _, id := range []string{ts.seeded.SeasonID, ts.seeded.SeasonID, "not-a-season"} {
		ts.do(t, "planner", http.MethodGet, "/api/v1/seasons/"+id, nil)
	}

	rec := ts.do(t, "", http.MethodGet, "/metrics", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("metrics: %d", rec.Code)
	}
	body := rec.Body.String()

	if !strings.Contains(body, `route="GET /api/v1/seasons/{id}"`) {
		t.Fatalf("requests must be counted against the route pattern:\n%s", body)
	}
	// The id must not appear anywhere in the exposition, or every season a
	// planner opens becomes a new time series.
	if strings.Contains(body, ts.seeded.SeasonID) {
		t.Error("a record id must never become a metric label")
	}
	if strings.Contains(body, "not-a-season") {
		t.Error("a path from the client must never become a metric label")
	}

	// Two found and one not found, on the same route.
	if !strings.Contains(body, `route="GET /api/v1/seasons/{id}",method="GET",status="200"} 2`) {
		t.Errorf("two successful reads expected:\n%s", extract(body, "seasons/{id}"))
	}
	if !strings.Contains(body, `route="GET /api/v1/seasons/{id}",method="GET",status="404"} 1`) {
		t.Errorf("one miss expected:\n%s", extract(body, "seasons/{id}"))
	}
}

func TestMetricsCarryLatencyAndUptime(t *testing.T) {
	ts := newTestServer(t)
	ts.do(t, "planner", http.MethodGet, "/api/v1/seasons", nil)

	body := ts.do(t, "", http.MethodGet, "/metrics", nil).Body.String()

	for _, want := range []string{
		"# TYPE sugarplan_http_requests_total counter",
		"# TYPE sugarplan_http_request_duration_seconds histogram",
		"# TYPE sugarplan_uptime_seconds gauge",
		// The buckets are chosen around the targets in section 23: 500 ms for a
		// list, 2 s for a dashboard.
		`le="0.5"`,
		`le="2"`,
		`le="+Inf"`,
		"sugarplan_http_request_duration_seconds_sum",
		"sugarplan_http_request_duration_seconds_count",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the exposition is missing %q", want)
		}
	}
}

func TestAnUnmatchedPathIsCountedUnderOneBucket(t *testing.T) {
	ts := newTestServer(t)

	// What a vulnerability scanner does. Each of these must not become a series.
	for _, path := range []string{"/wp-admin", "/.env", "/api/v1/nope"} {
		ts.do(t, "planner", http.MethodGet, path, nil)
	}

	body := ts.do(t, "", http.MethodGet, "/metrics", nil).Body.String()
	for _, path := range []string{"wp-admin", ".env", "v1/nope"} {
		if strings.Contains(body, path) {
			t.Errorf("a probed path became a metric label: %s", path)
		}
	}
}

func TestTheMetricsEndpointCarriesNoBusinessData(t *testing.T) {
	ts := newTestServer(t)
	ts.do(t, "planner", http.MethodGet, "/api/v1/seasons", nil)

	// It is unauthenticated, like the health probes, because that is where a
	// scrape expects it. What justifies that is that it carries nothing worth
	// protecting: route patterns, counts and latencies.
	rec := ts.do(t, "", http.MethodGet, "/metrics", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("a scrape must not need a token: %d", rec.Code)
	}
	body := rec.Body.String()
	for _, secret := range []string{
		"Kampong Speu", "2026-2027", "2300000", "planner@example.com",
		ts.seeded.CompanyID, ts.seeded.FactoryID,
	} {
		if strings.Contains(body, secret) {
			t.Errorf("the exposition leaked %q", secret)
		}
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Errorf("content type = %q, want the Prometheus exposition type", got)
	}
}

// extract pulls the lines mentioning a fragment, for a readable failure.
func extract(body, fragment string) string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, fragment) {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
