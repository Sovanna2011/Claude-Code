package service_test

import (
	"context"
	"testing"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
)

// Which version the executive dashboard reports on when nobody asked for one.
//
// This is not a cosmetic default. The dashboard is what a board reads to answer
// "how is the season going", and every KPI on it - the cane target, the
// achievement percentage, the storage forecast - is measured against whichever
// version this picks. Picking a speculative scenario there is not a display
// bug; it is the wrong number in front of the people who act on it.

// draftScenario copies the seeded budget under a given plan type, without
// regenerating it: the question here is which version is chosen, not what is in
// it.
func draftScenario(t *testing.T, h *harness, code string, planType domain.PlanType) domain.PlanVersion {
	t.Helper()
	v, err := h.planning.CopyVersion(h.as(auth.RoleProductionPlanner), service.CopyRequest{
		SourceVersionID: h.seeded.BudgetID, Code: code,
		Description: code, PlanType: planType,
	})
	if err != nil {
		t.Fatalf("copy %s: %v", code, err)
	}
	return v
}

func TestAWhatIfDoesNotBecomeTheSeasonPlan(t *testing.T) {
	h := newHarness(t, 14)
	ctx := h.as(auth.RoleExecutiveViewer)

	before, err := h.analytics.Dashboard(ctx, service.DashboardRequest{SeasonID: h.seeded.SeasonID})
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if before.PlanVersion.ID != h.seeded.BudgetID {
		t.Fatalf("the dashboard reports %s, want the seeded budget", before.PlanVersion.Code)
	}

	// A planner opens a what-if to try a lower recovery. It is created after
	// the budget, so it carries the higher version number - which is exactly
	// what used to decide this.
	whatIf := draftScenario(t, h, "WHATIF-1", domain.PlanTypeWhatIf)
	if whatIf.VersionNo <= before.PlanVersion.VersionNo {
		t.Fatalf("the scenario must be the later version for this test to mean anything: %d against %d",
			whatIf.VersionNo, before.PlanVersion.VersionNo)
	}

	after, err := h.analytics.Dashboard(ctx, service.DashboardRequest{SeasonID: h.seeded.SeasonID})
	if err != nil {
		t.Fatalf("dashboard after the scenario: %v", err)
	}
	if after.PlanVersion.ID != h.seeded.BudgetID {
		t.Fatalf("creating scenario %s changed the season the board reads to %s",
			whatIf.Code, after.PlanVersion.Code)
	}
	if !after.Cane.SeasonTarget.Equal(before.Cane.SeasonTarget) {
		t.Errorf("the cane target moved from %s to %s because a scenario was created",
			before.Cane.SeasonTarget, after.Cane.SeasonTarget)
	}

	// A forecast is a considered revision rather than a game, but it is still
	// not the budget while the budget is there.
	forecast := draftScenario(t, h, "FCST-1", domain.PlanTypeForecast)
	third, err := h.analytics.Dashboard(ctx, service.DashboardRequest{SeasonID: h.seeded.SeasonID})
	if err != nil {
		t.Fatalf("dashboard after the forecast: %v", err)
	}
	if third.PlanVersion.ID != h.seeded.BudgetID {
		t.Fatalf("creating forecast %s changed the season the board reads to %s",
			forecast.Code, third.PlanVersion.Code)
	}

	// Asking for the scenario by name still gets it. The ranking decides the
	// default, not what a planner is allowed to look at.
	named, err := h.analytics.Dashboard(ctx, service.DashboardRequest{
		SeasonID: h.seeded.SeasonID, PlanVersionID: whatIf.ID})
	if err != nil {
		t.Fatalf("dashboard for the named scenario: %v", err)
	}
	if named.PlanVersion.ID != whatIf.ID {
		t.Errorf("asked for %s, got %s", whatIf.Code, named.PlanVersion.Code)
	}
}

func TestAReleasedVersionOutranksTheBudgetDraft(t *testing.T) {
	h := newHarness(t, 0)
	// Whoever submits cannot also approve, so the two halves run as different
	// people - as they would at the mill.
	steps := []struct {
		action domain.PlanAction
		as     context.Context
	}{
		{domain.ActionSubmit, h.as(auth.RoleProductionPlanner)},
		{domain.ActionApprove, h.as(auth.RoleApprover)},
		{domain.ActionRelease, h.as(auth.RoleApprover)},
	}

	revised := draftScenario(t, h, "REV-1", domain.PlanTypeRevised)
	for _, step := range steps {
		v, err := h.planning.Transition(step.as, revised.ID,
			service.TransitionRequest{Action: step.action, RowVersion: revised.RowVersion})
		if err != nil {
			t.Fatalf("%s: %v", step.action, err)
		}
		revised = v
	}

	dash, err := h.analytics.Dashboard(h.as(auth.RoleExecutiveViewer),
		service.DashboardRequest{SeasonID: h.seeded.SeasonID})
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if dash.PlanVersion.ID != revised.ID {
		t.Errorf("the dashboard reports %s, want the released %s",
			dash.PlanVersion.Code, revised.Code)
	}
}
