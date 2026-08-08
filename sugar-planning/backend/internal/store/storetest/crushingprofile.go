package storetest

import (
	"context"
	"testing"

	"github.com/kss/sugarplan/internal/domain"
)

// The crushing profile, held to the same behaviour in both stores.
//
// Ordering is the whole point of this suite. A run-down written 15,000 then
// 12,000 then 8,000 t has to come back in that order: read the other way round
// the season ends at full rate and starts at a crawl, the raw silo fills on a
// different date, and nothing anywhere says a word about it. The same argument
// as the delivery schedule next door, and the same reason it is asserted here
// rather than left to each implementation to remember.
func testCrushingProfile(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	f := seedFixture(t, ctx, s)

	season, err := s.Planning().SaveSeason(ctx, domain.Season{
		CompanyID: f.company, FactoryID: f.factory, Code: "PROFILE-1",
		Name: "Profile test season", StartDate: "2026-12-01", PlannedDays: 137,
		Status: "OPEN",
	}, "seed")
	must(t, err, "save season")
	version, err := s.Planning().SaveVersion(ctx, domain.PlanVersion{
		SeasonID: season.ID, Code: "V1", PlanType: domain.PlanTypeBudget,
		Status: domain.StatusDraft, EffectiveFrom: "2026-12-01",
	}, "seed")
	must(t, err, "save version")

	// A version with no profile has none, rather than an error: an even spread
	// is a legitimate plan and was the only one the generator could make.
	steps, err := s.Planning().ListCrushingSteps(ctx, version.ID)
	must(t, err, "list an empty profile")
	if len(steps) != 0 {
		t.Errorf("a fresh version has %d steps, want none", len(steps))
	}

	// The reference workbook's shape.
	profile := domain.CrushingProfile{
		RampUp:       []domain.CrushingStep{{Rate: domain.DI(17000), Days: 2}},
		CleaningDays: []int{21, 42, 63},
		RunDown: []domain.CrushingStep{
			{Rate: domain.DI(15000), Days: 5},
			{Rate: domain.DI(12000), Days: 3},
			{Rate: domain.DI(8000), Days: 2},
		},
	}
	must(t, s.Planning().ReplaceCrushingSteps(ctx, version.ID,
		domain.RowsFromProfile(version.ID, profile), "seed"), "store the profile")

	steps, err = s.Planning().ListCrushingSteps(ctx, version.ID)
	must(t, err, "read the profile back")
	if len(steps) != 7 {
		t.Fatalf("%d steps, want 7 - one ramp, three wash-outs, three run-down", len(steps))
	}

	// Read back as a profile, it has to be the one that went in.
	got := domain.ProfileFromRows(steps, 0, domain.Zero)
	if len(got.RunDown) != 3 {
		t.Fatalf("%d run-down steps, want 3", len(got.RunDown))
	}
	for i, want := range []int64{15000, 12000, 8000} {
		if !got.RunDown[i].Rate.Equal(domain.DI(want)) {
			t.Errorf("run-down step %d is %s, want %d - the order is the plan",
				i, got.RunDown[i].Rate, want)
		}
	}
	if len(got.RampUp) != 1 || !got.RampUp[0].Rate.Equal(domain.DI(17000)) {
		t.Errorf("ramp up came back as %+v", got.RampUp)
	}
	if len(got.CleaningDays) != 3 {
		t.Errorf("%d wash-out days, want 3", len(got.CleaningDays))
	}

	// And it still builds the curve it described, which is the only thing the
	// storage round trip is for.
	plan, err := domain.BuildCrushingPlan(domain.DI(1000000), 60, got)
	if err != nil {
		t.Fatalf("build from the stored profile: %v", err)
	}
	if !plan.Daily[0].Equal(domain.DI(17000)) {
		t.Errorf("the stored profile opens at %s, want 17000", plan.Daily[0])
	}
	if !plan.Daily[59].Equal(domain.DI(8000)) {
		t.Errorf("the stored profile ends at %s, want 8000", plan.Daily[59])
	}

	// Replacing is a replacement, not an append. A shorter run-down must leave
	// no step of the longer one behind.
	shorter := domain.CrushingProfile{
		RunDown: []domain.CrushingStep{{Rate: domain.DI(9000), Days: 1}},
	}
	must(t, s.Planning().ReplaceCrushingSteps(ctx, version.ID,
		domain.RowsFromProfile(version.ID, shorter), "seed"), "replace the profile")
	steps, err = s.Planning().ListCrushingSteps(ctx, version.ID)
	must(t, err, "read the replaced profile")
	if len(steps) != 1 {
		t.Errorf("%d steps after replacing a seven-step profile with a one-step one, want 1", len(steps))
	}

	// A profile belongs to its version and to no other.
	other, err := s.Planning().SaveVersion(ctx, domain.PlanVersion{
		SeasonID: season.ID, Code: "V2", PlanType: domain.PlanTypeForecast,
		Status: domain.StatusDraft, EffectiveFrom: "2026-12-01",
	}, "seed")
	must(t, err, "save a second version")
	otherSteps, err := s.Planning().ListCrushingSteps(ctx, other.ID)
	must(t, err, "list the second version's profile")
	if len(otherSteps) != 0 {
		t.Errorf("a second version picked up %d steps from the first", len(otherSteps))
	}

	// Clearing it is saying "spread it evenly", which has to be sayable.
	must(t, s.Planning().ReplaceCrushingSteps(ctx, version.ID, nil, "seed"), "clear the profile")
	steps, err = s.Planning().ListCrushingSteps(ctx, version.ID)
	must(t, err, "list after clearing")
	if len(steps) != 0 {
		t.Errorf("%d steps after clearing, want none", len(steps))
	}
}
