package httpapi_test

// The planning master data over the real stack: growing seasons, cane varieties, and the activity
// chain the schedule engine walks. Skips without FARMAREA_TEST_DATABASE_URL, like the rest.

import (
	"net/http"
	"strings"
	"testing"
)

type activity struct {
	ID                      int     `json:"id"`
	Code                    string  `json:"code"`
	Name                    string  `json:"name"`
	Category                string  `json:"category"`
	ApplicableCropType      string  `json:"applicableCropType"`
	SequenceNo              int     `json:"sequenceNo"`
	StandardStartDayOffset  int     `json:"standardStartDayOffset"`
	StandardCapacityPerDay  float64 `json:"standardCapacityPerDay"`
	StandardLabourDaysPerHa float64 `json:"standardLabourDaysPerHa"`
	IsMandatory             bool    `json:"isMandatory"`
	RequiresTractor         bool    `json:"requiresTractor"`
	RequiresEquipment       bool    `json:"requiresEquipment"`
	RequiresLabour          bool    `json:"requiresLabour"`
	Version                 int     `json:"version"`
	Dependencies            []struct {
		ID                  int    `json:"id"`
		DependsOnID         int    `json:"dependsOnId"`
		DependsOnCode       string `json:"dependsOnCode"`
		DependsOnSequenceNo int    `json:"dependsOnSequenceNo"`
		LagDays             int    `json:"lagDays"`
		IsBlocking          bool   `json:"isBlocking"`
	} `json:"dependencies"`
}

func listActivities(t *testing.T, h *harness, query string) []activity {
	t.Helper()
	var page struct {
		Items      []activity `json:"items"`
		TotalCount int        `json:"totalCount"`
	}
	decode(t, h.do(t, "GET", "/api/activities"+query, h.token(t, "admin"), nil, http.StatusOK), &page)
	return page.Items
}

func TestTheNineteenSampleActivitiesAreSeededInSequence(t *testing.T) {
	h := newHarness(t)
	activities := listActivities(t, h, "?pageSize=100")

	if len(activities) != 19 {
		t.Fatalf("expected the specification's nineteen activities, got %d", len(activities))
	}
	for i, a := range activities {
		if a.SequenceNo != i+1 {
			t.Errorf("activity %s is at sequence %d, expected %d — the list is not in field order",
				a.Code, a.SequenceNo, i+1)
		}
	}

	first, last := activities[0], activities[18]
	if first.Code != "A001" || first.StandardStartDayOffset != -45 {
		t.Errorf("the survey should start 45 days before planting: %+v", first)
	}
	if last.Code != "A019" || last.StandardStartDayOffset != 65 {
		t.Errorf("the last herbicide should be 65 days after planting: %+v", last)
	}

	// Planting itself is the anchor: offset zero, and the whole programme is relative to it.
	var planting activity
	for _, a := range activities {
		if a.Code == "A012" {
			planting = a
		}
	}
	if planting.StandardStartDayOffset != 0 || planting.Category != "Planting" {
		t.Errorf("planting is the anchor of the programme: %+v", planting)
	}
}

func TestEveryActivityAfterTheFirstWaitsForTheOneBeforeIt(t *testing.T) {
	h := newHarness(t)
	activities := listActivities(t, h, "?pageSize=100")

	bySequence := map[int]activity{}
	for _, a := range activities {
		bySequence[a.SequenceNo] = a
	}

	for _, a := range activities {
		if a.SequenceNo == 1 {
			if len(a.Dependencies) != 0 {
				t.Errorf("the first activity waits for nothing, got %d dependencies", len(a.Dependencies))
			}
			continue
		}
		if len(a.Dependencies) != 1 {
			t.Errorf("%s has %d dependencies, expected one", a.Code, len(a.Dependencies))
			continue
		}
		dep := a.Dependencies[0]
		if dep.DependsOnSequenceNo != a.SequenceNo-1 {
			t.Errorf("%s waits for sequence %d, expected %d", a.Code, dep.DependsOnSequenceNo, a.SequenceNo-1)
		}
		// Planting waits a day after the furrows are opened.
		wantLag := 0
		if a.Code == "A012" {
			wantLag = 1
		}
		if dep.LagDays != wantLag {
			t.Errorf("%s waits %d days, expected %d", a.Code, dep.LagDays, wantLag)
		}
		// An optional activity only warns; the programme can proceed past it.
		if dep.IsBlocking != a.IsMandatory {
			t.Errorf("%s: blocking=%v but mandatory=%v — an optional step must not stop the plan",
				a.Code, dep.IsBlocking, a.IsMandatory)
		}
	}
}

func TestActivitiesCanBeFilteredByCropTypeAndCategory(t *testing.T) {
	h := newHarness(t)

	ratoon := listActivities(t, h, "?pageSize=100&plantingType=Ratoon")
	if len(ratoon) == 0 || len(ratoon) >= 19 {
		t.Fatalf("a ratoon crop skips the land preparation a new planting needs, got %d of 19", len(ratoon))
	}
	for _, a := range ratoon {
		if a.ApplicableCropType == "NewPlanting" {
			t.Errorf("%s applies only to new planting but came back under a ratoon filter", a.Code)
		}
	}

	prep := listActivities(t, h, "?pageSize=100&activityCategory=LandPreparation")
	if len(prep) == 0 {
		t.Fatal("no land-preparation activities")
	}
	for _, a := range prep {
		if a.Category != "LandPreparation" {
			t.Errorf("%s is %s, not land preparation", a.Code, a.Category)
		}
	}
}

func TestTheActivityChainIsReturnedInOrder(t *testing.T) {
	h := newHarness(t)
	var chain []activity
	decode(t, h.do(t, "GET", "/api/activities/chain?cropType=Both", h.token(t, "admin"), nil, http.StatusOK), &chain)

	if len(chain) == 0 {
		t.Fatal("the chain is empty")
	}
	for i := 1; i < len(chain); i++ {
		if chain[i].SequenceNo < chain[i-1].SequenceNo {
			t.Fatalf("the chain is out of order at %s", chain[i].Code)
		}
	}
}

// ---------------------------------------------------------------- the rules

func TestADependencyThatWouldCloseALoopIsRefused(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")
	activities := listActivities(t, h, "?pageSize=100")

	first, last := activities[0], activities[len(activities)-1]

	// The last activity already comes after the first through the whole chain, so making the first
	// wait for the last would make the schedule unsatisfiable.
	body := h.do(t, "POST", "/api/activities/dependencies", admin, map[string]any{
		"activityId": first.ID, "dependsOnId": last.ID, "lagDays": 0, "isBlocking": true,
	}, http.StatusUnprocessableEntity)

	if !strings.Contains(string(body), "loop") {
		t.Fatalf("expected a message about closing a loop, got %s", body)
	}
	if !strings.Contains(string(body), last.Code) || !strings.Contains(string(body), first.Code) {
		t.Errorf("the message should name both activities, got %s", body)
	}
}

func TestAnActivityCannotWaitForItself(t *testing.T) {
	h := newHarness(t)
	activities := listActivities(t, h, "?pageSize=5")

	body := h.do(t, "POST", "/api/activities/dependencies", h.token(t, "admin"), map[string]any{
		"activityId": activities[0].ID, "dependsOnId": activities[0].ID,
	}, http.StatusUnprocessableEntity)
	if !strings.Contains(string(body), "itself") {
		t.Errorf("expected a message about waiting for itself, got %s", body)
	}
}

func TestANegativeLagIsRefused(t *testing.T) {
	h := newHarness(t)
	activities := listActivities(t, h, "?pageSize=5")

	h.do(t, "POST", "/api/activities/dependencies", h.token(t, "admin"), map[string]any{
		"activityId": activities[2].ID, "dependsOnId": activities[0].ID, "lagDays": -3,
	}, http.StatusUnprocessableEntity)
}

func TestAnActivityThatNeedsAMachineMustHaveADailyCapacity(t *testing.T) {
	h := newHarness(t)

	body := h.do(t, "POST", "/api/activities", h.token(t, "admin"), map[string]any{
		"companyId": 1, "code": "TEST-NOCAP", "name": "Needs a tractor but has no capacity",
		"category": "LandPreparation", "applicableCropType": "Both", "sequenceNo": 900,
		"requiresTractor": true, "standardCapacityPerDay": 0,
	}, http.StatusUnprocessableEntity)

	if !strings.Contains(string(body), "standardCapacityPerDay") {
		t.Fatalf("expected the message on the capacity field, got %s", body)
	}
	if !strings.Contains(string(body), "how long it takes") {
		t.Errorf("the message should say why it matters, got %s", body)
	}
}

func TestAnUnknownCategoryIsRefusedBeforeItReachesTheEnum(t *testing.T) {
	h := newHarness(t)
	body := h.do(t, "POST", "/api/activities", h.token(t, "admin"), map[string]any{
		"companyId": 1, "code": "TEST-CAT", "name": "Unknown category",
		"category": "Ploughing", "sequenceNo": 901,
	}, http.StatusBadRequest)
	if !strings.Contains(string(body), "INVALID_CATEGORY") {
		t.Errorf("expected INVALID_CATEGORY, got %s", body)
	}
}

func TestAPlannerMayNotChangeTheActivityMaster(t *testing.T) {
	h := newHarness(t)
	// A planner builds projections; the activity master is the estate's standing method, and
	// changing it changes every plan generated afterwards.
	h.do(t, "POST", "/api/activities", h.token(t, "planner"), map[string]any{
		"companyId": 1, "code": "TEST-ROLE", "name": "Should not be created",
		"category": "Other", "sequenceNo": 902,
	}, http.StatusForbidden)
}

// ---------------------------------------------------------------- varieties and seasons

func TestVarietiesCarryTheFiguresTheProjectionFormulasRead(t *testing.T) {
	h := newHarness(t)
	var page struct {
		Items []struct {
			Code                string  `json:"code"`
			SeedRatePerHa       float64 `json:"seedRatePerHa"`
			ExpectedYieldPerHa  float64 `json:"expectedYieldPerHa"`
			ExpectedLossPercent float64 `json:"expectedLossPercent"`
			GrowingPeriodMonths int     `json:"growingPeriodMonths"`
		} `json:"items"`
	}
	decode(t, h.do(t, "GET", "/api/varieties", h.token(t, "admin"), nil, http.StatusOK), &page)

	if len(page.Items) == 0 {
		t.Fatal("no varieties")
	}
	for _, v := range page.Items {
		if v.SeedRatePerHa <= 0 {
			t.Errorf("%s has no seed rate, so its seed-cane requirement would be zero", v.Code)
		}
		if v.ExpectedLossPercent < 0 || v.ExpectedLossPercent > 100 {
			t.Errorf("%s has an impossible loss of %v%%", v.Code, v.ExpectedLossPercent)
		}
		if v.GrowingPeriodMonths < 1 {
			t.Errorf("%s has no growing period, so its harvest date cannot be derived", v.Code)
		}
	}
}

func TestAnImpossibleVarietyIsRefused(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")

	body := h.do(t, "POST", "/api/varieties", admin, map[string]any{
		"code": "TEST-BAD", "name": "Impossible", "growingPeriodMonths": 12,
		"seedRatePerHa": 0, "expectedYieldPerHa": -5, "expectedLossPercent": 150,
	}, http.StatusUnprocessableEntity)

	for _, field := range []string{"seedRatePerHa", "expectedYieldPerHa", "expectedLossPercent"} {
		if !strings.Contains(string(body), field) {
			t.Errorf("every problem should be reported at once; %s is missing from %s", field, body)
		}
	}
}

func TestSeasonsCarryTheirPlantingWindow(t *testing.T) {
	h := newHarness(t)
	var page struct {
		Items []struct {
			Code                string  `json:"code"`
			CropYear            int     `json:"cropYear"`
			PlantingWindowStart *string `json:"plantingWindowStart"`
			PlantingWindowEnd   *string `json:"plantingWindowEnd"`
			Status              string  `json:"status"`
		} `json:"items"`
	}
	decode(t, h.do(t, "GET", "/api/seasons", h.token(t, "admin"), nil, http.StatusOK), &page)

	if len(page.Items) < 2 {
		t.Fatalf("expected the two seeded seasons, got %d", len(page.Items))
	}
	for _, s := range page.Items {
		if s.PlantingWindowStart == nil || s.PlantingWindowEnd == nil {
			t.Errorf("%s has no planting window, so no projection can be checked against it", s.Code)
		}
		if s.Status == "" {
			t.Errorf("%s has no status", s.Code)
		}
	}
	// Newest first: a planner opens the dashboard on the season being managed.
	if page.Items[0].CropYear < page.Items[1].CropYear {
		t.Error("seasons should be listed newest first")
	}
}

func TestASeasonWindowCannotEndBeforeItStarts(t *testing.T) {
	h := newHarness(t)
	h.do(t, "POST", "/api/seasons", h.token(t, "admin"), map[string]any{
		"code": "TEST-WIN", "name": "Backwards window", "cropYear": 2027,
		"startsOn": "2027-03-01", "endsOn": "2028-02-29",
		"plantingWindowStart": "2027-06-01", "plantingWindowEnd": "2027-04-01",
	}, http.StatusUnprocessableEntity)
}

func TestSavingAnActivityIsAudited(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")
	activities := listActivities(t, h, "?pageSize=100")

	// Take the last one so no other test's expectations about the chain are disturbed.
	target := activities[len(activities)-1]
	h.do(t, "PUT", "/api/activities/"+itoa(target.ID), admin, map[string]any{
		"companyId": 1, "code": target.Code, "name": target.Name,
		"category": target.Category, "applicableCropType": target.ApplicableCropType,
		"sequenceNo": target.SequenceNo, "standardStartDayOffset": target.StandardStartDayOffset,
		"standardCapacityPerDay":  target.StandardCapacityPerDay,
		"standardLabourDaysPerHa": target.StandardLabourDaysPerHa,
		"isMandatory":             target.IsMandatory, "requiresTractor": target.RequiresTractor,
		"requiresEquipment": target.RequiresEquipment, "requiresLabour": target.RequiresLabour,
		"version": target.Version,
	}, http.StatusOK)

	var log struct {
		Items []struct {
			Entity string `json:"entity"`
			Action string `json:"action"`
			Actor  string `json:"actor"`
		} `json:"items"`
	}
	decode(t, h.do(t, "GET", "/api/audit?entity=Activity&pageSize=5", admin, nil, http.StatusOK), &log)
	if len(log.Items) == 0 || log.Items[0].Action != "Update" || log.Items[0].Actor != "admin" {
		t.Fatalf("the change was not recorded: %+v", log.Items)
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	digits := ""
	negative := v < 0
	if negative {
		v = -v
	}
	for v > 0 {
		digits = string(rune('0'+v%10)) + digits
		v /= 10
	}
	if negative {
		return "-" + digits
	}
	return digits
}
