package httpapi_test

// The activity plan over the real stack: generating a programme from an approved projection,
// regenerating it, and the Draft → Released → Closed lifecycle.

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

type activityPlan struct {
	ID                int      `json:"id"`
	ProjectionID      int      `json:"projectionId"`
	ProjectionNo      string   `json:"projectionNo"`
	PlanNo            string   `json:"planNo"`
	Status            string   `json:"status"`
	WorkOnSaturday    bool     `json:"workOnSaturday"`
	WorkOnSunday      bool     `json:"workOnSunday"`
	Holidays          []string `json:"holidays"`
	StartsOn          *string  `json:"startsOn"`
	EndsOn            *string  `json:"endsOn"`
	TaskCount         int      `json:"taskCount"`
	TotalAreaHa       float64  `json:"totalAreaHa"`
	TotalWorkingHours float64  `json:"totalWorkingHours"`
	TotalLabourDays   float64  `json:"totalLabourDays"`
	GeneratedBy       *string  `json:"generatedBy"`
	ReleasedBy        *string  `json:"releasedBy"`
	Version           int      `json:"version"`
	Editable          bool     `json:"editable"`
	Actions           []string `json:"actions"`
	Tasks             []struct {
		ID               int64   `json:"id"`
		BlockID          int     `json:"blockId"`
		BlockCode        string  `json:"blockCode"`
		ActivityID       int     `json:"activityId"`
		ActivityCode     string  `json:"activityCode"`
		Category         string  `json:"category"`
		SequenceNo       int     `json:"sequenceNo"`
		PlannedAreaHa    float64 `json:"plannedAreaHa"`
		PlannedStartDate string  `json:"plannedStartDate"`
		PlannedEndDate   string  `json:"plannedEndDate"`
		DurationDays     int     `json:"durationDays"`
		DailyTargetHa    float64 `json:"dailyTargetHa"`
		RequiredWorkers  int     `json:"requiredWorkers"`
		Status           string  `json:"status"`
	} `json:"tasks"`
}

// planLine plants in August. Approving a projection commits its blocks for their planting window,
// and the projection tests approve the same blocks in March, April and July — so these tests take a
// window of their own rather than racing them for the land.
func planLine(blockID int, area float64) map[string]any {
	body := line(blockID, area)
	body["plannedPlantingStart"] = "2026-08-03"
	body["plannedPlantingEnd"] = "2026-08-20"
	return body
}

// approvedProjection makes a projection over the given blocks and takes it all the way to Approved,
// which is the only state a plan may be generated from.
func approvedProjection(t *testing.T, h *harness, prefix string, blocks ...int) projection {
	t.Helper()
	manager := h.token(t, "manager")
	p := newProjection(t, h, manager, prefix)
	for _, b := range blocks {
		addLine(t, h, manager, p.ID, planLine(b, 20), http.StatusCreated)
	}
	act(t, h, manager, p.ID, "submit", nil, http.StatusOK)
	return act(t, h, manager, p.ID, "approve", nil, http.StatusOK)
}

func generatePlan(t *testing.T, h *harness, token string, body map[string]any, wantStatus int) activityPlan {
	t.Helper()
	raw := h.do(t, "POST", "/api/plans", token, body, wantStatus)
	var plan activityPlan
	if wantStatus == http.StatusOK {
		decode(t, raw, &plan)
	}
	return plan
}

func TestAPlanIsGeneratedFromAnApprovedProjection(t *testing.T) {
	h := newHarness(t)
	manager := h.token(t, "manager")
	p := approvedProjection(t, h, "PRJ-PLAN", 1, 2)

	plan := generatePlan(t, h, manager, map[string]any{"projectionId": p.ID}, http.StatusOK)

	if plan.Status != "Draft" {
		t.Fatalf("a fresh plan should be a draft, got %s", plan.Status)
	}
	if !strings.Contains(plan.PlanNo, p.ProjectionNo) {
		t.Errorf("the plan should be numbered after its projection, got %s", plan.PlanNo)
	}
	if plan.TaskCount == 0 || len(plan.Tasks) != plan.TaskCount {
		t.Fatalf("header says %d tasks, body carries %d", plan.TaskCount, len(plan.Tasks))
	}
	if plan.GeneratedBy == nil || *plan.GeneratedBy != "manager" {
		t.Errorf("the plan should record who generated it, got %v", plan.GeneratedBy)
	}

	// Two blocks, and the nineteen-activity master: every block gets its own programme.
	blocks := map[int]int{}
	for _, task := range plan.Tasks {
		blocks[task.BlockID]++
		if task.Status != "Planned" {
			t.Errorf("a freshly generated task should be Planned, got %s", task.Status)
		}
		if task.PlannedEndDate < task.PlannedStartDate {
			t.Errorf("%s on %s ends before it starts", task.ActivityCode, task.BlockCode)
		}
		if task.DurationDays < 1 {
			t.Errorf("%s on %s has no duration", task.ActivityCode, task.BlockCode)
		}
	}
	if len(blocks) != 2 {
		t.Fatalf("expected a programme for each of two blocks, got %d", len(blocks))
	}

	// The header window spans every task.
	if plan.StartsOn == nil || plan.EndsOn == nil {
		t.Fatal("the header should carry the window the tasks span")
	}
	for _, task := range plan.Tasks {
		if task.PlannedStartDate < *plan.StartsOn || task.PlannedEndDate > *plan.EndsOn {
			t.Errorf("%s on %s runs %s–%s, outside the header window %s–%s",
				task.ActivityCode, task.BlockCode, task.PlannedStartDate, task.PlannedEndDate,
				*plan.StartsOn, *plan.EndsOn)
		}
	}
}

// The whole point of a chain: within a block, an activity that waits for another starts after it.
func TestTasksRespectTheDependencyChain(t *testing.T) {
	h := newHarness(t)
	manager := h.token(t, "manager")
	p := approvedProjection(t, h, "PRJ-CHAIN", 3)
	plan := generatePlan(t, h, manager, map[string]any{"projectionId": p.ID}, http.StatusOK)

	// The seeded master is a straight chain in sequence order, so each task must start no earlier
	// than the previous one ended.
	var previousEnd, previousCode string
	for _, task := range plan.Tasks {
		if previousEnd != "" && task.PlannedStartDate <= previousEnd {
			t.Errorf("%s starts %s, but %s runs to %s — the chain was not honoured",
				task.ActivityCode, task.PlannedStartDate, previousCode, previousEnd)
		}
		previousEnd, previousCode = task.PlannedEndDate, task.ActivityCode
	}
	if previousEnd == "" {
		t.Fatal("no tasks were generated")
	}
}

// The calendar is an input, kept with the plan. Changing it and regenerating moves the dates.
func TestTheWorkingCalendarChangesTheDates(t *testing.T) {
	h := newHarness(t)
	manager := h.token(t, "manager")

	// A hundred hectares, not twenty: at the seeded capacities a small block finishes every
	// activity inside a day, so the programme is driven by the offsets and the working week never
	// shows. An area that takes weeks is what makes the calendar visible.
	p := newProjection(t, h, manager, "PRJ-CAL")
	addLine(t, h, manager, p.ID, planLine(4, 100), http.StatusCreated)
	act(t, h, manager, p.ID, "submit", nil, http.StatusOK)
	act(t, h, manager, p.ID, "approve", nil, http.StatusOK)

	sixDayWeek := generatePlan(t, h, manager, map[string]any{"projectionId": p.ID}, http.StatusOK)

	// A five-day week has fewer working days, so the same work runs later.
	fiveDayWeek := generatePlan(t, h, manager, map[string]any{
		"projectionId": p.ID, "workOnSaturday": false,
	}, http.StatusOK)

	if fiveDayWeek.WorkOnSaturday {
		t.Fatal("the calendar was not stored with the plan")
	}
	if *fiveDayWeek.EndsOn <= *sixDayWeek.EndsOn {
		t.Errorf("dropping Saturdays should push the programme later: %s then %s",
			*sixDayWeek.EndsOn, *fiveDayWeek.EndsOn)
	}
	// Regenerating replaced the programme rather than adding a second copy of it.
	if fiveDayWeek.ID != sixDayWeek.ID {
		t.Errorf("regenerating should reuse the plan, got %d then %d", sixDayWeek.ID, fiveDayWeek.ID)
	}
	if fiveDayWeek.TaskCount != sixDayWeek.TaskCount {
		t.Errorf("regenerating changed the task count from %d to %d",
			sixDayWeek.TaskCount, fiveDayWeek.TaskCount)
	}

	// A holiday is honoured too: no task may start on one.
	withHoliday := generatePlan(t, h, manager, map[string]any{
		"projectionId": p.ID, "holidays": []string{*sixDayWeek.StartsOn},
	}, http.StatusOK)
	for _, task := range withHoliday.Tasks {
		if task.PlannedStartDate == *sixDayWeek.StartsOn {
			t.Errorf("%s was scheduled to start on a holiday", task.ActivityCode)
		}
	}
	if len(withHoliday.Holidays) != 1 {
		t.Errorf("the holiday should be stored with the plan, got %v", withHoliday.Holidays)
	}
}

func TestOnlyAnApprovedProjectionCanBePlanned(t *testing.T) {
	h := newHarness(t)
	manager := h.token(t, "manager")

	draft := newProjection(t, h, manager, "PRJ-NOTAPPROVED")
	addLine(t, h, manager, draft.ID, planLine(5, 10), http.StatusCreated)

	body := h.do(t, "POST", "/api/plans", manager,
		map[string]any{"projectionId": draft.ID}, http.StatusConflict)
	if !strings.Contains(string(body), "PROJECTION_NOT_APPROVED") {
		t.Fatalf("expected PROJECTION_NOT_APPROVED, got %s", body)
	}

	// And a projection that does not exist at all is a 404, not a conflict.
	h.do(t, "POST", "/api/plans", manager, map[string]any{"projectionId": 999999}, http.StatusNotFound)
}

func TestThePlanLifecycle(t *testing.T) {
	h := newHarness(t)
	manager := h.token(t, "manager")
	p := approvedProjection(t, h, "PRJ-LIFECYCLE", 6)
	plan := generatePlan(t, h, manager, map[string]any{"projectionId": p.ID}, http.StatusOK)

	if !plan.Editable {
		t.Fatal("a draft should be editable")
	}

	var released activityPlan
	decode(t, h.do(t, "POST", "/api/plans/"+itoa(plan.ID)+"/release", manager, nil, http.StatusOK), &released)
	if released.Status != "Released" {
		t.Fatalf("status is %s, expected Released", released.Status)
	}
	if released.ReleasedBy == nil || *released.ReleasedBy != "manager" {
		t.Errorf("the release should be stamped, got %v", released.ReleasedBy)
	}
	if released.Editable {
		t.Error("a released plan must not be editable")
	}

	// A released plan's dates are a commitment: regenerating is refused until it is reopened.
	body := h.do(t, "POST", "/api/plans", manager, map[string]any{"projectionId": p.ID}, http.StatusConflict)
	if !strings.Contains(string(body), "PLAN_NOT_EDITABLE") {
		t.Fatalf("expected PLAN_NOT_EDITABLE, got %s", body)
	}
	// Nor may it be deleted.
	h.do(t, "DELETE", "/api/plans/"+itoa(plan.ID), manager, nil, http.StatusConflict)

	// Releasing twice is a conflict, not a second release.
	h.do(t, "POST", "/api/plans/"+itoa(plan.ID)+"/release", manager, nil, http.StatusConflict)

	var closed activityPlan
	decode(t, h.do(t, "POST", "/api/plans/"+itoa(plan.ID)+"/close", manager, nil, http.StatusOK), &closed)
	if closed.Status != "Closed" {
		t.Fatalf("status is %s, expected Closed", closed.Status)
	}

	// Reopening puts it back to a draft, and regenerating works again.
	var reopened activityPlan
	decode(t, h.do(t, "POST", "/api/plans/"+itoa(plan.ID)+"/reopen", manager, nil, http.StatusOK), &reopened)
	if reopened.Status != "Draft" {
		t.Fatalf("status is %s, expected Draft", reopened.Status)
	}
	generatePlan(t, h, manager, map[string]any{"projectionId": p.ID}, http.StatusOK)
}

func TestAViewerCanReadAPlanButNotMakeOne(t *testing.T) {
	h := newHarness(t)
	viewer := h.token(t, "viewer")
	p := approvedProjection(t, h, "PRJ-PLANVIEW", 7)
	plan := generatePlan(t, h, h.token(t, "manager"), map[string]any{"projectionId": p.ID}, http.StatusOK)

	seen := activityPlan{}
	decode(t, h.do(t, "GET", "/api/plans/"+itoa(plan.ID), viewer, nil, http.StatusOK), &seen)
	if len(seen.Actions) != 0 || seen.Editable {
		t.Fatalf("a report viewer should be offered nothing, got %v (editable=%v)", seen.Actions, seen.Editable)
	}
	h.do(t, "POST", "/api/plans", viewer, map[string]any{"projectionId": p.ID}, http.StatusForbidden)
	h.do(t, "POST", "/api/plans/"+itoa(plan.ID)+"/release", viewer, nil, http.StatusForbidden)
	h.do(t, "DELETE", "/api/plans/"+itoa(plan.ID), viewer, nil, http.StatusForbidden)
}

func TestPlanTasksCanBeNarrowedToOneBlock(t *testing.T) {
	h := newHarness(t)
	manager := h.token(t, "manager")
	p := approvedProjection(t, h, "PRJ-PLANFILTER", 8, 9)
	plan := generatePlan(t, h, manager, map[string]any{"projectionId": p.ID}, http.StatusOK)

	var oneBlock activityPlan
	decode(t, h.do(t, "GET", "/api/plans/"+itoa(plan.ID)+"?blockId=8", manager, nil, http.StatusOK), &oneBlock)
	if len(oneBlock.Tasks) == 0 || len(oneBlock.Tasks) >= len(plan.Tasks) {
		t.Fatalf("filtering to one block returned %d of %d tasks", len(oneBlock.Tasks), len(plan.Tasks))
	}
	for _, task := range oneBlock.Tasks {
		if task.BlockID != 8 {
			t.Errorf("block filter let through a task on block %d", task.BlockID)
		}
	}
	// The header keeps the whole plan's figures; only the task list is narrowed.
	if oneBlock.TaskCount != plan.TaskCount {
		t.Errorf("the header should still describe the whole plan, got %d of %d",
			oneBlock.TaskCount, plan.TaskCount)
	}

	// And by category.
	var planting activityPlan
	decode(t, h.do(t, "GET", "/api/plans/"+itoa(plan.ID)+"?activityCategory=Planting",
		manager, nil, http.StatusOK), &planting)
	for _, task := range planting.Tasks {
		if task.Category != "Planting" {
			t.Errorf("category filter let through a %s task", task.Category)
		}
	}
	if len(planting.Tasks) == 0 {
		t.Error("expected at least one planting task")
	}
}

// Regenerating the same plan twice with the same inputs must produce the same dates: the engine is
// deterministic, and a plan that drifted between runs could not be trusted.
func TestRegeneratingWithTheSameInputsGivesTheSamePlan(t *testing.T) {
	h := newHarness(t)
	manager := h.token(t, "manager")
	p := approvedProjection(t, h, "PRJ-SAME", 10)

	first := generatePlan(t, h, manager, map[string]any{"projectionId": p.ID}, http.StatusOK)
	time.Sleep(20 * time.Millisecond)
	second := generatePlan(t, h, manager, map[string]any{"projectionId": p.ID}, http.StatusOK)

	if len(first.Tasks) != len(second.Tasks) {
		t.Fatalf("%d tasks then %d", len(first.Tasks), len(second.Tasks))
	}
	for i := range first.Tasks {
		a, b := first.Tasks[i], second.Tasks[i]
		if a.ActivityCode != b.ActivityCode || a.BlockCode != b.BlockCode ||
			a.PlannedStartDate != b.PlannedStartDate || a.PlannedEndDate != b.PlannedEndDate ||
			a.DurationDays != b.DurationDays {
			t.Fatalf("task %d differs:\n %+v\n %+v", i, a, b)
		}
	}
}
