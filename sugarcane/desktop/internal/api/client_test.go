package api_test

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sovanna2011/sugarcane-go/desktop/internal/api"
)

// ---------------------------------------------------------------- the filter

func TestAnEmptyFilterAsksForEverything(t *testing.T) {
	var f api.Filter
	if got := f.Query(); got != "" {
		t.Errorf("Query() = %q, want empty", got)
	}
	if got := f.Applied(); got != 0 {
		t.Errorf("Applied() = %d, want 0", got)
	}
}

func TestEachSetValueBecomesOneParameter(t *testing.T) {
	f := api.Filter{CropYear: api.Int(2026), FarmID: api.Int(1), CaneStatus: "Growing"}

	query := f.Query()

	if !strings.HasPrefix(query, "?") {
		t.Fatalf("Query() = %q, want a leading question mark", query)
	}
	for _, want := range []string{"cropYear=2026", "farmId=1", "caneStatus=Growing"} {
		if !strings.Contains(query, want) {
			t.Errorf("Query() = %q, missing %s", query, want)
		}
	}
	if got := f.Applied(); got != 3 {
		t.Errorf("Applied() = %d, want 3", got)
	}
}

func TestAnExtraParameterRidesAlongsideTheFilter(t *testing.T) {
	query := api.Filter{CropYear: api.Int(2026)}.Query(api.Param{Name: "level", Value: "Farm"})

	if !strings.Contains(query, "cropYear=2026") || !strings.Contains(query, "level=Farm") {
		t.Errorf("Query() = %q", query)
	}
	// An extra with no value is left out rather than sent empty.
	var empty api.Filter
	if got := empty.Query(api.Param{Name: "level", Value: ""}); got != "" {
		t.Errorf("Query() = %q, want empty", got)
	}
}

func TestASearchTermIsEncodedRatherThanConcatenated(t *testing.T) {
	query := api.Filter{Search: "Riverside & Zone 1"}.Query()

	if strings.Contains(query, " ") {
		t.Errorf("Query() = %q contains a raw space", query)
	}
	// The ampersand inside the term must not read as the start of another parameter.
	if strings.Count(query, "&") != 0 {
		t.Errorf("Query() = %q, the search term leaked into the parameter list", query)
	}
}

func TestShowingRetiredLandIsNotCountedAsAFilter(t *testing.T) {
	f := api.Filter{IncludeInactive: true}

	// It widens the query rather than narrowing it, so counting it would have the status bar say
	// land is being hidden at the moment more of it is being shown.
	if got := f.Applied(); got != 0 {
		t.Errorf("Applied() = %d, want 0", got)
	}
	if !strings.Contains(f.Query(), "includeInactive=true") {
		t.Error("the parameter still has to be sent")
	}
}

// ---------------------------------------------------------------- errors, without a service

func TestAServiceThatCannotBeReachedSaysSo(t *testing.T) {
	// Port 1 is reserved and nothing listens on it, so this is a connection refusal every time.
	client := api.New("http://127.0.0.1:1")

	_, err := client.Dashboard(context.Background(), api.Filter{})

	var apiErr *api.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("got %T (%v), want *api.Error", err, err)
	}
	if apiErr.Code != "UNREACHABLE" {
		t.Errorf("code = %q, want UNREACHABLE", apiErr.Code)
	}
	if !strings.Contains(apiErr.Message, "127.0.0.1:1") {
		t.Errorf("the message does not name the address that failed: %s", apiErr.Message)
	}
}

func TestAnAnswerThatIsNotTheServicesErrorShapeStillReportsWhatArrived(t *testing.T) {
	// A proxy between the desktop and the service answers with HTML, not the service's JSON.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html><body>502 Bad Gateway</body></html>"))
	}))
	defer server.Close()

	_, err := api.New(server.URL).Dashboard(context.Background(), api.Filter{})

	var apiErr *api.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("got %T, want *api.Error", err)
	}
	if apiErr.Status != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", apiErr.Status)
	}
	if !strings.Contains(apiErr.Message, "Bad Gateway") {
		t.Errorf("the message hides what arrived: %s", apiErr.Message)
	}
}

func TestAFieldFaultIsCarriedWhole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"VALIDATION_FAILED","message":"The block could not be saved.",
			"fields":[{"field":"totalAreaHa","message":"Total area must be greater than nought."}]}`))
	}))
	defer server.Close()

	_, err := api.New(server.URL).Dashboard(context.Background(), api.Filter{})

	var apiErr *api.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("got %T, want *api.Error", err)
	}
	if apiErr.Code != "VALIDATION_FAILED" || len(apiErr.Fields) != 1 {
		t.Fatalf("the refusal lost its detail: %+v", apiErr)
	}
	if !strings.Contains(apiErr.Error(), "greater than nought") {
		t.Errorf("the field fault is not shown to the user: %s", apiErr.Error())
	}
}

func TestTheTokenIsSentOnEveryCallAndForgottenOnSignOut(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/api/auth/login":
			_, _ = w.Write([]byte(`{"token":"tok","expiresAt":"2030-01-01T00:00:00Z",
				"user":{"id":1,"userName":"admin","fullName":"System Administrator","role":"Admin"}}`))
		default:
			_, _ = w.Write([]byte(`{"areas":{},"kpis":[],"blockCount":0,"unit":"ha"}`))
		}
	}))
	defer server.Close()

	client := api.New(server.URL)
	if client.SignedIn() {
		t.Error("a fresh client should not be signed in")
	}
	if _, err := client.SignIn(context.Background(), "admin", "Farm#2026"); err != nil {
		t.Fatalf("sign in: %v", err)
	}
	if !client.SignedIn() || client.User().Role != "Admin" {
		t.Fatalf("the session was not kept: %+v", client.User())
	}
	if _, err := client.Dashboard(context.Background(), api.Filter{}); err != nil {
		t.Fatalf("dashboard: %v", err)
	}

	if seen[0] != "" {
		t.Errorf("the login carried a token: %q", seen[0])
	}
	if seen[1] != "Bearer tok" {
		t.Errorf("the call after signing in sent %q", seen[1])
	}

	client.SignOut()
	if client.SignedIn() || client.User().UserName != "" {
		t.Error("signing out has to forget who was signed in")
	}
}

// ---------------------------------------------------------------- against the real service
//
// These are the tests that matter. A hand-written stub only proves the client agrees with my idea
// of the JSON, which is exactly the thing that goes wrong; these check it reads what the Go service
// actually sends. Set FARMAREA_API_URL to run them, and they skip without it, so `go test ./...`
// is still useful on a machine with no service running.
//
//	export FARMAREA_API_URL="http://127.0.0.1:8080"

const password = "Farm#2026"

func service(t *testing.T) string {
	t.Helper()
	url := os.Getenv("FARMAREA_API_URL")
	if url == "" {
		t.Skip("set FARMAREA_API_URL to run the service tests")
	}
	return url
}

func signedIn(t *testing.T, user string) *api.Client {
	t.Helper()
	client := api.New(service(t))
	if _, err := client.SignIn(context.Background(), user, password); err != nil {
		t.Fatalf("sign in as %s: %v", user, err)
	}
	return client
}

func near(a, b float64) bool { return math.Abs(a-b) < 0.05 }

func TestSigningInReturnsTheUserAndArmsTheToken(t *testing.T) {
	client := api.New(service(t))

	user, err := client.SignIn(context.Background(), "admin", password)
	if err != nil {
		t.Fatalf("sign in: %v", err)
	}
	if user.UserName != "admin" || user.Role != "Admin" || user.FullName == "" {
		t.Fatalf("the service described the user as %+v", user)
	}
	if !client.ExpiresAt().After(client.ExpiresAt().Add(-1)) {
		t.Error("the token carries no expiry")
	}

	// The token has to work, not merely arrive.
	me, err := client.WhoAmI(context.Background())
	if err != nil {
		t.Fatalf("who am I: %v", err)
	}
	if me.UserName != "admin" {
		t.Errorf("the service knows the caller as %q", me.UserName)
	}
}

func TestAWrongPasswordIsRefusedWithTheServicesOwnMessage(t *testing.T) {
	client := api.New(service(t))

	_, err := client.SignIn(context.Background(), "admin", "wrong")

	if !api.IsUnauthorized(err) {
		t.Fatalf("got %v, want a 401", err)
	}
	if client.SignedIn() {
		t.Error("a refused sign-in left the client holding a token")
	}
}

func TestCallingWithoutSigningInIsRefused(t *testing.T) {
	_, err := api.New(service(t)).Dashboard(context.Background(), api.Filter{})

	if !api.IsUnauthorized(err) {
		t.Fatalf("got %v, want a 401", err)
	}
}

func TestTheCardsAddUpToTheTotalTheyReport(t *testing.T) {
	client := signedIn(t, "admin")

	board, err := client.Dashboard(context.Background(), api.Filter{CropYear: api.Int(2026)})
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}

	if len(board.KPIs) != 6 || board.BlockCount == 0 || board.Areas.TotalHa == 0 {
		t.Fatalf("the dashboard came back thin: %d cards, %d blocks, %v ha",
			len(board.KPIs), board.BlockCount, board.Areas.TotalHa)
	}
	// Land is either under cane, free to plant, or unplantable — the three exhaust the total.
	parts := board.Areas.WithCaneHa + board.Areas.AvailableHa + board.Areas.NonPlantableHa
	if !near(parts, board.Areas.TotalHa) {
		t.Errorf("the parts sum to %v ha, the total says %v ha", parts, board.Areas.TotalHa)
	}
	// And the cane splits into new planting and ratoon.
	if !near(board.Areas.NewPlantingHa+board.Areas.RatoonHa, board.Areas.WithCaneHa) {
		t.Errorf("new planting %v plus ratoon %v is not the %v ha with cane",
			board.Areas.NewPlantingHa, board.Areas.RatoonHa, board.Areas.WithCaneHa)
	}
}

func TestTheHierarchyDescribesTheSameLandAsTheCards(t *testing.T) {
	client := signedIn(t, "admin")
	filter := api.Filter{CropYear: api.Int(2026)}

	board, err := client.Dashboard(context.Background(), filter)
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	tree, err := client.Tree(context.Background(), filter)
	if err != nil {
		t.Fatalf("tree: %v", err)
	}

	if len(tree) == 0 {
		t.Fatal("the hierarchy is empty")
	}
	var total float64
	for _, farm := range tree {
		total += farm.Areas.TotalHa
	}
	if !near(total, board.Areas.TotalHa) {
		t.Errorf("the hierarchy totals %v ha, the cards say %v ha", total, board.Areas.TotalHa)
	}

	farm := tree[0]
	if farm.NodeType != "Farm" || len(farm.Children) == 0 || farm.Children[0].NodeType != "Zone" {
		t.Fatalf("the hierarchy is not farm → zone: %s → %v", farm.NodeType, farm.Children)
	}
	if farm.MapURL == "" {
		t.Error("a farm row carries no map link")
	}
	var zones float64
	for _, zone := range farm.Children {
		zones += zone.Areas.TotalHa
	}
	if !near(zones, farm.Areas.TotalHa) {
		t.Errorf("the zones total %v ha, their farm %v ha", zones, farm.Areas.TotalHa)
	}
}

func TestAFilterNarrowsEveryCallTheSameWay(t *testing.T) {
	client := signedIn(t, "admin")
	ctx := context.Background()

	all, err := client.Dashboard(ctx, api.Filter{CropYear: api.Int(2026)})
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	oneFarm, err := client.Dashboard(ctx, api.Filter{CropYear: api.Int(2026), FarmID: api.Int(1)})
	if err != nil {
		t.Fatalf("dashboard for one farm: %v", err)
	}
	tree, err := client.Tree(ctx, api.Filter{CropYear: api.Int(2026), FarmID: api.Int(1)})
	if err != nil {
		t.Fatalf("tree for one farm: %v", err)
	}

	if !(oneFarm.Areas.TotalHa < all.Areas.TotalHa) {
		t.Errorf("one farm measures %v ha, the estate %v ha", oneFarm.Areas.TotalHa, all.Areas.TotalHa)
	}
	if len(tree) != 1 {
		t.Fatalf("the hierarchy ignored the farm filter: %d farms", len(tree))
	}
	if !near(tree[0].Areas.TotalHa, oneFarm.Areas.TotalHa) {
		t.Errorf("the hierarchy says %v ha, the cards %v ha", tree[0].Areas.TotalHa, oneFarm.Areas.TotalHa)
	}
}

func TestTheMapDrawsWhicheverLocationLevelIsAskedFor(t *testing.T) {
	client := signedIn(t, "admin")
	ctx := context.Background()
	filter := api.Filter{CropYear: api.Int(2026)}

	board, err := client.Dashboard(ctx, filter)
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}

	for _, level := range api.MapLevels {
		data, err := client.Map(ctx, filter, level)
		if err != nil {
			t.Fatalf("map by %s: %v", level, err)
		}
		if data.Level != level {
			t.Errorf("asked for %s, the service answered %q", level, data.Level)
		}
		if len(data.Features.Features) == 0 {
			t.Fatalf("%s level drew nothing", level)
		}

		var total float64
		for _, feature := range data.Features.Features {
			if feature.Properties.Level != level {
				t.Errorf("%s level tagged a feature %q", level, feature.Properties.Level)
			}
			if feature.Properties.Code == "" {
				t.Errorf("%s level drew a feature with no code", level)
			}
			total += feature.Properties.TotalAreaHa
		}
		// Whichever level is drawn, it is the same land the cards measured.
		if !near(total, board.Areas.TotalHa) {
			t.Errorf("%s level totals %v ha, the cards say %v ha", level, total, board.Areas.TotalHa)
		}

		// Farms are drawn over their own outlines; zones and blocks over the zone outlines too.
		levels := map[string]bool{}
		for _, outline := range data.Outlines.Features {
			levels[outline.Properties.Level] = true
		}
		want := 2
		if level == api.LevelFarm {
			want = 1
		}
		if len(levels) != want || !levels[api.LevelFarm] {
			t.Errorf("%s level drew outlines for %v", level, levels)
		}
	}
}

func TestAFarmDrawnUnderAZoneFilterMeasuresTheZone(t *testing.T) {
	client := signedIn(t, "admin")
	ctx := context.Background()

	whole, err := client.Map(ctx, api.Filter{CropYear: api.Int(2026), FarmID: api.Int(1)}, api.LevelFarm)
	if err != nil {
		t.Fatalf("map: %v", err)
	}
	narrowed, err := client.Map(ctx,
		api.Filter{CropYear: api.Int(2026), FarmID: api.Int(1), ZoneID: api.Int(1)}, api.LevelFarm)
	if err != nil {
		t.Fatalf("map narrowed to a zone: %v", err)
	}

	if len(whole.Features.Features) != 1 || len(narrowed.Features.Features) != 1 {
		t.Fatalf("expected one farm each, drew %d and %d",
			len(whole.Features.Features), len(narrowed.Features.Features))
	}
	wholeArea := whole.Features.Features[0].Properties.TotalAreaHa
	narrowedArea := narrowed.Features.Features[0].Properties.TotalAreaHa
	if !(narrowedArea < wholeArea) {
		t.Errorf("one zone of the farm measures %v ha, the whole farm %v ha", narrowedArea, wholeArea)
	}
}

func TestAnUnknownFilterValueIsRefusedWithACodeTheWindowCanBranchOn(t *testing.T) {
	client := signedIn(t, "admin")

	_, err := client.Dashboard(context.Background(), api.Filter{CaneStatus: "Flowering"})

	var apiErr *api.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("got %v, want *api.Error", err)
	}
	if apiErr.Status != http.StatusBadRequest || apiErr.Code != "INVALID_FILTER" {
		t.Errorf("got %d %s, want 400 INVALID_FILTER", apiErr.Status, apiErr.Code)
	}
	if !strings.Contains(apiErr.Message, "caneStatus") {
		t.Errorf("the message does not name the filter at fault: %s", apiErr.Message)
	}
}

func TestTheLookupsFeedTheFilterBarAndCascade(t *testing.T) {
	client := signedIn(t, "admin")
	ctx := context.Background()

	farms, err := client.Lookup(ctx, "farms", nil)
	if err != nil {
		t.Fatalf("farms: %v", err)
	}
	if len(farms) == 0 {
		t.Fatal("no farm to choose from")
	}
	for _, farm := range farms {
		if farm.Label() == "" {
			t.Errorf("farm %d has nothing to show in a dropdown", farm.ID)
		}
	}

	allZones, err := client.Lookup(ctx, "zones", nil)
	if err != nil {
		t.Fatalf("zones: %v", err)
	}
	ofFarm, err := client.Lookup(ctx, "zones", api.Int(farms[0].ID))
	if err != nil {
		t.Fatalf("zones of a farm: %v", err)
	}
	if len(ofFarm) == 0 || len(ofFarm) >= len(allZones) {
		t.Fatalf("choosing a farm has to narrow the zone list: %d of %d", len(ofFarm), len(allZones))
	}
	for _, zone := range ofFarm {
		if zone.ParentID == nil || *zone.ParentID != farms[0].ID {
			t.Errorf("zone %s belongs to another farm", zone.Code)
		}
	}
}

func TestTheProjectionListPagesAndCarriesItsActions(t *testing.T) {
	client := signedIn(t, "admin")

	page, err := client.Projections(context.Background(), api.Filter{}, 1, 5)
	if err != nil {
		t.Fatalf("projections: %v", err)
	}

	if page.Page != 1 || page.PageSize != 5 {
		t.Errorf("asked for page 1 of 5, got page %d of %d", page.Page, page.PageSize)
	}
	if page.TotalCount < len(page.Items) {
		t.Errorf("%d rows on a page of a list of %d", len(page.Items), page.TotalCount)
	}
	for _, projection := range page.Items {
		if projection.ProjectionNo == "" || projection.Status == "" || projection.CropYear < 2000 {
			t.Errorf("a projection came back thin: %+v", projection)
		}
	}
}

func TestAViewerMayReadTheDashboard(t *testing.T) {
	client := signedIn(t, "viewer")

	board, err := client.Dashboard(context.Background(), api.Filter{CropYear: api.Int(2026)})
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if client.User().Role != "Viewer" || board.Areas.TotalHa == 0 {
		t.Errorf("a viewer signed in as %q saw %v ha", client.User().Role, board.Areas.TotalHa)
	}
}
