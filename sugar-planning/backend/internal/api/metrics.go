package api

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Metrics counts what a production deployment needs to alert on, in the
// Prometheus text exposition format.
//
// It is written by hand rather than pulled in from a client library. The
// library would be the right answer in a system with a hundred instrument
// types; this has four, the format is a few lines, and an on-premises
// deployment that has to vendor its dependencies is better off without another
// one. If that changes, the seam is this file.
//
// What is deliberately not here: a label per URL path. A metric labelled with
// the raw path grows a new series for every id anybody requests, which is the
// standard way to take down a Prometheus server. Requests are labelled with the
// **route pattern** the mux matched, which is a bounded set.
type Metrics struct {
	mu sync.Mutex

	requests  map[requestKey]int64
	durations map[string]*histogram

	// businessJobs is the outcome of each scheduled job, which is the thing an
	// operator actually pages on: an outbox that has stopped dispatching is
	// invisible in a request rate.
	jobRuns   map[jobKey]int64
	jobLastMs map[string]int64

	startedAt time.Time
	now       func() time.Time
}

type requestKey struct {
	route  string
	method string
	status int
}

type jobKey struct {
	job    string
	result string
}

// histogram is a fixed-bucket latency histogram. The buckets are chosen around
// the targets in section 23: 500 ms for a list, 2 s for a dashboard.
type histogram struct {
	counts [len(buckets)]int64
	sum    float64
	total  int64
}

var buckets = [...]float64{0.005, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10}

// NewMetrics builds the registry.
func NewMetrics(now func() time.Time) *Metrics {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Metrics{
		requests:  map[requestKey]int64{},
		durations: map[string]*histogram{},
		jobRuns:   map[jobKey]int64{},
		jobLastMs: map[string]int64{},
		startedAt: now(),
		now:       now,
	}
}

// RecordRequest counts one request against the route pattern it matched.
func (m *Metrics) RecordRequest(route, method string, status int, took time.Duration) {
	if m == nil {
		return
	}
	if route == "" {
		// A request that matched no route is counted under one bucket rather
		// than under its own path, which is what a scanner probing for
		// /wp-admin would otherwise create a series for.
		route = "(unmatched)"
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requests[requestKey{route, method, status}]++

	h := m.durations[route]
	if h == nil {
		h = &histogram{}
		m.durations[route] = h
	}
	seconds := took.Seconds()
	h.sum += seconds
	h.total++
	for i, upper := range buckets {
		if seconds <= upper {
			h.counts[i]++
		}
	}
}

// RecordJob records the outcome of one scheduled job run.
func (m *Metrics) RecordJob(job, result string, took time.Duration) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobRuns[jobKey{job, result}]++
	m.jobLastMs[job] = took.Milliseconds()
}

// Write renders the registry in the Prometheus text exposition format.
func (m *Metrics) Write(w io.Writer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var b strings.Builder

	b.WriteString("# HELP sugarplan_uptime_seconds Time since this instance started.\n")
	b.WriteString("# TYPE sugarplan_uptime_seconds gauge\n")
	fmt.Fprintf(&b, "sugarplan_uptime_seconds %.3f\n\n",
		m.now().Sub(m.startedAt).Seconds())

	b.WriteString("# HELP sugarplan_http_requests_total Requests by route, method and status.\n")
	b.WriteString("# TYPE sugarplan_http_requests_total counter\n")
	keys := make([]requestKey, 0, len(m.requests))
	for k := range m.requests {
		keys = append(keys, k)
	}
	// Sorted output, so a diff of two scrapes is readable and a test can assert
	// on it.
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].route != keys[j].route {
			return keys[i].route < keys[j].route
		}
		if keys[i].method != keys[j].method {
			return keys[i].method < keys[j].method
		}
		return keys[i].status < keys[j].status
	})
	for _, k := range keys {
		fmt.Fprintf(&b, "sugarplan_http_requests_total{route=%q,method=%q,status=\"%d\"} %d\n",
			escapeLabel(k.route), k.method, k.status, m.requests[k])
	}

	b.WriteString("\n# HELP sugarplan_http_request_duration_seconds Request latency by route.\n")
	b.WriteString("# TYPE sugarplan_http_request_duration_seconds histogram\n")
	routes := make([]string, 0, len(m.durations))
	for route := range m.durations {
		routes = append(routes, route)
	}
	sort.Strings(routes)
	for _, route := range routes {
		h := m.durations[route]
		for i, upper := range buckets {
			fmt.Fprintf(&b,
				"sugarplan_http_request_duration_seconds_bucket{route=%q,le=%q} %d\n",
				escapeLabel(route), strconv.FormatFloat(upper, 'g', -1, 64), h.counts[i])
		}
		fmt.Fprintf(&b, "sugarplan_http_request_duration_seconds_bucket{route=%q,le=\"+Inf\"} %d\n",
			escapeLabel(route), h.total)
		fmt.Fprintf(&b, "sugarplan_http_request_duration_seconds_sum{route=%q} %.6f\n",
			escapeLabel(route), h.sum)
		fmt.Fprintf(&b, "sugarplan_http_request_duration_seconds_count{route=%q} %d\n",
			escapeLabel(route), h.total)
	}

	b.WriteString("\n# HELP sugarplan_job_runs_total Scheduled job runs by outcome.\n")
	b.WriteString("# TYPE sugarplan_job_runs_total counter\n")
	jobs := make([]jobKey, 0, len(m.jobRuns))
	for k := range m.jobRuns {
		jobs = append(jobs, k)
	}
	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].job != jobs[j].job {
			return jobs[i].job < jobs[j].job
		}
		return jobs[i].result < jobs[j].result
	})
	for _, k := range jobs {
		fmt.Fprintf(&b, "sugarplan_job_runs_total{job=%q,result=%q} %d\n",
			escapeLabel(k.job), k.result, m.jobRuns[k])
	}

	b.WriteString("\n# HELP sugarplan_job_last_duration_ms How long each job's last run took.\n")
	b.WriteString("# TYPE sugarplan_job_last_duration_ms gauge\n")
	names := make([]string, 0, len(m.jobLastMs))
	for job := range m.jobLastMs {
		names = append(names, job)
	}
	sort.Strings(names)
	for _, job := range names {
		fmt.Fprintf(&b, "sugarplan_job_last_duration_ms{job=%q} %d\n",
			escapeLabel(job), m.jobLastMs[job])
	}

	_, _ = io.WriteString(w, b.String())
}

// escapeLabel makes a value safe to put between quotes in the exposition
// format. Route patterns contain no backslashes or newlines today; escaping
// them anyway costs nothing and means a future route cannot corrupt a scrape.
func escapeLabel(v string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return r.Replace(v)
}

// Instrumentation counts every request against the route pattern it matched.
//
// It sits directly outside the mux and asks the mux which pattern a request
// would match, rather than reading the raw path. That is the whole point: a
// counter labelled with the path grows a series per id, and `/api/v1/seasons/
// {id}` is one series however many seasons there are.
func Instrumentation(m *Metrics, mux *http.ServeMux) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, pattern := mux.Handler(r)

			rec := &statusRecorder{ResponseWriter: w}
			started := time.Now()
			next.ServeHTTP(rec, r)

			status := rec.status
			if status == 0 {
				// A handler that wrote a body without a status wrote a 200.
				status = http.StatusOK
			}
			m.RecordRequest(pattern, r.Method, status, time.Since(started))
		})
	}
}

// handleMetrics serves the registry.
//
// It sits outside /api/v1 and outside authentication, like the health probes,
// because that is where a scrape expects it and because it carries no business
// data: route patterns, counts and latencies. A deployment that wants it closed
// binds it to an internal interface or blocks the path at the ingress, which is
// how this is normally done.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	s.metrics.Write(w)
}
