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

		master := repository.NewMasterRepository(db)
		dash := repository.NewDashboardRepository(db)
		support := repository.NewSupportRepository(db)
		activities := repository.NewActivityRepository(db)
		svc := service.New(db, master, dash, support, activities)
		tokens := auth.NewTokens("a-test-signing-key-that-is-long-enough", time.Hour)
		api := httpapi.New(svc, tokens, support, db, log)

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

	var mapData struct {
		Blocks struct {
			Features []struct {
				Properties map[string]any `json:"properties"`
			} `json:"features"`
		} `json:"blocks"`
	}
	decode(t, h.do(t, "GET", "/api/dashboard/farm-area/map?cropYear=2026&farmId=1", admin, nil, http.StatusOK), &mapData)
	if len(mapData.Blocks.Features) != oneFarm.BlockCount {
		t.Errorf("the map drew %d blocks, the cards counted %d", len(mapData.Blocks.Features), oneFarm.BlockCount)
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
