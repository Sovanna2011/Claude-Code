package httpapi_test

// End-to-end tests over the real stack: HTTP handler → service → repository → PostgreSQL with
// PostGIS. They exercise what no in-memory substitute can — that the SQL is valid, that the
// constraint triggers actually fire, that ST_Area agrees with the stored figures, and that the
// tree, the KPI cards and the map describe the same land.
//
// They need a database. Set FARMAREA_TEST_DATABASE_URL to run them; without it they skip, so
// `go test ./...` is still useful on a machine with no Postgres.
//
//	createdb farmarea_test && psql -d farmarea_test -c 'CREATE EXTENSION postgis'
//	export FARMAREA_TEST_DATABASE_URL="postgres://farmarea:farmarea@127.0.0.1:5432/farmarea_test"

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sovanna2011/sugarcane-go/backend/internal/auth"
	"github.com/sovanna2011/sugarcane-go/backend/internal/database"
	"github.com/sovanna2011/sugarcane-go/backend/internal/httpapi"
	"github.com/sovanna2011/sugarcane-go/backend/internal/repository"
	"github.com/sovanna2011/sugarcane-go/backend/internal/service"
)

type harness struct {
	server *httptest.Server
	db     *database.DB
	tokens map[string]string
}

var (
	once   sync.Once
	shared *harness
)

func newHarness(t *testing.T) *harness {
	t.Helper()
	url := os.Getenv("FARMAREA_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set FARMAREA_TEST_DATABASE_URL to run the database tests")
	}

	once.Do(func() {
		log := slog.New(slog.NewTextHandler(io.Discard, nil))
		ctx := context.Background()

		db, err := database.Open(ctx, url, log)
		if err != nil {
			t.Fatalf("open test database: %v", err)
		}
		if err := db.Migrate(ctx, "../../../db/migrations"); err != nil {
			t.Fatalf("migrate test database: %v", err)
		}

		support := repository.NewSupportRepository(db)
		svc := service.New(db, service.Repositories{
			Master:      repository.NewMasterRepository(db),
			Dashboard:   repository.NewDashboardRepository(db),
			Support:     support,
			Activities:  repository.NewActivityRepository(db),
			Projections: repository.NewProjectionRepository(db),
			Plans:       repository.NewPlanRepository(db),
		})
		tokens := auth.NewTokens("a-test-signing-key-that-is-long-enough", time.Hour)
		api := httpapi.New(svc, tokens, support, db, log)

		// An approved projection takes its blocks for its planting window, and the test database
		// is not dropped between runs — so last run's approvals would refuse this run's. The
		// projections these tests create are all numbered PRJ-…; clearing them leaves the seeded
		// plantation, which every other test reads, exactly as it was.
		if _, err := db.Pool().Exec(ctx,
			`DELETE FROM planting_projection WHERE projection_no LIKE 'PRJ-%'`); err != nil {
			panic("clear test projections: " + err.Error())
		}

		shared = &harness{
			server: httptest.NewServer(api.Routes([]string{"http://localhost:8081"})),
			db:     db,
			tokens: map[string]string{},
		}
	})
	return shared
}

func (h *harness) token(t *testing.T, user string) string {
	t.Helper()
	if tok, ok := h.tokens[user]; ok {
		return tok
	}
	body := h.do(t, "POST", "/api/auth/login", "", map[string]string{
		"userName": user, "password": "Farm#2026",
	}, http.StatusOK)

	var out struct {
		Token string `json:"token"`
	}
	decode(t, body, &out)
	h.tokens[user] = out.Token
	return out.Token
}

// do issues a request and asserts the status, returning the body. A wrong status prints what the
// server actually said, which is the difference between a one-minute fix and a debugging session.
func (h *harness) do(t *testing.T, method, path, token string, body any, wantStatus int) []byte {
	t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, h.server.URL+path, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := h.server.Client().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s %s = %d, want %d\nbody: %s", method, path, resp.StatusCode, wantStatus, raw)
	}
	return raw
}

func decode(t *testing.T, raw []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
}

type areas struct {
	Total        float64 `json:"totalAreaHa"`
	Plantable    float64 `json:"plantableAreaHa"`
	NonPlantable float64 `json:"nonPlantableAreaHa"`
	NewPlanting  float64 `json:"newPlantingAreaHa"`
	Ratoon       float64 `json:"ratoonAreaHa"`
	WithCane     float64 `json:"areaWithCaneHa"`
	Available    float64 `json:"availableAreaHa"`
}

type node struct {
	NodeType   string  `json:"nodeType"`
	ID         int     `json:"id"`
	Code       string  `json:"code"`
	Areas      areas   `json:"areas"`
	MapURL     string  `json:"mapUrl"`
	BlockCount int     `json:"blockCount"`
	Children   []*node `json:"children"`
}

func near(a, b float64) bool { return math.Abs(a-b) < 0.01 }

// errorCode reads the machine-readable code out of a refusal, so a test can assert which rule
// answered rather than only that something was refused.
func errorCode(t *testing.T, raw []byte) string {
	t.Helper()
	var body struct {
		Code string `json:"code"`
	}
	decode(t, raw, &body)
	return body.Code
}

// ---------------------------------------------------------------- authentication

func TestLogin(t *testing.T) {
	h := newHarness(t)

	body := h.do(t, "POST", "/api/auth/login", "", map[string]string{
		"userName": "admin", "password": "Farm#2026"}, http.StatusOK)
	var ok struct {
		Token string `json:"token"`
		User  struct {
			Username string `json:"userName"`
			Role     string `json:"role"`
		} `json:"user"`
	}
	decode(t, body, &ok)
	if ok.Token == "" || ok.User.Role != "Admin" {
		t.Fatalf("login returned %+v", ok)
	}

	h.do(t, "POST", "/api/auth/login", "", map[string]string{
		"userName": "admin", "password": "wrong"}, http.StatusUnauthorized)

	// An unknown user must be indistinguishable from a wrong password.
	unknown := h.do(t, "POST", "/api/auth/login", "", map[string]string{
		"userName": "nobody", "password": "wrong"}, http.StatusUnauthorized)
	wrong := h.do(t, "POST", "/api/auth/login", "", map[string]string{
		"userName": "admin", "password": "wrong"}, http.StatusUnauthorized)
	if string(unknown) != string(wrong) {
		t.Errorf("the two answers differ, which lets an attacker enumerate accounts:\n%s\n%s", unknown, wrong)
	}
}

func TestProtectedEndpointsRequireAToken(t *testing.T) {
	h := newHarness(t)
	for _, path := range []string{"/api/dashboard/farm-area", "/api/farms/tree", "/api/blocks"} {
		h.do(t, "GET", path, "", nil, http.StatusUnauthorized)
	}
}

func TestAViewerMayReadButNotWrite(t *testing.T) {
	h := newHarness(t)
	viewer := h.token(t, "viewer")

	h.do(t, "GET", "/api/dashboard/farm-area", viewer, nil, http.StatusOK)
	h.do(t, "POST", "/api/farms", viewer, map[string]any{
		"plantationId": 1, "code": "FRM-99", "name": "Should not be created"}, http.StatusForbidden)

	// Only an administrator sees who changed what.
	h.do(t, "GET", "/api/audit", viewer, nil, http.StatusForbidden)
	h.do(t, "GET", "/api/audit", h.token(t, "admin"), nil, http.StatusOK)
}

// ---------------------------------------------------------------- the numbers agree

func TestKPIsMatchTheTreeAndTheMap(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")

	var kpi struct {
		Areas      areas `json:"areas"`
		BlockCount int   `json:"blockCount"`
		KPIs       []struct {
			Key     string  `json:"key"`
			AreaHa  float64 `json:"areaHa"`
			Percent float64 `json:"percentOfTotalArea"`
		} `json:"kpis"`
	}
	decode(t, h.do(t, "GET", "/api/dashboard/farm-area?cropYear=2026", admin, nil, http.StatusOK), &kpi)

	var tree []*node
	decode(t, h.do(t, "GET", "/api/farms/tree?cropYear=2026", admin, nil, http.StatusOK), &tree)

	var summed areas
	blocks := 0
	for _, farm := range tree {
		summed.Total += farm.Areas.Total
		summed.Plantable += farm.Areas.Plantable
		summed.NonPlantable += farm.Areas.NonPlantable
		summed.NewPlanting += farm.Areas.NewPlanting
		summed.Ratoon += farm.Areas.Ratoon
		summed.WithCane += farm.Areas.WithCane
		summed.Available += farm.Areas.Available
		blocks += farm.BlockCount
	}

	if !near(summed.Total, kpi.Areas.Total) || !near(summed.WithCane, kpi.Areas.WithCane) ||
		!near(summed.Available, kpi.Areas.Available) || !near(summed.NonPlantable, kpi.Areas.NonPlantable) {
		t.Fatalf("the tree and the KPI cards disagree:\ntree %+v\nkpi  %+v", summed, kpi.Areas)
	}
	if blocks != kpi.BlockCount {
		t.Errorf("block count: tree %d, kpi %d", blocks, kpi.BlockCount)
	}

	// The classification has to close: total is plantable plus unplantable, and plantable is the
	// cane plus what is still available.
	if !near(kpi.Areas.Total, kpi.Areas.Plantable+kpi.Areas.NonPlantable) {
		t.Errorf("total %v != plantable %v + cannot plant %v",
			kpi.Areas.Total, kpi.Areas.Plantable, kpi.Areas.NonPlantable)
	}
	if !near(kpi.Areas.Plantable, kpi.Areas.WithCane+kpi.Areas.Available) {
		t.Errorf("plantable %v != with cane %v + available %v",
			kpi.Areas.Plantable, kpi.Areas.WithCane, kpi.Areas.Available)
	}
	if !near(kpi.Areas.WithCane, kpi.Areas.NewPlanting+kpi.Areas.Ratoon) {
		t.Errorf("with cane %v != new planting %v + ratoon %v",
			kpi.Areas.WithCane, kpi.Areas.NewPlanting, kpi.Areas.Ratoon)
	}

	// Every card carries its share of the total, which is what the specification's table shows.
	for _, card := range kpi.KPIs {
		want := 0.0
		if kpi.Areas.Total > 0 {
			want = math.Round(card.AreaHa/kpi.Areas.Total*10000) / 100
		}
		if !near(card.Percent, want) {
			t.Errorf("%s: %.2f%% of total, want %.2f%%", card.Key, card.Percent, want)
		}
	}
}

func TestTreeRollsUpAtEveryLevel(t *testing.T) {
	h := newHarness(t)
	var tree []*node
	decode(t, h.do(t, "GET", "/api/farms/tree?cropYear=2026", h.token(t, "admin"), nil, http.StatusOK), &tree)

	if len(tree) == 0 {
		t.Fatal("no farms")
	}
	emptyZones := 0
	for _, farm := range tree {
		var farmSum areas
		for _, zone := range farm.Children {
			var zoneSum areas
			for _, block := range zone.Children {
				zoneSum.Total += block.Areas.Total
				zoneSum.WithCane += block.Areas.WithCane
				zoneSum.Available += block.Areas.Available
			}
			if len(zone.Children) == 0 {
				emptyZones++
				if zone.Areas.Total != 0 {
					t.Errorf("zone %s has no blocks but reports %v ha", zone.Code, zone.Areas.Total)
				}
			}
			if !near(zoneSum.Total, zone.Areas.Total) || !near(zoneSum.WithCane, zone.Areas.WithCane) {
				t.Errorf("zone %s: blocks sum to %+v, zone reports %+v", zone.Code, zoneSum, zone.Areas)
			}
			farmSum.Total += zone.Areas.Total
			farmSum.WithCane += zone.Areas.WithCane
		}
		if !near(farmSum.Total, farm.Areas.Total) || !near(farmSum.WithCane, farm.Areas.WithCane) {
			t.Errorf("farm %s: zones sum to %+v, farm reports %+v", farm.Code, farmSum, farm.Areas)
		}
		if farm.MapURL == "" {
			t.Errorf("farm %s has no map link", farm.Code)
		}
	}
	if emptyZones == 0 {
		t.Error("the sample estate should include a zone with no blocks, to prove they survive the roll-up")
	}
}

// ---------------------------------------------------------------- filters

func TestFiltersNarrowEveryPartOfTheDashboardTogether(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")

	var all, oneFarm struct {
		Areas      areas `json:"areas"`
		BlockCount int   `json:"blockCount"`
	}
	decode(t, h.do(t, "GET", "/api/dashboard/farm-area?cropYear=2026", admin, nil, http.StatusOK), &all)
	decode(t, h.do(t, "GET", "/api/dashboard/farm-area?cropYear=2026&farmId=1", admin, nil, http.StatusOK), &oneFarm)

	if oneFarm.Areas.Total >= all.Areas.Total || oneFarm.BlockCount >= all.BlockCount {
		t.Fatalf("filtering by farm did not narrow anything: all %+v, one farm %+v", all, oneFarm)
	}

	var tree []*node
	decode(t, h.do(t, "GET", "/api/farms/tree?cropYear=2026&farmId=1", admin, nil, http.StatusOK), &tree)
	if len(tree) != 1 || tree[0].ID != 1 {
		t.Fatalf("the tree ignored the farm filter: %d farms", len(tree))
	}
	if !near(tree[0].Areas.Total, oneFarm.Areas.Total) {
		t.Errorf("tree %v and KPI %v disagree under the same filter", tree[0].Areas.Total, oneFarm.Areas.Total)
	}

	mapData := readMap(t, h, admin, "cropYear=2026&farmId=1")
	if len(mapData.Features.Features) != oneFarm.BlockCount {
		t.Errorf("the map drew %d blocks, the cards counted %d", len(mapData.Features.Features), oneFarm.BlockCount)
	}
}

// mapResponse is the shape of GET /api/dashboard/farm-area/map at any of its three levels.
type mapResponse struct {
	Level    string        `json:"level"`
	Features mapCollection `json:"features"`
	Outlines mapCollection `json:"outlines"`
}

type mapCollection struct {
	Features []struct {
		Geometry   map[string]any `json:"geometry"`
		Properties map[string]any `json:"properties"`
	} `json:"features"`
}

func readMap(t *testing.T, h *harness, token, query string) mapResponse {
	t.Helper()
	var out mapResponse
	decode(t, h.do(t, "GET", "/api/dashboard/farm-area/map?"+query, token, nil, http.StatusOK), &out)
	return out
}

// The map can be drawn by farm, by zone or by block. Whichever level is asked for, it has to
// describe the same land as the KPI cards above it — the point of the level is to group the land
// differently, not to measure a different amount of it.
func TestMapDrawsTheSameLandAtEveryLocationLevel(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")

	var cards struct {
		Areas      areas `json:"areas"`
		BlockCount int   `json:"blockCount"`
	}
	decode(t, h.do(t, "GET", "/api/dashboard/farm-area?cropYear=2026", admin, nil, http.StatusOK), &cards)

	// The tree is the authority on which locations exist. The map may leave out one that has no
	// shape to draw — a zone surveyed on paper only — but it may never draw one the tree does not
	// list, and it may never leave out one that holds land.
	var tree []*node
	decode(t, h.do(t, "GET", "/api/farms/tree?cropYear=2026", admin, nil, http.StatusOK), &tree)

	known := map[string]map[int]bool{"Farm": {}, "Zone": {}, "Block": {}}
	holdsLand := map[string]map[int]bool{"Farm": {}, "Zone": {}, "Block": {}}
	var walk func(level string, nodes []*node)
	walk = func(level string, nodes []*node) {
		next := map[string]string{"Farm": "Zone", "Zone": "Block"}[level]
		for _, n := range nodes {
			known[level][n.ID] = true
			if n.Areas.Total > 0 {
				holdsLand[level][n.ID] = true
			}
			if next != "" {
				walk(next, n.Children)
			}
		}
	}
	walk("Farm", tree)

	// Under the filled features the map draws the registered boundaries, so a location shows both
	// its full extent and the part of it the filter admits.
	outlineLevels := map[string][]string{
		"Farm":  {"Farm"},
		"Zone":  {"Farm", "Zone"},
		"Block": {"Farm", "Zone"},
	}

	for _, level := range []string{"Farm", "Zone", "Block"} {
		drawn := readMap(t, h, admin, "cropYear=2026&level="+level)

		if drawn.Level != level {
			t.Errorf("asked for %s, the response says %q", level, drawn.Level)
		}
		total := 0.0
		wasDrawn := map[int]bool{}
		for _, feature := range drawn.Features.Features {
			if feature.Geometry["type"] == nil {
				t.Errorf("%s level drew a feature with no geometry: %v", level, feature.Properties["code"])
			}
			if feature.Properties["level"] != level {
				t.Errorf("%s level tagged a feature %q", level, feature.Properties["level"])
			}
			for _, key := range []string{"id", "code", "name", "totalAreaHa", "plantedPercent"} {
				if feature.Properties[key] == nil {
					t.Errorf("%s level feature is missing %s", level, key)
				}
			}
			id := int(feature.Properties["id"].(float64))
			if !known[level][id] {
				t.Errorf("%s level drew %v, which the tree does not list", level, feature.Properties["code"])
			}
			wasDrawn[id] = true
			total += feature.Properties["totalAreaHa"].(float64)
		}
		for id := range holdsLand[level] {
			if !wasDrawn[id] {
				t.Errorf("%s level left out %d, which holds land", level, id)
			}
		}
		if !near(total, cards.Areas.Total) {
			t.Errorf("%s level totals %v ha, the cards say %v ha", level, total, cards.Areas.Total)
		}

		seen := map[string]bool{}
		for _, outline := range drawn.Outlines.Features {
			seen[outline.Properties["level"].(string)] = true
		}
		if len(seen) != len(outlineLevels[level]) {
			t.Errorf("%s level drew outlines for %v, expected %v", level, seen, outlineLevels[level])
		}
		for _, want := range outlineLevels[level] {
			if !seen[want] {
				t.Errorf("%s level drew no %s outline", level, want)
			}
		}
	}
}

// A farm-level map under a zone filter has to describe the zone, not the whole farm: the shape and
// the figures both come from the blocks the filter admits.
func TestMapByFarmFollowsTheFilterBeneathIt(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")

	whole := readMap(t, h, admin, "cropYear=2026&farmId=1&level=Farm")
	narrowed := readMap(t, h, admin, "cropYear=2026&farmId=1&zoneId=1&level=Farm")

	if len(whole.Features.Features) != 1 || len(narrowed.Features.Features) != 1 {
		t.Fatalf("expected one farm each, drew %d and %d",
			len(whole.Features.Features), len(narrowed.Features.Features))
	}
	wholeArea := whole.Features.Features[0].Properties["totalAreaHa"].(float64)
	narrowedArea := narrowed.Features.Features[0].Properties["totalAreaHa"].(float64)
	if !(narrowedArea < wholeArea) {
		t.Errorf("one zone of the farm measures %v ha, the whole farm %v ha", narrowedArea, wholeArea)
	}

	var zoneSummary []struct {
		ID    int   `json:"id"`
		Areas areas `json:"areas"`
	}
	decode(t, h.do(t, "GET", "/api/summaries/zone-area?cropYear=2026&farmId=1&zoneId=1", admin, nil, http.StatusOK), &zoneSummary)
	if len(zoneSummary) != 1 {
		t.Fatalf("expected one zone summary, got %d", len(zoneSummary))
	}
	if !near(narrowedArea, zoneSummary[0].Areas.Total) {
		t.Errorf("the farm-level map says %v ha, the zone summary %v ha", narrowedArea, zoneSummary[0].Areas.Total)
	}
}

func TestMapRefusesAnUnknownLocationLevel(t *testing.T) {
	h := newHarness(t)
	body := h.do(t, "GET", "/api/dashboard/farm-area/map?level=district", h.token(t, "admin"), nil, http.StatusBadRequest)
	if code := errorCode(t, body); code != "INVALID_FILTER" {
		t.Errorf("expected INVALID_FILTER, got %s", code)
	}
}

func TestPlantingTypeFilterSplitsTheCane(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")

	read := func(query string) areas {
		var out struct {
			Areas areas `json:"areas"`
		}
		decode(t, h.do(t, "GET", "/api/dashboard/farm-area?"+query, admin, nil, http.StatusOK), &out)
		return out.Areas
	}

	all := read("cropYear=2026")
	newOnly := read("cropYear=2026&plantingType=NewPlanting")
	ratoonOnly := read("cropYear=2026&plantingType=Ratoon")

	if !near(newOnly.Ratoon, 0) || !near(ratoonOnly.NewPlanting, 0) {
		t.Errorf("the filter leaked: new-only ratoon %v, ratoon-only new %v", newOnly.Ratoon, ratoonOnly.NewPlanting)
	}
	if !near(newOnly.NewPlanting+ratoonOnly.Ratoon, all.WithCane) {
		t.Errorf("%v + %v != %v", newOnly.NewPlanting, ratoonOnly.Ratoon, all.WithCane)
	}
	// The land itself does not change when the crop filter does.
	if !near(newOnly.Total, all.Total) {
		t.Errorf("filtering by planting type changed the total area from %v to %v", all.Total, newOnly.Total)
	}
}

func TestAnUnknownFilterValueIsRefused(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")
	body := h.do(t, "GET", "/api/dashboard/farm-area?plantingType=Coppice", admin, nil, http.StatusBadRequest)
	if !strings.Contains(string(body), "INVALID_FILTER") {
		t.Errorf("expected INVALID_FILTER, got %s", body)
	}
}

func TestSearchFindsABlockByCode(t *testing.T) {
	h := newHarness(t)
	var result struct {
		Items      []map[string]any `json:"items"`
		TotalCount int              `json:"totalCount"`
	}
	decode(t, h.do(t, "GET", "/api/blocks?search=BLK-001", h.token(t, "admin"), nil, http.StatusOK), &result)
	if result.TotalCount != 1 || result.Items[0]["code"] != "BLK-001" {
		t.Fatalf("search returned %+v", result)
	}
}

func TestPagination(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")

	var page1, page2 struct {
		Items      []map[string]any `json:"items"`
		Page       int              `json:"page"`
		PageSize   int              `json:"pageSize"`
		TotalCount int              `json:"totalCount"`
		TotalPages int              `json:"totalPages"`
	}
	decode(t, h.do(t, "GET", "/api/blocks?page=1&pageSize=5", admin, nil, http.StatusOK), &page1)
	decode(t, h.do(t, "GET", "/api/blocks?page=2&pageSize=5", admin, nil, http.StatusOK), &page2)

	if len(page1.Items) != 5 || page1.PageSize != 5 {
		t.Fatalf("page 1 = %d items", len(page1.Items))
	}
	if page1.TotalPages < 2 {
		t.Fatalf("expected more than one page of blocks, got %d", page1.TotalPages)
	}
	if page1.Items[0]["code"] == page2.Items[0]["code"] {
		t.Error("page 2 repeated page 1")
	}
}

// ---------------------------------------------------------------- geometry

func TestBlockGeometryMatchesItsRegisteredArea(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")

	var block struct {
		Code           string  `json:"code"`
		BoundaryAreaHa float64 `json:"boundaryAreaHa"`
		Areas          areas   `json:"areas"`
	}
	decode(t, h.do(t, "GET", "/api/blocks/1", admin, nil, http.StatusOK), &block)

	var geo struct {
		Geometry struct {
			Type        string          `json:"type"`
			Coordinates json.RawMessage `json:"coordinates"`
		} `json:"geometry"`
		BoundaryAreaHa float64 `json:"boundaryAreaHa"`
	}
	decode(t, h.do(t, "GET", "/api/blocks/1/geometry", admin, nil, http.StatusOK), &geo)

	if geo.Geometry.Type != "MultiPolygon" {
		t.Fatalf("boundary is %q, want MultiPolygon", geo.Geometry.Type)
	}
	// The registered figure was rounded to the nearest tenth of a hectare; the measurement should
	// still land within that.
	if math.Abs(geo.BoundaryAreaHa-block.Areas.Total) > 0.06 {
		t.Errorf("PostGIS measures %.4f ha, the register says %.4f ha", geo.BoundaryAreaHa, block.Areas.Total)
	}
}

func TestSettingABoundaryReturnsTheAreaPostGISMeasures(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")

	// A square of one hundredth of a degree near Lusaka: about 119 hectares.
	polygon := map[string]any{
		"type": "Polygon",
		"coordinates": [][][]float64{{
			{28.60, -15.60}, {28.61, -15.60}, {28.61, -15.61}, {28.60, -15.61}, {28.60, -15.60},
		}},
	}
	body := h.do(t, "PUT", "/api/blocks/2/geometry", admin, map[string]any{"geometry": polygon}, http.StatusOK)

	var out struct {
		BoundaryAreaHa float64 `json:"boundaryAreaHa"`
	}
	decode(t, body, &out)
	if out.BoundaryAreaHa < 100 || out.BoundaryAreaHa > 140 {
		t.Errorf("measured %.2f ha; a 0.01° square at 15.6°S is about 119 ha", out.BoundaryAreaHa)
	}

	// Put the original back so the rest of the suite sees the seeded estate.
	h.do(t, "PUT", "/api/blocks/2/geometry", admin, map[string]any{"geometry": map[string]any{
		"type": "Polygon",
		"coordinates": [][][]float64{{
			{28.2437, -15.474}, {28.2542, -15.474}, {28.2542, -15.4845}, {28.2437, -15.4845}, {28.2437, -15.474},
		}},
	}}, http.StatusOK)
}

func TestRubbishGeometryIsRefused(t *testing.T) {
	h := newHarness(t)
	body := h.do(t, "PUT", "/api/blocks/3/geometry", h.token(t, "admin"),
		map[string]any{"geometry": map[string]any{"type": "Polygon", "coordinates": "not coordinates"}},
		http.StatusBadRequest)
	if !strings.Contains(string(body), "INVALID_GEOMETRY") {
		t.Errorf("expected INVALID_GEOMETRY, got %s", body)
	}
}

// ---------------------------------------------------------------- validation, in the database

func TestTheDatabaseRefusesMoreCaneThanTheBlockCanHold(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")

	var block struct {
		Areas areas `json:"areas"`
	}
	decode(t, h.do(t, "GET", "/api/blocks/4", admin, nil, http.StatusOK), &block)

	body := h.do(t, "POST", "/api/planting", admin, map[string]any{
		"blockId": 4, "cropSeasonId": 2, "cropYear": 2030, "plantingYear": 2030,
		"plantingType": "NewPlanting", "plannedAreaHa": 0,
		"actualAreaHa": block.Areas.Plantable + 50,
	}, http.StatusUnprocessableEntity)

	if !strings.Contains(string(body), "VALIDATION_FAILED") {
		t.Fatalf("expected the area rule to reject this, got %s", body)
	}
}

func TestPlantableAreaCannotExceedTheTotal(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")

	var block map[string]any
	decode(t, h.do(t, "GET", "/api/blocks/5", admin, nil, http.StatusOK), &block)

	body := h.do(t, "PUT", "/api/blocks/5", admin, map[string]any{
		"zoneId": int(block["zoneId"].(float64)), "code": block["code"], "name": block["name"],
		"totalAreaHa": 100.0, "plantableAreaHa": 150.0,
		"landStatus": block["landStatus"], "caneStatus": block["caneStatus"],
		"version": int(block["version"].(float64)),
	}, http.StatusUnprocessableEntity)

	if !strings.Contains(string(body), "plantableAreaHa") {
		t.Fatalf("expected a message against plantableAreaHa, got %s", body)
	}
}

func TestShrinkingABlockBelowItsCaneIsRefused(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")

	var block map[string]any
	decode(t, h.do(t, "GET", "/api/blocks/6", admin, nil, http.StatusOK), &block)
	withCane := block["areas"].(map[string]any)["areaWithCaneHa"].(float64)
	if withCane <= 0 {
		t.Skip("block 6 carries no cane in the sample data")
	}

	h.do(t, "PUT", "/api/blocks/6", admin, map[string]any{
		"zoneId": int(block["zoneId"].(float64)), "code": block["code"], "name": block["name"],
		"totalAreaHa": withCane - 1, "plantableAreaHa": withCane - 1,
		"landStatus": block["landStatus"], "caneStatus": block["caneStatus"],
		"version": int(block["version"].(float64)),
	}, http.StatusUnprocessableEntity)
}

func TestOptimisticConcurrency(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")

	var farm map[string]any
	decode(t, h.do(t, "GET", "/api/farms/2", admin, nil, http.StatusOK), &farm)
	version := int(farm["version"].(float64))

	update := map[string]any{
		"plantationId": int(farm["plantationId"].(float64)),
		"code":         farm["code"], "name": farm["name"], "version": version,
	}
	h.do(t, "PUT", "/api/farms/2", admin, update, http.StatusOK)

	// The second caller still holds the version they read before the first save.
	body := h.do(t, "PUT", "/api/farms/2", admin, update, http.StatusConflict)
	if !strings.Contains(string(body), "CONCURRENCY_CONFLICT") {
		t.Fatalf("expected CONCURRENCY_CONFLICT, got %s", body)
	}
}

func TestAMissingRowIsANotFoundNotAConflict(t *testing.T) {
	h := newHarness(t)
	h.do(t, "GET", "/api/farms/99999", h.token(t, "admin"), nil, http.StatusNotFound)
}

// ---------------------------------------------------------------- audit

func TestAChangeIsAudited(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")

	var farm map[string]any
	decode(t, h.do(t, "GET", "/api/farms/3", admin, nil, http.StatusOK), &farm)
	marker := fmt.Sprintf("Audited at %d", time.Now().UnixNano())

	h.do(t, "PUT", "/api/farms/3", admin, map[string]any{
		"plantationId": int(farm["plantationId"].(float64)),
		"code":         farm["code"], "name": farm["name"], "remark": marker,
		"version": int(farm["version"].(float64)),
	}, http.StatusOK)

	var log struct {
		Items []struct {
			Actor  string `json:"actor"`
			Action string `json:"action"`
			Entity string `json:"entity"`
			After  struct {
				Remark string `json:"remark"`
			} `json:"after"`
		} `json:"items"`
	}
	decode(t, h.do(t, "GET", "/api/audit?entity=Farm&pageSize=5", admin, nil, http.StatusOK), &log)

	for _, entry := range log.Items {
		if entry.After.Remark == marker {
			if entry.Actor != "admin" || entry.Action != "Update" {
				t.Errorf("audit entry = %+v", entry)
			}
			return
		}
	}
	t.Fatalf("the change was not recorded in the audit trail: %+v", log.Items)
}

// ---------------------------------------------------------------- planning versus actual

func TestPlanVersusActual(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")

	var out struct {
		Planned     areas   `json:"plannedAreas"`
		Actual      areas   `json:"actualAreas"`
		Variance    areas   `json:"varianceAreas"`
		Achievement float64 `json:"achievementPercent"`
		Rows        []struct {
			Code        string  `json:"code"`
			Planned     float64 `json:"plannedAreaWithCaneHa"`
			Actual      float64 `json:"actualAreaWithCaneHa"`
			Variance    float64 `json:"varianceAreaHa"`
			Achievement float64 `json:"achievementPercent"`
		} `json:"rows"`
		Monthly []struct {
			Month  int     `json:"month"`
			Actual float64 `json:"actualAreaWithCaneHa"`
		} `json:"monthly"`
	}
	decode(t, h.do(t, "GET", "/api/reports/planting-plan-vs-actual?cropYear=2026&level=farm",
		admin, nil, http.StatusOK), &out)

	if !near(out.Variance.WithCane, out.Actual.WithCane-out.Planned.WithCane) {
		t.Errorf("variance %v != actual %v - planned %v",
			out.Variance.WithCane, out.Actual.WithCane, out.Planned.WithCane)
	}
	if out.Planned.WithCane > 0 {
		want := math.Round(out.Actual.WithCane/out.Planned.WithCane*10000) / 100
		if !near(out.Achievement, want) {
			t.Errorf("achievement %v, want %v", out.Achievement, want)
		}
	}
	if len(out.Rows) == 0 {
		t.Fatal("no rows")
	}

	var rowsActual float64
	for _, r := range out.Rows {
		if !near(r.Variance, r.Actual-r.Planned) {
			t.Errorf("%s: variance %v != %v - %v", r.Code, r.Variance, r.Actual, r.Planned)
		}
		rowsActual += r.Actual
	}
	if !near(rowsActual, out.Actual.WithCane) {
		t.Errorf("the rows sum to %v but the header says %v", rowsActual, out.Actual.WithCane)
	}

	if len(out.Monthly) == 0 {
		t.Error("the monthly breakdown is empty; the progress chart would have nothing to draw")
	}
	for _, m := range out.Monthly {
		if m.Month < 1 || m.Month > 12 {
			t.Errorf("month %d is not a month", m.Month)
		}
	}
}

func TestChartsCoverEverySeriesTheDashboardDraws(t *testing.T) {
	h := newHarness(t)
	var series []struct {
		Key    string `json:"key"`
		Points []struct {
			Category string  `json:"category"`
			Value    float64 `json:"value"`
		} `json:"points"`
	}
	decode(t, h.do(t, "GET", "/api/dashboard/charts?cropYear=2026", h.token(t, "admin"), nil, http.StatusOK), &series)

	want := map[string]bool{
		"landUtilisation": false, "caneArea": false, "areaByFarm": false,
		"areaByZone": false, "monthlyProgress": false,
	}
	for _, s := range series {
		if _, ok := want[s.Key]; ok {
			want[s.Key] = true
			if len(s.Points) == 0 {
				t.Errorf("series %s has no points", s.Key)
			}
		}
	}
	for key, found := range want {
		if !found {
			t.Errorf("series %s is missing", key)
		}
	}
}

// ---------------------------------------------------------------- export

func TestExcelExportIsAWorkbookWithEveryRow(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")

	req, _ := http.NewRequest("GET", h.server.URL+"/api/reports/farm-area-tree.xlsx?cropYear=2026", nil)
	req.Header.Set("Authorization", "Bearer "+admin)
	resp, err := h.server.Client().Do(req)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("export = %d", resp.StatusCode)
	}
	if disp := resp.Header.Get("Content-Disposition"); !strings.Contains(disp, ".xlsx") {
		t.Errorf("Content-Disposition = %q; without a file name the browser saves it as the URL", disp)
	}

	body, _ := io.ReadAll(resp.Body)
	// An xlsx is a zip; if the bytes do not open as one, the file is broken however plausible its
	// length looks.
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("the export is not a readable workbook: %v", err)
	}
	var hasSheet bool
	for _, f := range reader.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/") {
			hasSheet = true
		}
	}
	if !hasSheet {
		t.Error("the workbook contains no worksheet")
	}
}

// ---------------------------------------------------------------- CORS

func TestCORSAllowsTheDashboardOriginAndRefusesOthers(t *testing.T) {
	h := newHarness(t)

	req, _ := http.NewRequest("OPTIONS", h.server.URL+"/api/dashboard/farm-area", nil)
	req.Header.Set("Origin", "http://localhost:8081")
	req.Header.Set("Access-Control-Request-Method", "GET")
	resp, err := h.server.Client().Do(req)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	defer resp.Body.Close()
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:8081" {
		t.Errorf("allow-origin = %q", got)
	}
	if got := resp.Header.Get("Access-Control-Expose-Headers"); !strings.Contains(got, "Content-Disposition") {
		t.Errorf("without Content-Disposition exposed the export downloads under the wrong name; got %q", got)
	}

	req2, _ := http.NewRequest("OPTIONS", h.server.URL+"/api/dashboard/farm-area", nil)
	req2.Header.Set("Origin", "https://somewhere.else")
	resp2, err := h.server.Client().Do(req2)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	defer resp2.Body.Close()
	if got := resp2.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("an unlisted origin was allowed: %q", got)
	}
}

// ---------------------------------------------------------------- audit columns

// Every business table records who created a row and when, and who last changed it and when. The
// times come from the database clock and the names from the signed-in user, so neither can be
// supplied by a caller.
func TestEveryTableRecordsWhoWroteItAndWhen(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	var missing []string
	rows, err := h.db.Pool().Query(ctx, `
		SELECT c.relname
		  FROM pg_class c
		  JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = 'public'
		 WHERE c.relkind = 'r'
		   -- The trail tables record the same two facts under the names at and actor, and are
		   -- append-only, so they are deliberately excluded.
		   AND c.relname NOT IN ('schema_migrations', 'spatial_ref_sys', 'audit_log', 'projection_approval')
		   AND NOT (
		       EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid = c.oid AND NOT a.attisdropped AND a.attname = 'created_at')
		   AND EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid = c.oid AND NOT a.attisdropped AND a.attname = 'created_by')
		   AND EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid = c.oid AND NOT a.attisdropped AND a.attname = 'updated_at')
		   AND EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid = c.oid AND NOT a.attisdropped AND a.attname = 'updated_by'))
		 ORDER BY c.relname`)
	if err != nil {
		t.Fatalf("inspect columns: %v", err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		missing = append(missing, name)
	}
	rows.Close()
	if len(missing) > 0 {
		t.Fatalf("these tables carry no audit columns: %s", strings.Join(missing, ", "))
	}

	// And every one of them is stamped by a trigger, not by whichever statement remembered to.
	var unstamped []string
	rows, err = h.db.Pool().Query(ctx, `
		SELECT c.relname
		  FROM pg_class c
		  JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = 'public'
		 WHERE c.relkind = 'r'
		   AND EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid = c.oid AND NOT a.attisdropped AND a.attname = 'created_by')
		   AND NOT EXISTS (SELECT 1 FROM pg_trigger g
		                    WHERE g.tgrelid = c.oid AND NOT g.tgisinternal AND g.tgname = 'stamp_' || c.relname)
		 ORDER BY c.relname`)
	if err != nil {
		t.Fatalf("inspect triggers: %v", err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		unstamped = append(unstamped, name)
	}
	rows.Close()
	if len(unstamped) > 0 {
		t.Fatalf("these tables have the columns but no stamp trigger: %s", strings.Join(unstamped, ", "))
	}
}

// The stamps name the signed-in user and cannot be set by the caller: a create by one person and
// an edit by another must leave the creation stamp on the first and the change stamp on the second,
// whatever the request bodies claimed.
func TestTheStampsNameTheCallerAndCannotBeForged(t *testing.T) {
	h := newHarness(t)
	code := fmt.Sprintf("STAMP-%d", time.Now().UnixNano()%100000)

	// The manager creates it, trying to backdate it and sign it as somebody else. Unknown fields
	// are refused outright, which is the first line of defence.
	h.do(t, "POST", "/api/varieties", h.token(t, "manager"), map[string]any{
		"code": code, "name": "Stamp test", "growingPeriodMonths": 12,
		"seedRatePerHa": 8, "expectedYieldPerHa": 90, "expectedLossPercent": 5,
		"createdBy": "somebody-else", "createdAt": "1999-01-01T00:00:00Z",
	}, http.StatusBadRequest)

	var created struct {
		ID      int `json:"id"`
		Version int `json:"version"`
	}
	decode(t, h.do(t, "POST", "/api/varieties", h.token(t, "manager"), map[string]any{
		"code": code, "name": "Stamp test", "growingPeriodMonths": 12,
		"seedRatePerHa": 8, "expectedYieldPerHa": 90, "expectedLossPercent": 5,
	}, http.StatusCreated), &created)

	var createdBy, updatedBy string
	var createdAt, updatedAt time.Time
	row := h.db.Pool().QueryRow(context.Background(),
		`SELECT created_by, updated_by, created_at, updated_at FROM cane_variety WHERE id = $1`, created.ID)
	if err := row.Scan(&createdBy, &updatedBy, &createdAt, &updatedAt); err != nil {
		t.Fatalf("read stamps: %v", err)
	}
	if createdBy != "manager" || updatedBy != "manager" {
		t.Fatalf("a new row should be stamped with its creator, got created_by=%s updated_by=%s",
			createdBy, updatedBy)
	}
	if time.Since(createdAt) > time.Minute || time.Since(createdAt) < 0 {
		t.Fatalf("created_at is not the database's own clock: %s", createdAt)
	}

	// The administrator edits it. The creation stamp must not move.
	h.do(t, "PUT", "/api/varieties/"+itoa(created.ID), h.token(t, "admin"), map[string]any{
		"code": code, "name": "Stamp test edited", "growingPeriodMonths": 12,
		"seedRatePerHa": 8, "expectedYieldPerHa": 90, "expectedLossPercent": 5,
		"version": created.Version,
	}, http.StatusOK)

	var createdBy2, updatedBy2 string
	var createdAt2, updatedAt2 time.Time
	row = h.db.Pool().QueryRow(context.Background(),
		`SELECT created_by, updated_by, created_at, updated_at FROM cane_variety WHERE id = $1`, created.ID)
	if err := row.Scan(&createdBy2, &updatedBy2, &createdAt2, &updatedAt2); err != nil {
		t.Fatalf("read stamps after the edit: %v", err)
	}
	if createdBy2 != "manager" || !createdAt2.Equal(createdAt) {
		t.Errorf("the creation stamp moved: %s at %s, was manager at %s", createdBy2, createdAt2, createdAt)
	}
	if updatedBy2 != "admin" {
		t.Errorf("the change stamp should name the editor, got %s", updatedBy2)
	}
	if !updatedAt2.After(updatedAt) {
		t.Errorf("updated_at did not move forward: %s then %s", updatedAt, updatedAt2)
	}

	_, _ = h.db.Pool().Exec(context.Background(), `DELETE FROM cane_variety WHERE id = $1`, created.ID)
}
