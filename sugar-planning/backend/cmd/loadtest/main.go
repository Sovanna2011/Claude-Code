// Command loadtest holds the system to the performance targets in section 25
// of the specification, at the history volume section 25 asks it to carry.
//
//	./load-test.sh                            generate and measure
//	go run ./cmd/loadtest -dsn ... -generate  ten years of history
//	go run ./cmd/loadtest -base ... -measure  measure a running instance
//
// The two numbers it exists to check:
//
//	list APIs   p95 under 500 ms   for normal indexed filters
//	dashboards  p95 under 2 s      with caching or pre-aggregation where needed
//
// at "at least 10 years of daily history and high-volume transaction and audit
// data". A target nobody has measured is a wish, and a measurement taken
// against a fortnight of demonstration data is a measurement of nothing: every
// query is fast when the table has four hundred rows in it.
//
// It measures over HTTP against a running instance, because that is where the
// target lives - the connection pool, the JSON encoding and the row scanning
// are all part of what a user waits for, and a benchmark that calls the store
// directly would leave them out.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The targets, quoted from section 25 rather than invented here.
const (
	listTargetP95      = 500 * time.Millisecond
	dashboardTargetP95 = 2000 * time.Millisecond
)

func main() {
	var (
		dsn         = flag.String("dsn", "", "PostgreSQL connection string, for -generate")
		base        = flag.String("base", "http://localhost:8080", "the running instance to measure")
		seasons     = flag.Int("seasons", 10, "seasons of history to generate")
		doGenerate  = flag.Bool("generate", false, "write the history fixture")
		doMeasure   = flag.Bool("measure", false, "measure a running instance")
		concurrency = flag.Int("concurrency", 8, "requests in flight")
		requests    = flag.Int("requests", 60, "requests per endpoint")
		factory     = flag.String("factory", "F1", "the factory code to extend")
		docsPerDay  = flag.Int("docs-per-day", 150, "inventory documents posted per crushing day")
	)
	flag.Parse()

	if !*doGenerate && !*doMeasure {
		fmt.Fprintln(os.Stderr, "nothing to do: pass -generate, -measure, or both")
		os.Exit(2)
	}
	ctx := context.Background()

	if *doGenerate {
		if *dsn == "" {
			fmt.Fprintln(os.Stderr, "-generate needs -dsn")
			os.Exit(2)
		}
		if err := runGenerate(ctx, *dsn, *factory, *seasons, *docsPerDay); err != nil {
			fmt.Fprintf(os.Stderr, "generate: %v\n", err)
			os.Exit(1)
		}
	}

	if *doMeasure {
		code, err := runMeasure(ctx, *base, *concurrency, *requests)
		if err != nil {
			fmt.Fprintf(os.Stderr, "measure: %v\n", err)
			os.Exit(1)
		}
		os.Exit(code)
	}
}

// ---------------------------------------------------------------------------
// Generation
// ---------------------------------------------------------------------------

func runGenerate(ctx context.Context, dsn, factory string, seasons, docsPerDay int) error {
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	ref, err := loadReference(ctx, db, factory)
	if err != nil {
		return err
	}

	sc := defaultScale
	sc.DocsPerDay = docsPerDay
	est := sc.rowEstimate(seasons)
	fmt.Printf("Generating %d seasons of history at factory %s\n", seasons, factory)
	for _, t := range sortedKeys(est) {
		if t == "TOTAL" {
			continue
		}
		fmt.Printf("  %-26s %9s rows\n", t, thousands(est[t]))
	}
	fmt.Printf("  %-26s %9s rows\n\n", "total", thousands(est["TOTAL"]))

	started := time.Now()
	err = generate(ctx, db, ref, seasons, sc, func(msg string) {
		fmt.Printf("  %s\n", msg)
	})
	if err != nil {
		return err
	}
	fmt.Printf("\ndone in %s\n", time.Since(started).Round(time.Second))

	// Read the volume back rather than trusting the estimate: a COPY that
	// silently wrote nothing would otherwise be reported as a success.
	fmt.Println("\nWhat the database now holds:")
	for _, table := range []string{"seasons", "plan_versions", "daily_cane_plans",
		"daily_product_plans", "daily_storage_plans", "inventory_documents",
		"inventory_document_items", "audit_events"} {
		var n int64
		if err := db.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n); err != nil {
			return err
		}
		fmt.Printf("  %-26s %9s rows\n", table, thousands(int(n)))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Measurement
// ---------------------------------------------------------------------------

// probe is one endpoint under test, with the target it is held to.
type probe struct {
	name   string
	path   string // {season} and {version} are substituted
	user   string
	target time.Duration
	kind   string // "list" or "dashboard", for the report
}

func runMeasure(ctx context.Context, base string, concurrency, requests int) (int, error) {
	c := &measureClient{
		base:   strings.TrimRight(base, "/"),
		http:   &http.Client{Timeout: 60 * time.Second},
		tokens: map[string]string{},
	}
	if err := c.waitReady(60 * time.Second); err != nil {
		return 1, fmt.Errorf("the instance at %s is not answering: %w", c.base, err)
	}
	season, version, err := c.discover()
	if err != nil {
		return 1, err
	}
	warehouse, err := c.firstWarehouse()
	if err != nil {
		return 1, err
	}

	probes := []probe{
		// The daily planning grids: the screens somebody has open all day.
		{"daily cane, one season", "/api/v1/versions/{version}/cane?$top=200",
			"planner", listTargetP95, "list"},
		{"daily cane, a fortnight", "/api/v1/versions/{version}/cane?from=2027-01-01&to=2027-01-14",
			"planner", listTargetP95, "list"},
		{"daily production, one season", "/api/v1/versions/{version}/production?$top=200",
			"planner", listTargetP95, "list"},
		{"daily stock ledger", "/api/v1/versions/{version}/storage?$top=200",
			"planner", listTargetP95, "list"},
		// The transaction lists, over the tables the fixture made large.
		{"stock position", "/api/v1/stock?$top=200", "warehouse", listTargetP95, "list"},
		{"inventory documents", "/api/v1/inventory/documents?$top=100",
			"warehouse", listTargetP95, "list"},
		{"documents, one month", "/api/v1/inventory/documents?from=2021-01-01&to=2021-01-31&$top=100",
			"warehouse", listTargetP95, "list"},
		{"documents, one warehouse", "/api/v1/inventory/documents?warehouseId={warehouse}&$top=100",
			"warehouse", listTargetP95, "list"},
		{"audit trail", "/api/v1/audit?$top=100", "auditor", listTargetP95, "list"},
		{"audit, one entity type", "/api/v1/audit?entity=inventory_document&$top=100",
			"auditor", listTargetP95, "list"},
		{"audit, one actor", "/api/v1/audit?actor=loadtest&$top=100",
			"auditor", listTargetP95, "list"},
		{"seasons", "/api/v1/seasons?$top=100", "planner", listTargetP95, "list"},
		{"production orders", "/api/v1/production-orders?$top=100",
			"supervisor", listTargetP95, "list"},
		// And the dashboard, which is the aggregate over all of it.
		{"executive dashboard", "/api/v1/dashboard?seasonId={season}",
			"executive", dashboardTargetP95, "dashboard"},
		{"season summary report", "/api/v1/reports/season-summary?seasonId={season}",
			"executive", dashboardTargetP95, "dashboard"},
		{"capacity forecast report", "/api/v1/reports/capacity-forecast?seasonId={season}",
			"executive", dashboardTargetP95, "dashboard"},
	}

	fmt.Printf("Performance targets, section 25 — %s\n", c.base)
	fmt.Printf("%d cores, load generator and database co-located\n", runtime.NumCPU())
	fmt.Printf("%d requests per endpoint, alone and then %d in flight\n\n", requests, concurrency)

	// Every probe is measured twice, and the two answer different questions.
	//
	// Alone is the query: whether the schema, the indexes and the SQL are
	// right. Under load is the box: whether this hardware can serve that query
	// to that many callers at once. Reporting only the second conflates a slow
	// query with a small machine, and the fix for each is nothing like the fix
	// for the other - one is an index, the other is a bigger server.
	failed, saturated := 0, 0
	fmt.Printf("%-30s %9s %9s %9s   %s\n",
		"", "alone p95", "under p95", "target", "")
	fmt.Println(strings.Repeat("─", 78))

	for _, p := range probes {
		path := strings.ReplaceAll(p.path, "{season}", season)
		path = strings.ReplaceAll(path, "{version}", version)
		path = strings.ReplaceAll(path, "{warehouse}", warehouse)

		alone, err := c.measure(ctx, p.user, path, 1, requests/3+1)
		if err != nil {
			fmt.Printf("%-30s  %v\n", p.name, err)
			failed++
			continue
		}
		under, err := c.measure(ctx, p.user, path, concurrency, requests)
		if err != nil {
			fmt.Printf("%-30s  %v\n", p.name, err)
			failed++
			continue
		}

		verdict := "ok"
		switch {
		case alone.p95 > p.target:
			// Slow on its own: the query is the problem, and no amount of
			// hardware hides that.
			verdict = "SLOW"
			failed++
		case under.p95 > p.target:
			// Fine alone and over under load: this box cannot serve this many
			// callers at once. Worth knowing, and a different conversation.
			verdict = "saturated"
			saturated++
		}
		fmt.Printf("%-30s %9s %9s %9s   %s\n", p.name,
			ms(alone.p95), ms(under.p95), ms(p.target), verdict)
	}

	fmt.Println(strings.Repeat("─", 78))
	switch {
	case failed > 0:
		fmt.Printf("%d endpoint(s) miss the target on their own; that is the query, "+
			"not the hardware\n", failed)
		return 1, nil
	case saturated > 0:
		fmt.Printf("every query is inside its target on its own.\n"+
			"%d endpoint(s) exceed it at %d concurrent callers on %d cores, with the\n"+
			"load generator and the database sharing them. That is a capacity finding\n"+
			"about this machine, not a latency defect - measure again on the hardware\n"+
			"the deployment will actually use before deciding anything from it.\n",
			saturated, concurrency, runtime.NumCPU())
		return 0, nil
	}
	fmt.Printf("every endpoint is inside its target, alone and at %d concurrent callers\n",
		concurrency)
	return 0, nil
}

type stats struct{ p50, p95, p99, max time.Duration }

// measure runs the same request repeatedly and reports the distribution.
//
// The percentile is taken over the whole run rather than per worker: what a
// user experiences is one request among all of them, not one among the ones
// that happened to share a connection.
func (c *measureClient) measure(ctx context.Context, user, path string,
	concurrency, requests int) (stats, error) {

	if _, err := c.token(user); err != nil {
		return stats{}, err
	}
	// One warm request, unmeasured: the first call to any endpoint pays for
	// the connection and the first plan, and reporting that as the p99 would
	// be measuring start-up rather than steady state.
	if code, err := c.once(ctx, user, path); err != nil {
		return stats{}, err
	} else if code >= 400 {
		return stats{}, fmt.Errorf("answered %d", code)
	}

	var (
		mu       sync.Mutex
		samples  = make([]time.Duration, 0, requests)
		firstErr error
	)
	work := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range work {
				started := time.Now()
				code, err := c.once(ctx, user, path)
				took := time.Since(started)
				mu.Lock()
				switch {
				case err != nil && firstErr == nil:
					firstErr = err
				case code >= 400 && firstErr == nil:
					firstErr = fmt.Errorf("answered %d", code)
				default:
					samples = append(samples, took)
				}
				mu.Unlock()
			}
		}()
	}
	for i := 0; i < requests; i++ {
		work <- i
	}
	close(work)
	wg.Wait()

	if firstErr != nil {
		return stats{}, firstErr
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	return stats{
		p50: percentile(samples, 50),
		p95: percentile(samples, 95),
		p99: percentile(samples, 99),
		max: samples[len(samples)-1],
	}, nil
}

// percentile uses the nearest-rank method: the smallest sample at or above the
// rank. With sixty samples an interpolated p95 would invent a number that no
// request actually took.
func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	rank := int(math.Ceil(float64(p) / 100 * float64(len(sorted))))
	if rank < 1 {
		rank = 1
	}
	if rank > len(sorted) {
		rank = len(sorted)
	}
	return sorted[rank-1]
}

func ms(d time.Duration) string {
	return fmt.Sprintf("%d ms", d.Milliseconds())
}

func thousands(n int) string {
	s := fmt.Sprint(n)
	out := ""
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += ","
		}
		out += string(r)
	}
	return out
}

func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ---------------------------------------------------------------------------
// A small HTTP client, kept separate from the acceptance harness because this
// one cares about timing and that one cares about correctness.
// ---------------------------------------------------------------------------

type measureClient struct {
	base   string
	http   *http.Client
	tokens map[string]string
	mu     sync.Mutex
}

func (c *measureClient) waitReady(within time.Duration) error {
	deadline := time.Now().Add(within)
	var last error
	for time.Now().Before(deadline) {
		res, err := c.http.Get(c.base + "/readyz")
		if err == nil {
			_ = res.Body.Close()
			if res.StatusCode == http.StatusOK {
				return nil
			}
			last = fmt.Errorf("/readyz answered %d", res.StatusCode)
		} else {
			last = err
		}
		time.Sleep(250 * time.Millisecond)
	}
	return last
}

func (c *measureClient) token(user string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t, ok := c.tokens[user]; ok {
		return t, nil
	}
	body := strings.NewReader(fmt.Sprintf(`{"username":%q}`, user))
	res, err := c.http.Post(c.base+"/api/v1/auth/dev-login", "application/json", body)
	if err != nil {
		return "", err
	}
	defer func() { _ = res.Body.Close() }()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("signing in as %q: %d %s", user, res.StatusCode, raw)
	}
	token := between(string(raw), `"accessToken":"`, `"`)
	if token == "" {
		return "", fmt.Errorf("signing in as %q returned no token", user)
	}
	c.tokens[user] = token
	return token, nil
}

// once makes one request and discards the body, but reads it fully first: a
// response left unread is a response whose transfer time was never measured.
func (c *measureClient) once(ctx context.Context, user, path string) (int, error) {
	token, err := c.token(user)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = res.Body.Close() }()
	if _, err := io.Copy(io.Discard, res.Body); err != nil {
		return res.StatusCode, err
	}
	return res.StatusCode, nil
}

// discover finds the current season and its plan version, so the probes read
// the same rows a person would.
func (c *measureClient) discover() (season, version string, err error) {
	token, err := c.token("planner")
	if err != nil {
		return "", "", err
	}
	get := func(path string) (string, error) {
		req, _ := http.NewRequest(http.MethodGet, c.base+path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		res, err := c.http.Do(req)
		if err != nil {
			return "", err
		}
		defer func() { _ = res.Body.Close() }()
		raw, _ := io.ReadAll(res.Body)
		if res.StatusCode != http.StatusOK {
			return "", fmt.Errorf("GET %s: %d %s", path, res.StatusCode, raw)
		}
		return string(raw), nil
	}

	body, err := get("/api/v1/seasons?$top=100")
	if err != nil {
		return "", "", err
	}
	// The seeded reference season, which is the one with a fortnight of real
	// actuals against ten years of fixture around it.
	idx := strings.Index(body, `"code":"2026-2027"`)
	if idx == -1 {
		return "", "", fmt.Errorf("season 2026-2027 is not in this instance")
	}
	season = between(body[:idx], `"id":"`, `"`)
	for {
		next := strings.Index(body[:idx], `"id":"`)
		if next == -1 {
			break
		}
		season = between(body[next:idx], `"id":"`, `"`)
		break
	}
	// Take the id nearest before the code, which is the same object.
	last := strings.LastIndex(body[:idx], `"id":"`)
	if last == -1 {
		return "", "", fmt.Errorf("could not read the season id")
	}
	season = between(body[last:], `"id":"`, `"`)

	body, err = get("/api/v1/seasons/" + season + "/versions")
	if err != nil {
		return "", "", err
	}
	at := strings.Index(body, `"planType":"BUDGET"`)
	if at == -1 {
		return "", "", fmt.Errorf("the season has no budget version")
	}
	last = strings.LastIndex(body[:at], `"id":"`)
	version = between(body[last:], `"id":"`, `"`)
	if season == "" || version == "" {
		return "", "", fmt.Errorf("could not read the season and version ids")
	}
	return season, version, nil
}

// firstWarehouse gives the probes a real id to filter by, since a filter on
// nothing measures the unfiltered query all over again.
func (c *measureClient) firstWarehouse() (string, error) {
	token, err := c.token("warehouse")
	if err != nil {
		return "", err
	}
	req, _ := http.NewRequest(http.MethodGet, c.base+"/api/v1/master/warehouses?$top=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = res.Body.Close() }()
	raw, _ := io.ReadAll(res.Body)
	id := between(string(raw), `"id":"`, `"`)
	if id == "" {
		return "", fmt.Errorf("no warehouse to filter by: %s", raw)
	}
	return id, nil
}

func between(s, start, end string) string {
	i := strings.Index(s, start)
	if i == -1 {
		return ""
	}
	s = s[i+len(start):]
	j := strings.Index(s, end)
	if j == -1 {
		return ""
	}
	return s[:j]
}
