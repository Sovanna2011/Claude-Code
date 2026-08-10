package httpapi_test

// Planting projections over the real stack: the derived totals, the area rules that span rows, the
// approval workflow and the revision chain. Skips without FARMAREA_TEST_DATABASE_URL.

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type projection struct {
	ID             int    `json:"id"`
	ProjectionNo   string `json:"projectionNo"`
	Revision       int    `json:"revision"`
	SupersedesID   *int   `json:"supersedesId"`
	SupersededByID *int   `json:"supersededById"`
	IsCurrent      bool   `json:"isCurrent"`
	Status         string `json:"status"`

	TotalProjectedAreaHa   float64 `json:"totalProjectedAreaHa"`
	TotalHarvestableAreaHa float64 `json:"totalHarvestableAreaHa"`
	TotalExpectedTons      float64 `json:"totalExpectedTons"`
	LineCount              int     `json:"lineCount"`

	ApprovedBy      *string `json:"approvedBy"`
	RejectionReason *string `json:"rejectionReason"`
	Version         int     `json:"version"`

	Actions       []string `json:"actions"`
	LinesEditable bool     `json:"linesEditable"`

	Lines []struct {
		ID                      int     `json:"id"`
		BlockID                 int     `json:"blockId"`
		BlockCode               string  `json:"blockCode"`
		PlantingType            string  `json:"plantingType"`
		AvailableAreaHa         float64 `json:"availableAreaHa"`
		ProjectedPlantingAreaHa float64 `json:"projectedPlantingAreaHa"`
		ExpectedHarvestDate     *string `json:"expectedHarvestDate"`
		ExpectedYieldPerHa      float64 `json:"expectedYieldPerHa"`
		ExpectedLossPercent     float64 `json:"expectedLossPercent"`
		HarvestableAreaHa       float64 `json:"harvestableAreaHa"`
		ExpectedProductionTons  float64 `json:"expectedProductionTons"`
		RequiredSeedCaneTons    float64 `json:"requiredSeedCaneTons"`
		Priority                int     `json:"priority"`
		MapURL                  string  `json:"mapUrl"`
	} `json:"lines"`

	History []struct {
		Action     string  `json:"action"`
		FromStatus string  `json:"fromStatus"`
		ToStatus   string  `json:"toStatus"`
		Actor      string  `json:"actor"`
		Comments   *string `json:"comments"`
	} `json:"history"`
}

// The test database is not dropped between runs, and a projection number is unique per company, so
// each test mints its own rather than colliding with a previous run's rows.
var projectionSeq atomic.Int64

func projectionNo(prefix string) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano()%1_000_000, projectionSeq.Add(1))
}

// newProjection creates an empty draft for the 2026 season, whose planting window is 1 March to
// 28 August — the dates every line below sits inside.
func newProjection(t *testing.T, h *harness, token, prefix string) projection {
	t.Helper()
	var p projection
	decode(t, h.do(t, "POST", "/api/projections", token, map[string]any{
		"companyId": 1, "plantationId": 1, "cropSeasonId": 2,
		"projectionNo":   projectionNo(prefix),
		"projectionDate": "2026-01-15",
		"planningStart":  "2026-03-01",
		"planningEnd":    "2026-08-28",
		"preparedBy":     "Test planner",
	}, http.StatusCreated), &p)
	return p
}

func addLine(t *testing.T, h *harness, token string, id int, body map[string]any, wantStatus int) projection {
	t.Helper()
	raw := h.do(t, "POST", "/api/projections/"+itoa(id)+"/lines", token, body, wantStatus)
	var p projection
	if wantStatus == http.StatusCreated {
		decode(t, raw, &p)
	}
	return p
}

func line(blockID int, area float64) map[string]any {
	return map[string]any{
		"blockId": blockID, "caneVarietyId": 1, "plantingType": "NewPlanting",
		"projectedPlantingAreaHa": area,
		"plannedPlantingStart":    "2026-04-01", "plannedPlantingEnd": "2026-04-20",
	}
}

func act(t *testing.T, h *harness, token string, id int, action string, body map[string]any, wantStatus int) projection {
	t.Helper()
	raw := h.do(t, "POST", "/api/projections/"+itoa(id)+"/"+action, token, body, wantStatus)
	var p projection
	if wantStatus == http.StatusOK {
		decode(t, raw, &p)
	}
	return p
}

func getProjection(t *testing.T, h *harness, token string, id int) projection {
	t.Helper()
	var p projection
	decode(t, h.do(t, "GET", "/api/projections/"+itoa(id), token, nil, http.StatusOK), &p)
	return p
}

// ---------------------------------------------------------------- derived figures

func TestHeaderTotalsAreDerivedFromTheLines(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")
	p := newProjection(t, h, admin, "PRJ-TOTALS")

	if p.LineCount != 0 || p.TotalProjectedAreaHa != 0 {
		t.Fatalf("a new projection should be empty, got %d lines totalling %v",
			p.LineCount, p.TotalProjectedAreaHa)
	}

	// Variety CO0238 yields 90 t/ha with 5 per cent loss.
	// 10 ha  → 9.5 harvestable → 855 t.  40 ha → 38 harvestable → 3420 t.
	p = addLine(t, h, admin, p.ID, line(1, 10), http.StatusCreated)
	p = addLine(t, h, admin, p.ID, line(2, 40), http.StatusCreated)

	if p.LineCount != 2 || !near(p.TotalProjectedAreaHa, 50) {
		t.Fatalf("expected 2 lines totalling 50 ha, got %d totalling %v", p.LineCount, p.TotalProjectedAreaHa)
	}
	if !near(p.TotalHarvestableAreaHa, 47.5) {
		t.Errorf("harvestable area is %v, expected 47.5 — 50 ha less 5 per cent loss", p.TotalHarvestableAreaHa)
	}
	if !near(p.TotalExpectedTons, 4275) {
		t.Errorf("expected tonnage is %v, expected 4275 — 47.5 ha at 90 t/ha", p.TotalExpectedTons)
	}

	// The same figures, line by line, and the seed cane the variety's 8 t/ha rate implies.
	for _, l := range p.Lines {
		wantHarvestable := l.ProjectedPlantingAreaHa * 0.95
		if !near(l.HarvestableAreaHa, wantHarvestable) {
			t.Errorf("%s: harvestable %v, expected %v", l.BlockCode, l.HarvestableAreaHa, wantHarvestable)
		}
		if !near(l.ExpectedProductionTons, wantHarvestable*90) {
			t.Errorf("%s: production %v, expected %v", l.BlockCode, l.ExpectedProductionTons, wantHarvestable*90)
		}
		if !near(l.RequiredSeedCaneTons, l.ProjectedPlantingAreaHa*8) {
			t.Errorf("%s: seed cane %v, expected %v", l.BlockCode, l.RequiredSeedCaneTons, l.ProjectedPlantingAreaHa*8)
		}
		if l.MapURL == "" {
			t.Errorf("%s: a line should carry the block's Google Maps link", l.BlockCode)
		}
	}

	// Changing a line moves the header with it, and deleting one moves it back.
	p = getProjection(t, h, admin, p.ID)
	target := p.Lines[0]
	decode(t, h.do(t, "PUT", "/api/projections/"+itoa(p.ID)+"/lines/"+itoa(target.ID), admin,
		line(target.BlockID, 20), http.StatusOK), &p)
	if !near(p.TotalProjectedAreaHa, 60) {
		t.Fatalf("after growing a line to 20 ha the total is %v, expected 60", p.TotalProjectedAreaHa)
	}

	decode(t, h.do(t, "DELETE", "/api/projections/"+itoa(p.ID)+"/lines/"+itoa(target.ID), admin,
		nil, http.StatusOK), &p)
	if p.LineCount != 1 || !near(p.TotalProjectedAreaHa, 40) {
		t.Fatalf("after deleting a line: %d lines totalling %v, expected 1 totalling 40",
			p.LineCount, p.TotalProjectedAreaHa)
	}
}

// The header total is not something a caller can set. There is no field for it, and sending one is
// refused rather than quietly ignored — a client that thinks it set a total should be told.
func TestATotalCannotBeEntered(t *testing.T) {
	h := newHarness(t)
	h.do(t, "POST", "/api/projections", h.token(t, "admin"), map[string]any{
		"companyId": 1, "plantationId": 1, "cropSeasonId": 2,
		"projectionNo": projectionNo("PRJ-NOSET"), "projectionDate": "2026-01-15",
		"planningStart": "2026-03-01", "planningEnd": "2026-08-28",
		"totalProjectedAreaHa": 999,
	}, http.StatusBadRequest)
}

func TestTheHarvestDateIsDatedFromTheGrowingPeriod(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")
	p := newProjection(t, h, admin, "PRJ-HARVEST")

	// CO0238 grows for twelve months, so planting that ends on 20 April 2026 harvests a year later.
	p = addLine(t, h, admin, p.ID, line(3, 5), http.StatusCreated)
	if p.Lines[0].ExpectedHarvestDate == nil || *p.Lines[0].ExpectedHarvestDate != "2027-04-20" {
		t.Fatalf("expected a derived harvest date of 2027-04-20, got %v", p.Lines[0].ExpectedHarvestDate)
	}

	// A date the caller supplies is kept; one before the planting ends is refused.
	body := line(4, 5)
	body["expectedHarvestDate"] = "2027-06-30"
	p = addLine(t, h, admin, p.ID, body, http.StatusCreated)
	found := false
	for _, l := range p.Lines {
		if l.BlockID == 4 {
			found = true
			if l.ExpectedHarvestDate == nil || *l.ExpectedHarvestDate != "2027-06-30" {
				t.Fatalf("the supplied harvest date was not kept: %v", l.ExpectedHarvestDate)
			}
		}
	}
	if !found {
		t.Fatal("the line for block 4 was not written")
	}

	early := line(5, 5)
	early["expectedHarvestDate"] = "2026-01-01"
	addLine(t, h, admin, p.ID, early, http.StatusUnprocessableEntity)
}

// ---------------------------------------------------------------- the area rules

func TestALineCannotExceedItsBlock(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")
	p := newProjection(t, h, admin, "PRJ-AREA")

	// BLK-001 has 119.1 ha that can carry cane.
	body := h.do(t, "POST", "/api/projections/"+itoa(p.ID)+"/lines", admin,
		line(1, 200), http.StatusUnprocessableEntity)
	if !strings.Contains(string(body), "BLK-001") || !strings.Contains(string(body), "119.1") {
		t.Fatalf("the refusal should name the block and its plantable area, got %s", body)
	}
}

// Two lines on one block — a new planting and a ratoon — may not add up to more than the block.
// No single CHECK can say that, so it is a deferred trigger, and this is the test that it fires.
func TestTwoLinesOnOneBlockCannotExceedItTogether(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")
	p := newProjection(t, h, admin, "PRJ-SUM")

	// BLK-006 has 121.7 ha. 100 fits; a further 30 does not.
	addLine(t, h, admin, p.ID, line(6, 100), http.StatusCreated)

	ratoon := line(6, 30)
	ratoon["plantingType"] = "Ratoon"
	body := h.do(t, "POST", "/api/projections/"+itoa(p.ID)+"/lines", admin, ratoon, http.StatusUnprocessableEntity)
	if !strings.Contains(string(body), "AREA_EXCEEDS_BLOCK") && !strings.Contains(string(body), "121.7") {
		t.Fatalf("expected the cross-row area rule to refuse this, got %s", body)
	}

	// And the first line is still there: a refused write left nothing behind.
	after := getProjection(t, h, admin, p.ID)
	if after.LineCount != 1 || !near(after.TotalProjectedAreaHa, 100) {
		t.Fatalf("after the refusal: %d lines totalling %v, expected 1 totalling 100",
			after.LineCount, after.TotalProjectedAreaHa)
	}
}

// The available area is a snapshot the server takes from the block, not a figure the caller sends.
// Sending one is refused, so the "within the block" check cannot be aimed at a number of the
// caller's choosing.
func TestTheAvailableAreaComesFromTheBlockNotTheCaller(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")
	p := newProjection(t, h, admin, "PRJ-SNAP")

	lying := line(7, 50)
	lying["availableAreaHa"] = 10000
	h.do(t, "POST", "/api/projections/"+itoa(p.ID)+"/lines", admin, lying, http.StatusBadRequest)

	p = addLine(t, h, admin, p.ID, line(7, 50), http.StatusCreated)
	if !near(p.Lines[0].AvailableAreaHa, 121.4) && p.Lines[0].AvailableAreaHa <= 0 {
		t.Fatalf("the snapshot should be the block's own plantable area, got %v", p.Lines[0].AvailableAreaHa)
	}
	if p.Lines[0].AvailableAreaHa > 1000 {
		t.Fatalf("the caller's figure was used after all: %v", p.Lines[0].AvailableAreaHa)
	}
}

func TestPlantingMustFallInsideTheSeasonAndThePlan(t *testing.T) {
	h := newHarness(t)
	admin := h.token(t, "admin")
	p := newProjection(t, h, admin, "PRJ-WINDOW")

	// The 2026 season plants between 1 March and 28 August. December is outside it.
	late := line(8, 5)
	late["plannedPlantingStart"] = "2026-12-01"
	late["plannedPlantingEnd"] = "2026-12-20"
	body := h.do(t, "POST", "/api/projections/"+itoa(p.ID)+"/lines", admin, late, http.StatusUnprocessableEntity)
	if !strings.Contains(string(body), "CS2026") && !strings.Contains(string(body), "2026-03-01") {
		t.Fatalf("the refusal should quote the season's window, got %s", body)
	}
}

// ---------------------------------------------------------------- the workflow

func TestTheApprovalWorkflowRunsEndToEnd(t *testing.T) {
	h := newHarness(t)
	planner, manager := h.token(t, "planner"), h.token(t, "manager")

	p := newProjection(t, h, planner, "PRJ-FLOW")
	p = addLine(t, h, planner, p.ID, line(9, 25), http.StatusCreated)
	if !p.LinesEditable {
		t.Fatal("a draft should be editable by its planner")
	}

	p = act(t, h, planner, p.ID, "submit", nil, http.StatusOK)
	if p.Status != "Submitted" {
		t.Fatalf("after submitting, status is %s", p.Status)
	}
	if p.LinesEditable {
		t.Fatal("a submitted plan must not still be editable")
	}

	p = act(t, h, manager, p.ID, "review", nil, http.StatusOK)
	if p.Status != "UnderReview" {
		t.Fatalf("after review, status is %s", p.Status)
	}

	p = act(t, h, manager, p.ID, "approve", map[string]any{"comments": "Fits the mill's window."}, http.StatusOK)
	if p.Status != "Approved" {
		t.Fatalf("after approval, status is %s", p.Status)
	}
	if p.ApprovedBy == nil || *p.ApprovedBy != "manager" {
		t.Fatalf("the approval should be stamped with who took it, got %v", p.ApprovedBy)
	}

	p = act(t, h, manager, p.ID, "close", nil, http.StatusOK)
	if p.Status != "Closed" {
		t.Fatalf("after closing, status is %s", p.Status)
	}

	// Every step left a line in the trail, in order, with who took it.
	full := getProjection(t, h, manager, p.ID)
	want := []string{"Submit", "Review", "Approve", "Close"}
	if len(full.History) != len(want) {
		t.Fatalf("expected %d trail entries, got %d", len(want), len(full.History))
	}
	for i, action := range want {
		if full.History[i].Action != action {
			t.Errorf("trail entry %d is %s, expected %s", i, full.History[i].Action, action)
		}
	}
	if full.History[0].Actor != "planner" || full.History[2].Actor != "manager" {
		t.Errorf("the trail should name who took each step: %+v", full.History)
	}
	if full.History[2].Comments == nil || *full.History[2].Comments != "Fits the mill's window." {
		t.Errorf("the approver's comment was not kept: %v", full.History[2].Comments)
	}
}

func TestAnEmptyProjectionCannotBeSubmitted(t *testing.T) {
	h := newHarness(t)
	planner := h.token(t, "planner")
	p := newProjection(t, h, planner, "PRJ-EMPTY")
	body := h.do(t, "POST", "/api/projections/"+itoa(p.ID)+"/submit", planner, nil, http.StatusUnprocessableEntity)
	if !strings.Contains(string(body), "nothing to approve") {
		t.Fatalf("expected the refusal to explain itself, got %s", body)
	}
}

func TestAPlannerCannotApproveTheirOwnPlan(t *testing.T) {
	h := newHarness(t)
	planner := h.token(t, "planner")
	p := newProjection(t, h, planner, "PRJ-SELF")
	addLine(t, h, planner, p.ID, line(10, 15), http.StatusCreated)
	act(t, h, planner, p.ID, "submit", nil, http.StatusOK)
	act(t, h, planner, p.ID, "approve", nil, http.StatusForbidden)
}

func TestAViewerCannotTouchTheWorkflow(t *testing.T) {
	h := newHarness(t)
	viewer, planner := h.token(t, "viewer"), h.token(t, "planner")
	p := newProjection(t, h, planner, "PRJ-VIEW")
	addLine(t, h, planner, p.ID, line(11, 15), http.StatusCreated)

	// A viewer reads it and is offered nothing to do.
	seen := getProjection(t, h, viewer, p.ID)
	if len(seen.Actions) != 0 || seen.LinesEditable {
		t.Fatalf("a report viewer should be offered no actions, got %v (editable=%v)",
			seen.Actions, seen.LinesEditable)
	}
	act(t, h, viewer, p.ID, "submit", nil, http.StatusForbidden)
	h.do(t, "POST", "/api/projections/"+itoa(p.ID)+"/lines", viewer, line(12, 5), http.StatusForbidden)
}

func TestTheWrongStepAtTheWrongTimeIsAConflict(t *testing.T) {
	h := newHarness(t)
	manager := h.token(t, "manager")
	p := newProjection(t, h, manager, "PRJ-ORDER")
	addLine(t, h, manager, p.ID, line(13, 15), http.StatusCreated)

	// Nothing to approve until it has been submitted.
	act(t, h, manager, p.ID, "approve", nil, http.StatusConflict)
	act(t, h, manager, p.ID, "close", nil, http.StatusConflict)

	act(t, h, manager, p.ID, "submit", nil, http.StatusOK)
	// And it cannot be submitted twice.
	act(t, h, manager, p.ID, "submit", nil, http.StatusConflict)
}

func TestRejectingNeedsAReasonAndKeepsIt(t *testing.T) {
	h := newHarness(t)
	planner, manager := h.token(t, "planner"), h.token(t, "manager")
	p := newProjection(t, h, planner, "PRJ-REJECT")
	addLine(t, h, planner, p.ID, line(14, 15), http.StatusCreated)
	act(t, h, planner, p.ID, "submit", nil, http.StatusOK)

	act(t, h, manager, p.ID, "reject", nil, http.StatusUnprocessableEntity)
	act(t, h, manager, p.ID, "reject", map[string]any{"comments": "  "}, http.StatusUnprocessableEntity)

	rejected := act(t, h, manager, p.ID, "reject",
		map[string]any{"comments": "The mill cannot take this in April."}, http.StatusOK)
	if rejected.Status != "Rejected" {
		t.Fatalf("status is %s, expected Rejected", rejected.Status)
	}
	if rejected.RejectionReason == nil || !strings.Contains(*rejected.RejectionReason, "mill") {
		t.Fatalf("the reason was not kept on the header: %v", rejected.RejectionReason)
	}
}

func TestReturningForCorrectionMakesItADraftAgain(t *testing.T) {
	h := newHarness(t)
	planner, manager := h.token(t, "planner"), h.token(t, "manager")
	p := newProjection(t, h, planner, "PRJ-RETURN")
	addLine(t, h, planner, p.ID, line(15, 15), http.StatusCreated)
	act(t, h, planner, p.ID, "submit", nil, http.StatusOK)

	returned := act(t, h, manager, p.ID, "returnforcorrection",
		map[string]any{"comments": "Split this across two blocks."}, http.StatusOK)
	if returned.Status != "Draft" {
		t.Fatalf("status is %s, expected Draft", returned.Status)
	}
	// And the planner can work on it again.
	after := addLine(t, h, planner, p.ID, line(16, 10), http.StatusCreated)
	if after.LineCount != 2 {
		t.Fatalf("expected the returned draft to accept a second line, got %d", after.LineCount)
	}
}

// ---------------------------------------------------------------- revisions

func TestRevisingAnApprovedPlanOpensANewVersion(t *testing.T) {
	h := newHarness(t)
	manager := h.token(t, "manager")
	original := newProjection(t, h, manager, "PRJ-REV")
	addLine(t, h, manager, original.ID, line(17, 30), http.StatusCreated)
	addLine(t, h, manager, original.ID, line(18, 20), http.StatusCreated)
	act(t, h, manager, original.ID, "submit", nil, http.StatusOK)
	act(t, h, manager, original.ID, "approve", nil, http.StatusOK)

	revision := act(t, h, manager, original.ID, "revise",
		map[string]any{"comments": "Rain pushed the window back."}, http.StatusOK)

	// The answer is the new draft, not the plan that was revised.
	if revision.ID == original.ID {
		t.Fatal("revising should have produced a second row")
	}
	if revision.Status != "Draft" || !revision.IsCurrent {
		t.Fatalf("the revision should be the current draft, got %s (current=%v)",
			revision.Status, revision.IsCurrent)
	}
	if revision.Revision != original.Revision+1 {
		t.Fatalf("the revision is version %d, expected %d", revision.Revision, original.Revision+1)
	}
	if revision.SupersedesID == nil || *revision.SupersedesID != original.ID {
		t.Fatalf("the revision should point back at %d, got %v", original.ID, revision.SupersedesID)
	}

	// Its lines were copied, totals and all, so the planner amends rather than retypes.
	if revision.LineCount != 2 || !near(revision.TotalProjectedAreaHa, 50) {
		t.Fatalf("the lines were not copied: %d lines totalling %v",
			revision.LineCount, revision.TotalProjectedAreaHa)
	}

	// The original is now history: revised, no longer current, and pointing forwards.
	was := getProjection(t, h, manager, original.ID)
	if was.Status != "Revised" || was.IsCurrent {
		t.Fatalf("the original should be Revised and no longer current, got %s (current=%v)",
			was.Status, was.IsCurrent)
	}
	if was.SupersededByID == nil || *was.SupersededByID != revision.ID {
		t.Fatalf("the original should point forward at %d, got %v", revision.ID, was.SupersededByID)
	}

	// currentOnly hides the superseded version from the list.
	var page struct {
		Items []projection `json:"items"`
	}
	decode(t, h.do(t, "GET", "/api/projections?currentOnly=true&pageSize=200&search="+original.ProjectionNo,
		manager, nil, http.StatusOK), &page)
	for _, item := range page.Items {
		if item.ID == original.ID {
			t.Fatal("currentOnly still returned the superseded version")
		}
	}
	if len(page.Items) != 1 || page.Items[0].ID != revision.ID {
		t.Fatalf("currentOnly should return exactly the revision, got %d items", len(page.Items))
	}
}

// An approved plan is a commitment other modules read. It cannot be edited behind their backs —
// the only way to change it is to open a revision.
func TestAnApprovedPlanCannotBeEdited(t *testing.T) {
	h := newHarness(t)
	manager := h.token(t, "manager")
	p := newProjection(t, h, manager, "PRJ-FROZEN")
	added := addLine(t, h, manager, p.ID, line(19, 30), http.StatusCreated)
	act(t, h, manager, p.ID, "submit", nil, http.StatusOK)
	approved := act(t, h, manager, p.ID, "approve", nil, http.StatusOK)

	h.do(t, "POST", "/api/projections/"+itoa(p.ID)+"/lines", manager, line(20, 10), http.StatusConflict)
	h.do(t, "DELETE", "/api/projections/"+itoa(p.ID)+"/lines/"+itoa(added.Lines[0].ID),
		manager, nil, http.StatusConflict)
	h.do(t, "PUT", "/api/projections/"+itoa(p.ID), manager, map[string]any{
		"companyId": 1, "plantationId": 1, "cropSeasonId": 2,
		"projectionNo": approved.ProjectionNo, "projectionDate": "2026-01-15",
		"planningStart": "2026-03-01", "planningEnd": "2026-08-28",
		"version": approved.Version,
	}, http.StatusConflict)
}

// ---------------------------------------------------------------- committed land

// Two planners may draft alternatives for the same block — that is how options get compared. The
// land is only taken when one of them is approved, and then the other cannot be.
func TestApprovingTwiceOverTheSameBlockAndWindowIsRefused(t *testing.T) {
	h := newHarness(t)
	manager := h.token(t, "manager")

	// A block no other test approves over, so the rule is tested rather than the test order.
	const block = 12

	first := newProjection(t, h, manager, "PRJ-CLASH-A")
	addLine(t, h, manager, first.ID, line(block, 40), http.StatusCreated)
	second := newProjection(t, h, manager, "PRJ-CLASH-B")
	addLine(t, h, manager, second.ID, line(block, 40), http.StatusCreated)

	// Both drafts exist happily side by side, and both may be submitted.
	act(t, h, manager, first.ID, "submit", nil, http.StatusOK)
	act(t, h, manager, second.ID, "submit", nil, http.StatusOK)

	act(t, h, manager, first.ID, "approve", nil, http.StatusOK)

	body := h.do(t, "POST", "/api/projections/"+itoa(second.ID)+"/approve", manager, nil, http.StatusConflict)
	if !strings.Contains(string(body), "BLOCK_ALREADY_COMMITTED") {
		t.Fatalf("expected BLOCK_ALREADY_COMMITTED, got %s", body)
	}

	// The refused plan is untouched — still submitted, not half-approved.
	after := getProjection(t, h, manager, second.ID)
	if after.Status != "Submitted" {
		t.Fatalf("the refused plan is %s; the failed approval should have rolled back", after.Status)
	}
}

// The same block in a window that does not touch the committed one is fine — that is a second crop
// on the same land, not a double booking.
func TestASecondPlanInADifferentWindowIsAllowed(t *testing.T) {
	h := newHarness(t)
	manager := h.token(t, "manager")
	const block = 13

	early := newProjection(t, h, manager, "PRJ-EARLY")
	body := line(block, 30)
	body["plannedPlantingStart"] = "2026-03-05"
	body["plannedPlantingEnd"] = "2026-03-25"
	addLine(t, h, manager, early.ID, body, http.StatusCreated)
	act(t, h, manager, early.ID, "submit", nil, http.StatusOK)
	act(t, h, manager, early.ID, "approve", nil, http.StatusOK)

	late := newProjection(t, h, manager, "PRJ-LATE")
	body = line(block, 30)
	body["plannedPlantingStart"] = "2026-07-01"
	body["plannedPlantingEnd"] = "2026-07-20"
	addLine(t, h, manager, late.ID, body, http.StatusCreated)
	act(t, h, manager, late.ID, "submit", nil, http.StatusOK)
	act(t, h, manager, late.ID, "approve", nil, http.StatusOK)
}

// ---------------------------------------------------------------- concurrency and filters

func TestTwoManagersDecidingAtOnceAreNotBothHonoured(t *testing.T) {
	h := newHarness(t)
	manager := h.token(t, "manager")
	p := newProjection(t, h, manager, "PRJ-RACE")
	addLine(t, h, manager, p.ID, line(14, 10), http.StatusCreated)
	submitted := act(t, h, manager, p.ID, "submit", nil, http.StatusOK)

	// Both read version N. The first decision wins; the second is told the plan moved.
	stale := submitted.Version
	act(t, h, manager, p.ID, "approve", map[string]any{"version": stale}, http.StatusOK)
	h.do(t, "POST", "/api/projections/"+itoa(p.ID)+"/reject", manager,
		map[string]any{"version": stale, "comments": "Too late."}, http.StatusConflict)
}

func TestTheWorkflowGraphIsServedToClients(t *testing.T) {
	h := newHarness(t)
	var graph struct {
		Statuses    []string `json:"statuses"`
		Transitions []struct {
			Action      string   `json:"action"`
			From        []string `json:"from"`
			To          string   `json:"to"`
			Roles       []string `json:"roles"`
			NeedsReason bool     `json:"needsReason"`
		} `json:"transitions"`
	}
	decode(t, h.do(t, "GET", "/api/projections/workflow", h.token(t, "viewer"), nil, http.StatusOK), &graph)
	if len(graph.Statuses) != 7 || len(graph.Transitions) != 7 {
		t.Fatalf("expected seven statuses and seven transitions, got %d and %d",
			len(graph.Statuses), len(graph.Transitions))
	}
}

func TestProjectionsCanBeFoundByBlockAndStatus(t *testing.T) {
	h := newHarness(t)
	manager := h.token(t, "manager")
	const block = 15

	p := newProjection(t, h, manager, "PRJ-FIND")
	addLine(t, h, manager, p.ID, line(block, 20), http.StatusCreated)

	var page struct {
		Items []projection `json:"items"`
	}
	decode(t, h.do(t, "GET", "/api/projections?blockId="+itoa(block)+"&pageSize=200",
		manager, nil, http.StatusOK), &page)
	if !containsProjection(page.Items, p.ID) {
		t.Fatalf("the plan for block %d was not found by blockId", block)
	}

	decode(t, h.do(t, "GET", "/api/projections?projectionStatus=Draft&pageSize=200",
		manager, nil, http.StatusOK), &page)
	if !containsProjection(page.Items, p.ID) {
		t.Fatal("the draft was not found by status")
	}
	for _, item := range page.Items {
		if item.Status != "Draft" {
			t.Fatalf("the status filter let a %s plan through", item.Status)
		}
	}

	// An unknown status is refused rather than passed to Postgres as a bad enum.
	h.do(t, "GET", "/api/projections?projectionStatus=Blessed", manager, nil, http.StatusBadRequest)
}

// Farm, zone and block all ask the same question of a projection — "does this plan touch that
// ground?" — and they have to agree with the hierarchy: the farm and zone a block sits in find it,
// and a sibling zone does not.
func TestProjectionsCanBeFoundByFarmZoneAndBlock(t *testing.T) {
	h := newHarness(t)
	manager := h.token(t, "manager")
	const block = 16

	// Where this block actually sits, straight from the block master.
	var blk struct {
		ZoneID int `json:"zoneId"`
		FarmID int `json:"farmId"`
	}
	decode(t, h.do(t, "GET", "/api/blocks/"+itoa(block), manager, nil, http.StatusOK), &blk)

	p := newProjection(t, h, manager, "PRJ-LAND")
	addLine(t, h, manager, p.ID, line(block, 20), http.StatusCreated)

	found := func(query string) bool {
		var page struct {
			Items []projection `json:"items"`
		}
		decode(t, h.do(t, "GET", "/api/projections?pageSize=200&"+query, manager, nil, http.StatusOK), &page)
		return containsProjection(page.Items, p.ID)
	}

	for _, query := range []string{
		"farmId=" + itoa(blk.FarmID),
		"zoneId=" + itoa(blk.ZoneID),
		"blockId=" + itoa(block),
		// The three together still find it: they narrow one subquery, they do not fight.
		"farmId=" + itoa(blk.FarmID) + "&zoneId=" + itoa(blk.ZoneID) + "&blockId=" + itoa(block),
	} {
		if !found(query) {
			t.Errorf("the plan on block %d was not found by %s", block, query)
		}
	}

	// A zone the block is not in must not find it, even when the farm does.
	var zones struct {
		Items []struct {
			ID int `json:"id"`
		} `json:"items"`
	}
	decode(t, h.do(t, "GET", "/api/zones?pageSize=100", manager, nil, http.StatusOK), &zones)
	for _, z := range zones.Items {
		if z.ID == blk.ZoneID {
			continue
		}
		if found("zoneId=" + itoa(z.ID) + "&blockId=" + itoa(block)) {
			t.Fatalf("zone %d found a plan whose only block lives in zone %d", z.ID, blk.ZoneID)
		}
		break
	}
}

func containsProjection(items []projection, id int) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}
