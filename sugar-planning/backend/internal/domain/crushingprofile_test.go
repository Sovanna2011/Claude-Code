package domain_test

import (
	"testing"

	"github.com/kss/sugarplan/internal/domain"
)

// The crushing profile, held to the reference workbook.
//
// These are not invented figures. `ProductionPlan_2627_2.3mt Rev.1` is the
// mill's own daily plan for the 2026/27 season, and the assertions below are
// read off it. If the generator can reproduce that curve from a profile, a
// planner can change the target and get a plan that still looks like a season.

// referenceProfile is Kampong Speu's 2026/27 shape.
func referenceProfile() domain.CrushingProfile {
	return domain.CrushingProfile{
		RampUp:          []domain.CrushingStep{{Rate: domain.DI(17000), Days: 2}},
		PreCleaningRate: domain.DI(9000),
		CleaningDays:    []int{21, 42, 63, 84, 105, 119},
		RunDown: []domain.CrushingStep{
			{Rate: domain.DI(15000), Days: 5},
			{Rate: domain.DI(12000), Days: 3},
			{Rate: domain.DI(8000), Days: 2},
			{Rate: domain.DI(4000), Days: 2},
			{Rate: domain.DI(3000), Days: 2},
		},
	}
}

func TestTheProfileReproducesTheWorkbookCurve(t *testing.T) {
	plan, err := domain.BuildCrushingPlan(domain.DI(2300000), 137, referenceProfile())
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(plan.Daily) != 137 {
		t.Fatalf("%d days, want 137", len(plan.Daily))
	}

	// The plateau the mill runs at. This is solved, not entered: it is whatever
	// rate makes 2,300,000 t fit around the shoulders and the wash-outs.
	if !plan.PlateauRate.Equal(domain.DI(19000)) {
		t.Errorf("plateau rate = %s, want the workbook's 19,000", plan.PlateauRate)
	}
	if plan.PlateauDays != 109 {
		t.Errorf("%d days at full rate, the workbook has 109", plan.PlateauDays)
	}
	if plan.CleaningDays != 6 {
		t.Errorf("%d wash-outs, the workbook has 6", plan.CleaningDays)
	}

	// Day by day against the workbook, at every point the rate changes.
	for _, c := range []struct {
		day  int // one-based campaign day
		tons int64
		why  string
	}{
		{1, 17000, "the opening day, while the boilers come up"},
		{2, 17000, "still starting up"},
		{3, 19000, "full rate"},
		{19, 19000, "still full rate"},
		{20, 9000, "half rate, the day before the first wash-out"},
		{21, 0, "the first wash-out"},
		{22, 19000, "back to full rate"},
		{41, 9000, "winding down for the second wash-out"},
		{42, 0, "the second wash-out"},
		{104, 9000, "winding down for the fifth"},
		{105, 0, "the fifth wash-out"},
		{118, 9000, "winding down for the last"},
		{119, 0, "the last wash-out, pulled forward off the cadence"},
		{123, 19000, "the last full-rate day"},
		{124, 15000, "the run-down starts"},
		{128, 15000, "still 15,000"},
		{129, 12000, "the cane is thinning"},
		{131, 12000, ""},
		{132, 8000, ""},
		{134, 4000, ""},
		{136, 3000, ""},
		{137, 3000, "the last day of the season"},
	} {
		got := plan.Daily[c.day-1]
		if !got.Equal(domain.DI(c.tons)) {
			t.Errorf("day %d = %s, the workbook says %d — %s", c.day, got, c.tons, c.why)
		}
	}

	// And it adds up to the target exactly, which is the point of solving the
	// plateau rather than entering it.
	total := domain.Zero
	for _, v := range plan.Daily {
		total = total.Add(v)
	}
	if !total.Equal(domain.DI(2300000)) {
		t.Errorf("the season totals %s, want exactly 2,300,000", total)
	}
}

func TestTheSeasonStillAddsUpWhenTheTargetIsNotRound(t *testing.T) {
	// The remainder has to land somewhere. A target that does not divide by the
	// plateau days is the ordinary case the moment anybody edits one.
	plan, err := domain.BuildCrushingPlan(domain.D("2300001"), 137, referenceProfile())
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	total := domain.Zero
	for _, v := range plan.Daily {
		total = total.Add(v)
	}
	if !total.Equal(domain.D("2300001")) {
		t.Errorf("the season totals %s, want exactly 2300001", total)
	}
}

func TestAnEmptyProfileIsAnEvenSpread(t *testing.T) {
	// The behaviour before profiles existed, kept: a plan with nothing said
	// about its shape is still a plan.
	plan, err := domain.BuildCrushingPlan(domain.DI(1000), 10, domain.CrushingProfile{})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if plan.PlateauDays != 10 {
		t.Errorf("%d plateau days, want all 10", plan.PlateauDays)
	}
	for i, v := range plan.Daily {
		if !v.Equal(domain.DI(100)) {
			t.Errorf("day %d = %s, want 100", i+1, v)
		}
	}
}

func TestTheCadenceStopsTheMillEveryNthDay(t *testing.T) {
	plan, err := domain.BuildCrushingPlan(domain.DI(1000), 10, domain.CrushingProfile{
		CleaningEveryDays: 4,
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if plan.CleaningDays != 2 {
		t.Errorf("%d wash-outs in ten days at a four-day cadence, want 2", plan.CleaningDays)
	}
	for _, day := range []int{4, 8} {
		if !plan.Daily[day-1].Equal(domain.Zero) {
			t.Errorf("day %d should be a wash-out, got %s", day, plan.Daily[day-1])
		}
	}
}

func TestAProfileThatCannotFitTheTargetSaysSo(t *testing.T) {
	// The shoulders alone come to more than the season. Silently scaling them
	// down would produce a plan nobody wrote.
	_, err := domain.BuildCrushingPlan(domain.DI(1000), 137, referenceProfile())
	if err == nil {
		t.Error("a target smaller than the shoulders must be refused, not scaled")
	}
}

func TestTheShouldersMayNotOverlap(t *testing.T) {
	_, err := domain.BuildCrushingPlan(domain.DI(100000), 3, domain.CrushingProfile{
		RampUp:  []domain.CrushingStep{{Rate: domain.DI(100), Days: 2}},
		RunDown: []domain.CrushingStep{{Rate: domain.DI(100), Days: 2}},
	})
	if err == nil {
		t.Error("a ramp up and a run down that overlap must be refused")
	}
}

func TestAWashOutInsideAShoulderIsNotCountedTwice(t *testing.T) {
	// The run-down already says what happens on those days. A cadence that
	// lands there must not overwrite the planner's own figure.
	plan, err := domain.BuildCrushingPlan(domain.DI(1000), 10, domain.CrushingProfile{
		RunDown:           []domain.CrushingStep{{Rate: domain.DI(50), Days: 2}},
		CleaningEveryDays: 10,
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !plan.Daily[9].Equal(domain.DI(50)) {
		t.Errorf("the last day is %s; the run-down said 50 and a wash-out must not "+
			"overwrite it", plan.Daily[9])
	}
}
