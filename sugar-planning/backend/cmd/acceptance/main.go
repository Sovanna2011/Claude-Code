// Command acceptance drives a running instance over HTTP and checks it against
// the acceptance criteria in section 27 of the specification.
//
//	./test-system.sh                       boot a test instance and check it
//	go run ./cmd/acceptance -base URL      check an instance already running
//
// It is not a unit test. Every check goes over the wire against a server that
// is really running, signed in as a real account, and reads back what the API
// returns - which is the only way to find the things a unit test cannot: a
// route that is registered but not wired, a permission that is enforced in the
// service and forgotten in the handler, a figure that reconciles inside one
// process and not across two requests.
//
// It prints one line per check and a verdict per criterion, and exits non-zero
// if anything failed. Nothing it does is destructive to the demonstration data:
// it builds its own season and posts its own documents, and reads everything
// else.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {
	var (
		base    = flag.String("base", "http://localhost:8080", "the running instance to check")
		timeout = flag.Duration("timeout", 60*time.Second, "how long to wait for the instance to answer")
		verbose = flag.Bool("v", false, "print every request as it is made")
	)
	flag.Parse()

	c := &client{
		base:    strings.TrimRight(*base, "/"),
		http:    &http.Client{Timeout: 30 * time.Second},
		tokens:  map[string]string{},
		verbose: *verbose,
	}
	if err := c.waitReady(*timeout); err != nil {
		fmt.Fprintf(os.Stderr, "the instance at %s is not answering: %v\n", c.base, err)
		os.Exit(2)
	}
	if err := c.requireDevLogin(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}

	r := &run{client: c}
	fmt.Printf("Acceptance criteria, section 27 — %s\n\n", c.base)
	runAll(r)
	os.Exit(r.report())
}

// ---------------------------------------------------------------------------
// The client
// ---------------------------------------------------------------------------

type client struct {
	base    string
	http    *http.Client
	tokens  map[string]string
	verbose bool
}

// resp is what a check reads: the status, the headers and the raw body. The
// body is kept as bytes because half the checks want JSON and the other half
// want to know that a PDF starts with %PDF.
type resp struct {
	Status int
	Header http.Header
	Body   []byte
}

func (r *resp) decode(v any) error {
	if err := json.Unmarshal(r.Body, v); err != nil {
		return fmt.Errorf("the response is not the JSON expected: %w (%s)", err, r.snippet())
	}
	return nil
}

// snippet is the part of a body worth putting in a failure message. A whole
// dashboard in a terminal helps nobody find the problem.
func (r *resp) snippet() string {
	s := strings.TrimSpace(string(r.Body))
	if len(s) > 300 {
		s = s[:300] + "…"
	}
	return strings.ReplaceAll(s, "\n", " ")
}

func (c *client) waitReady(within time.Duration) error {
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

// requireDevLogin refuses to run anywhere the development login is off.
//
// That is the guard that keeps this harness away from a real plant: it signs in
// as canned accounts and posts stock, and an instance that authenticates
// against the enterprise identity provider will not issue it a token. Better to
// say so at the start than to fail eleven criteria one at a time.
func (c *client) requireDevLogin() error {
	res, err := c.do("", http.MethodGet, "/api/v1/auth/dev-users", nil, nil)
	if err != nil {
		return err
	}
	var page struct {
		Value []struct {
			Username string `json:"username"`
		} `json:"value"`
	}
	if err := res.decode(&page); err != nil {
		return err
	}
	if len(page.Value) == 0 {
		return fmt.Errorf("%s has no development accounts, so it is either a real "+
			"deployment or is running in oidc mode; this harness signs in as canned "+
			"accounts and will not run here", c.base)
	}
	return nil
}

// token signs in and remembers the result, so a run makes one login per account
// rather than one per request.
func (c *client) token(user string) (string, error) {
	if t, ok := c.tokens[user]; ok {
		return t, nil
	}
	res, err := c.do("", http.MethodPost, "/api/v1/auth/dev-login",
		map[string]string{"username": user}, nil)
	if err != nil {
		return "", err
	}
	if res.Status != http.StatusOK {
		return "", fmt.Errorf("signing in as %q: %d %s", user, res.Status, res.snippet())
	}
	var body struct {
		AccessToken string `json:"accessToken"`
	}
	if err := res.decode(&body); err != nil {
		return "", err
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("signing in as %q returned no token", user)
	}
	c.tokens[user] = body.AccessToken
	return body.AccessToken, nil
}

// do makes one request. A nil body sends none; a []byte body is sent as-is,
// which is how the import check uploads a file; anything else is marshalled.
func (c *client) do(user, method, path string, body any, header http.Header) (*resp, error) {
	var reader io.Reader
	contentType := ""
	switch b := body.(type) {
	case nil:
	case []byte:
		reader, contentType = bytes.NewReader(b), "text/csv"
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			return nil, err
		}
		reader, contentType = bytes.NewReader(raw), "application/json"
	}

	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	if user != "" {
		t, err := c.token(user)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+t)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if c.verbose {
		fmt.Printf("    %-6s %-60s → %d (%d bytes) as %s\n",
			method, path, res.StatusCode, len(raw), orAnonymous(user))
	}
	return &resp{Status: res.StatusCode, Header: res.Header, Body: raw}, nil
}

func orAnonymous(user string) string {
	if user == "" {
		return "nobody"
	}
	return user
}

// get, post and put are the shorthands the checks are written in. Each returns
// the response and an error only for a transport failure - a 403 is an answer,
// and often the answer a check is looking for.
func (c *client) get(user, path string) (*resp, error) {
	return c.do(user, http.MethodGet, path, nil, nil)
}

func (c *client) post(user, path string, body any) (*resp, error) {
	return c.do(user, http.MethodPost, path, body, nil)
}

func (c *client) put(user, path string, body any) (*resp, error) {
	return c.do(user, http.MethodPut, path, body, nil)
}

// getOK is for the majority of reads, where anything but 200 is a failure and
// the check wants the decoded body.
func (c *client) getOK(user, path string, into any) error {
	res, err := c.get(user, path)
	if err != nil {
		return err
	}
	if res.Status != http.StatusOK {
		return fmt.Errorf("GET %s as %s: %d %s", path, user, res.Status, res.snippet())
	}
	if into == nil {
		return nil
	}
	return res.decode(into)
}

// ---------------------------------------------------------------------------
// The report
// ---------------------------------------------------------------------------

type status string

const (
	pass status = "PASS"
	fail status = "FAIL"
	skip status = "SKIP"
)

type result struct {
	criterion string
	name      string
	status    status
	detail    string
	took      time.Duration
}

// run holds the client, the results and the handful of ids the checks discover
// as they go: the second criterion needs the version the first one built, and
// re-deriving it would be slower and would hide a failure behind a lookup.
type run struct {
	*client
	criterion string
	results   []result
	state     state
}

// state is what one check learns and the next one needs.
type state struct {
	factoryID   string
	companyID   string
	seasonID    string // the harness's own season
	versionID   string // its draft, released by criterion 2
	variantID   string // a copy of it, for the comparison
	demoSeason  string // the seeded reference season
	demoActual  string // its actuals container
	btbSeason   string // the second tenant's season
	warehouseID string
	productID   string
	packagingID string
	products    map[string]string
	warehouses  map[string]string
}

// criterionFn is one line of section 27. It gets the run and returns nothing:
// every check inside it records its own result, so one failing check does not
// hide the ones after it.
func (r *run) on(criterion string, f func(*run)) {
	r.criterion = criterion
	f(r)
}

// check runs one assertion and records it.
func (r *run) check(name string, f func() error) bool {
	started := time.Now()
	err := f()
	res := result{criterion: r.criterion, name: name, took: time.Since(started)}
	switch {
	case err == nil:
		res.status = pass
	case isSkip(err):
		res.status, res.detail = skip, strings.TrimPrefix(err.Error(), skipPrefix)
	default:
		res.status, res.detail = fail, err.Error()
	}
	r.results = append(r.results, res)
	fmt.Printf("  %-4s %s\n", res.status, name)
	if res.detail != "" {
		fmt.Printf("       %s\n", res.detail)
	}
	return res.status == pass
}

const skipPrefix = "skip: "

// skipped marks a check the harness cannot make from here. It is deliberately
// distinct from a pass: a criterion nobody checked must not read as a criterion
// that was met.
func skipped(format string, a ...any) error {
	return fmt.Errorf(skipPrefix+format, a...)
}

func isSkip(err error) bool { return strings.HasPrefix(err.Error(), skipPrefix) }

// report prints the verdict per criterion and returns the exit code.
func (r *run) report() int {
	byCriterion := map[string][]result{}
	order := []string{}
	for _, res := range r.results {
		if _, seen := byCriterion[res.criterion]; !seen {
			order = append(order, res.criterion)
		}
		byCriterion[res.criterion] = append(byCriterion[res.criterion], res)
	}
	sort.SliceStable(order, func(i, j int) bool { return criteria[order[i]].n < criteria[order[j]].n })

	fmt.Printf("\n%s\n", strings.Repeat("─", 78))
	failed, skippedCount := 0, 0
	for _, code := range order {
		checks := byCriterion[code]
		verdict, bad, unchecked := pass, 0, 0
		for _, c := range checks {
			switch c.status {
			case fail:
				bad++
			case skip:
				unchecked++
			}
		}
		switch {
		case bad > 0:
			verdict = fail
			failed++
		case unchecked == len(checks):
			verdict = skip
			skippedCount++
		}
		fmt.Printf("%-4s %2d. %s\n", verdict, criteria[code].n, criteria[code].text)
		if bad > 0 {
			fmt.Printf("        %d of %d checks failed\n", bad, len(checks))
		}
	}
	fmt.Printf("%s\n", strings.Repeat("─", 78))

	total := len(r.results)
	badChecks := 0
	for _, c := range r.results {
		if c.status == fail {
			badChecks++
		}
	}
	fmt.Printf("%d checks, %d failed, %d criteria failed, %d not checkable from here\n",
		total, badChecks, failed, skippedCount)
	if failed > 0 {
		return 1
	}
	return 0
}

// criteria is section 27, verbatim, so the report can be read next to the
// specification without anybody having to translate.
var criteria = map[string]struct {
	n    int
	text string
}{
	"C1":  {1, "A planner can create a season plan, enter assumptions, generate daily targets, and compare versions"},
	"C2":  {2, "An approver can approve/reject and release a locked baseline with full history"},
	"C3":  {3, "Operators can enter actual cane, production, packing, shipment, downtime, and quality data by shift/day"},
	"C4":  {4, "The system automatically calculates cumulative totals, recovery, balances, stock, capacity use, and forecast risk dates"},
	"C5":  {5, "Inventory and production postings are atomic, reversible through documents, and auditable"},
	"C6":  {6, "Dashboards reconcile to transaction data and the seeded scenario"},
	"C7":  {7, "Capacity, shortage, quality, downtime, and variance alerts work with configurable thresholds"},
	"C8":  {8, "Role and factory/company restrictions are enforced by the backend"},
	"C9":  {9, "Excel import identifies errors before commit and preserves source-row traceability"},
	"C10": {10, "Reports export correctly to Excel, PDF, and CSV"},
	"C11": {11, "Automated tests pass, no critical security findings remain, and backup/restore has been demonstrated"},
}
