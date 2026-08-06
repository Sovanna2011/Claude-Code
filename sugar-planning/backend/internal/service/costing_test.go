package service_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
)

// These tests run against the seeded reference scenario, so the driver
// quantities are the real ones: 2,300,000 t of cane over 137 days producing
// 242,100 t of finished sugar.

func costHarness(t *testing.T, actualDays int) (*harness, *service.Costing) {
	t.Helper()
	h := newHarness(t, actualDays)
	return h, service.NewCosting(h.store, h.planning, fixedClock())
}

func TestTheSeededScenarioCanBeCosted(t *testing.T) {
	h, costing := costHarness(t, 0)
	ctx := h.as(auth.RoleCostController)

	result, err := costing.Run(ctx, service.CostRunRequest{SeasonID: h.seeded.SeasonID})
	if err != nil {
		t.Fatalf("cost run: %v", err)
	}

	if len(result.Lines) != 12 {
		t.Fatalf("cost lines = %d, want the twelve seeded elements", len(result.Lines))
	}
	// Every element must be priced. A line reading zero because nobody entered
	// a rate is the failure this checks for.
	for _, line := range result.Lines {
		if line.StandardRate.LessThanOrEqual(domain.Zero) {
			t.Errorf("%s has no standard rate: %+v", line.ElementCode, line.Missing)
		}
	}
	for _, w := range result.Warnings {
		if w.Code == "COST_NO_STANDARD_RATE" || w.Code == "COST_NO_EXCHANGE_RATE" {
			t.Errorf("the seeded scenario must cost cleanly, got %s: %s", w.Code, w.Detail)
		}
	}

	// The drivers are the plan's own figures, so the cost report and the
	// operations report cannot disagree.
	if !result.Planned.CaneTons.Equal(domain.D("2300000")) {
		t.Errorf("cane driver = %s, want the planned 2,300,000 t", result.Planned.CaneTons)
	}
	if !result.Planned.SugarTons.Equal(domain.D("242100")) {
		t.Errorf("sugar driver = %s, want the planned 242,100 t of finished goods",
			result.Planned.SugarTons)
	}

	// Cane is quoted in riel and the company reports in dollars.
	for _, line := range result.Lines {
		if line.ElementCode != "CANE" {
			continue
		}
		if !line.StandardRate.Equal(domain.D("22.5")) {
			t.Errorf("cane rate = %s, want 92,250 KHR converted at 4,100 to 22.50 USD",
				line.StandardRate)
		}
		// 2,300,000 t × 22.50
		if !line.PlannedCost.Equal(domain.D("51750000")) {
			t.Errorf("cane cost = %s, want 51,750,000", line.PlannedCost)
		}
	}

	if result.Currency != "USD" {
		t.Errorf("currency = %s, want the company's USD", result.Currency)
	}
	if result.Totals.PlannedUnitCost.LessThanOrEqual(domain.Zero) {
		t.Error("the plan must produce a cost per ton")
	}
	// A sanity band rather than an exact figure: the point is that the number
	// is a plausible cost per ton of sugar, not that it is one particular value.
	if result.Totals.PlannedUnitCost.LessThan(domain.D("100")) ||
		result.Totals.PlannedUnitCost.GreaterThan(domain.D("1000")) {
		t.Errorf("unit cost = %s, outside any plausible range for a ton of sugar",
			result.Totals.PlannedUnitCost)
	}

	// Fixed and variable must account for the whole actual cost.
	if !result.Totals.FixedCost.Add(result.Totals.VariableCost).
		Equal(result.Totals.ActualCost) {
		t.Errorf("fixed %s + variable %s != actual %s",
			result.Totals.FixedCost, result.Totals.VariableCost, result.Totals.ActualCost)
	}
}

func TestACostRunReconcilesAgainstRecordedActuals(t *testing.T) {
	// Fourteen days of actuals are seeded, so the run has a real variance.
	h, costing := costHarness(t, 14)
	ctx := h.as(auth.RoleCostController)

	result, err := costing.Run(ctx, service.CostRunRequest{
		SeasonID: h.seeded.SeasonID, From: "2026-12-01", To: "2026-12-14",
	})
	if err != nil {
		t.Fatalf("cost run: %v", err)
	}

	if result.Actual.CaneTons.LessThanOrEqual(domain.Zero) {
		t.Fatalf("no cane was recorded in the first fortnight: %+v", result.Actual)
	}
	if result.Actual.RunHours.LessThanOrEqual(domain.Zero) {
		t.Errorf("run hours = %s, want the recorded hours", result.Actual.RunHours)
	}

	// The whole point of the model: the two halves reconstruct the total.
	if !domain.VarianceCheck(result.Totals.ActualCost, result.Totals.PlannedCost,
		result.Totals.RateVariance, result.Totals.UsageVariance) {
		t.Errorf("the run does not reconcile: actual %s, planned %s, rate %s, usage %s",
			result.Totals.ActualCost, result.Totals.PlannedCost,
			result.Totals.RateVariance, result.Totals.UsageVariance)
	}

	// Fuel was invoiced at 155.00 against a standard of 140.00, so its rate
	// variance is unfavourable and nothing else's is.
	for _, line := range result.Lines {
		switch line.ElementCode {
		case "FUEL":
			if !line.ActualRate.Equal(domain.D("155")) {
				t.Errorf("fuel actual rate = %s, want the invoiced 155.00", line.ActualRate)
			}
			if line.RateVariance.LessThanOrEqual(domain.Zero) {
				t.Errorf("fuel was invoiced above budget, so the rate variance is unfavourable: %s",
					line.RateVariance)
			}
		case "LIME":
			// Lime came in under budget.
			if line.RateVariance.GreaterThanOrEqual(domain.Zero) {
				t.Errorf("lime was invoiced below budget, so the rate variance is favourable: %s",
					line.RateVariance)
			}
		case "STAFF":
			// Nobody has invoiced the payroll differently, so it is charged at
			// standard and its rate variance is zero.
			if !line.RateVariance.IsZero() {
				t.Errorf("an uninvoiced element has no rate variance, got %s", line.RateVariance)
			}
		}
	}
}

func TestARangeCostsOnlyItsOwnDays(t *testing.T) {
	h, costing := costHarness(t, 14)
	ctx := h.as(auth.RoleCostController)

	week, err := costing.Run(ctx, service.CostRunRequest{
		SeasonID: h.seeded.SeasonID, From: "2026-12-01", To: "2026-12-07",
	})
	if err != nil {
		t.Fatalf("cost the first week: %v", err)
	}
	fortnight, err := costing.Run(ctx, service.CostRunRequest{
		SeasonID: h.seeded.SeasonID, From: "2026-12-01", To: "2026-12-14",
	})
	if err != nil {
		t.Fatalf("cost the fortnight: %v", err)
	}

	if !week.Planned.CaneTons.LessThan(fortnight.Planned.CaneTons) {
		t.Errorf("a week (%s t) must cost less cane than a fortnight (%s t)",
			week.Planned.CaneTons, fortnight.Planned.CaneTons)
	}
	if !week.Totals.PlannedCost.LessThan(fortnight.Totals.PlannedCost) {
		t.Errorf("a week costs %s and a fortnight %s",
			week.Totals.PlannedCost, fortnight.Totals.PlannedCost)
	}
	// The calendar-day driver follows the range, so a salaried crew is charged
	// for seven days and fourteen.
	if !week.Planned.CalendarDays.Equal(domain.DI(7)) {
		t.Errorf("planned calendar days = %s, want 7", week.Planned.CalendarDays)
	}
	if !fortnight.Planned.CalendarDays.Equal(domain.DI(14)) {
		t.Errorf("planned calendar days = %s, want 14", fortnight.Planned.CalendarDays)
	}
}

func TestASavedRunReproducesTheFigureLater(t *testing.T) {
	h, costing := costHarness(t, 14)
	ctx := h.as(auth.RoleCostController)

	saved, err := costing.Run(ctx, service.CostRunRequest{
		SeasonID: h.seeded.SeasonID, From: "2026-12-01", To: "2026-12-14",
		Save: true, Code: "DEC-W1-2", Note: "First fortnight",
	})
	if err != nil {
		t.Fatalf("save the run: %v", err)
	}
	if !saved.Saved || saved.Run.ID == "" {
		t.Fatalf("the run was not saved: %+v", saved.Run)
	}

	// The rates move on, and the saved figure must not.
	elements, err := costing.ListElements(ctx, store.ListOptions{Top: 100})
	if err != nil {
		t.Fatalf("list elements: %v", err)
	}
	var fuel domain.CostElement
	for _, e := range elements.Items {
		if e.Code == "FUEL" {
			fuel = e
		}
	}
	if _, err := costing.SaveRate(ctx, domain.CostRate{
		ElementID: fuel.ID, FactoryID: h.seeded.FactoryID, RateType: domain.RateStandard,
		Rate: domain.D("999.000000"), Currency: "USD", ValidFrom: "2026-12-01",
	}); err != nil {
		t.Fatalf("change the rate: %v", err)
	}

	reread, err := costing.GetRun(ctx, saved.Run.ID)
	if err != nil {
		t.Fatalf("re-read the run: %v", err)
	}
	if !reread.Totals.ActualCost.Equal(saved.Totals.ActualCost) {
		t.Errorf("the saved figure moved with the rates: %s, was %s",
			reread.Totals.ActualCost, saved.Totals.ActualCost)
	}
	for _, line := range reread.Lines {
		if line.ElementCode == "FUEL" && !line.StandardRate.Equal(domain.D("140")) {
			t.Errorf("the saved line must keep the rate it used, got %s", line.StandardRate)
		}
	}

	// Season and code are the business key, so re-running a period replaces it
	// rather than leaving two answers to the same question.
	if _, err := costing.Run(ctx, service.CostRunRequest{
		SeasonID: h.seeded.SeasonID, From: "2026-12-01", To: "2026-12-14",
		Save: true, Code: "DEC-W1-2",
	}); err != nil {
		t.Fatalf("re-run the period: %v", err)
	}
	runs, err := costing.ListRuns(ctx, store.CostFilter{SeasonID: h.seeded.SeasonID})
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if runs.Count != 1 {
		t.Errorf("saved runs = %d, want 1 after a re-run of the same period", runs.Count)
	}
}

func TestCostFiguresAreNotVisibleToEverybody(t *testing.T) {
	h, costing := costHarness(t, 0)

	// A planner reads plans, not what the factory pays for cane.
	planner := h.as(auth.RoleProductionPlanner)
	if _, err := costing.Run(planner, service.CostRunRequest{
		SeasonID: h.seeded.SeasonID,
	}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("a planner may not cost the season, got %v", err)
	}
	if _, err := costing.ListRates(planner, store.CostFilter{}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("a planner may not read the rates, got %v", err)
	}

	// A warehouse operator certainly may not.
	if _, err := costing.ListRates(h.keeper(), store.CostFilter{}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("a keeper may not read the rates, got %v", err)
	}

	// The executive reads the cost but does not set the rates.
	executive := h.as(auth.RoleExecutiveViewer)
	if _, err := costing.Run(executive, service.CostRunRequest{
		SeasonID: h.seeded.SeasonID,
	}); err != nil {
		t.Errorf("an executive may read the cost: %v", err)
	}
	if _, err := costing.SaveRate(executive, domain.CostRate{
		ElementID: "x", FactoryID: h.seeded.FactoryID, RateType: domain.RateStandard,
		Rate: domain.D("1"), Currency: "USD", ValidFrom: "2026-12-01",
	}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("an executive may not set a rate, got %v", err)
	}

	// Saving a run is a write even for somebody who may read one.
	if _, err := costing.Run(executive, service.CostRunRequest{
		SeasonID: h.seeded.SeasonID, Save: true, Code: "X",
	}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("saving a run needs the write permission, got %v", err)
	}
}

func TestARateIsValidatedBeforeItIsStored(t *testing.T) {
	h, costing := costHarness(t, 0)
	ctx := h.as(auth.RoleCostController)

	_, err := costing.SaveRate(ctx, domain.CostRate{
		FactoryID: h.seeded.FactoryID, RateType: "GUESS",
		Rate: domain.D("-5"), Currency: "DOLLARS", ValidFrom: "not-a-date",
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("an invalid rate must be refused, got %v", err)
	}
	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("the refusal must name the fields, got %v", err)
	}
	fields := map[string]bool{}
	for _, e := range verr.Errors {
		fields[e.Field] = true
	}
	for _, want := range []string{"elementId", "rateType", "rate", "currency", "validFrom"} {
		if !fields[want] {
			t.Errorf("%s was not reported: %+v", want, verr.Errors)
		}
	}

	// A rate against an element that does not exist is refused rather than
	// stored to dangle.
	if _, err := costing.SaveRate(ctx, domain.CostRate{
		ElementID: "3f1d1a1e-0000-4000-8000-000000000000", FactoryID: h.seeded.FactoryID,
		RateType: domain.RateStandard, Rate: domain.D("10"), Currency: "USD",
		ValidFrom: "2026-12-01",
	}); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("a rate against an unknown element must be refused, got %v", err)
	}
}

func TestAnElementMustNameAKnownDriver(t *testing.T) {
	h, costing := costHarness(t, 0)
	ctx := h.as(auth.RoleCostController)

	_, err := costing.SaveElement(ctx, domain.CostElement{
		Code: "X", Name: "Mystery", Category: domain.CategoryOverhead, Driver: "PER_WISH",
		Validity: domain.Validity{Active: true},
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("an unknown driver must be refused, got %v", err)
	}
	if !strings.Contains(err.Error(), "cane tons") {
		t.Errorf("the refusal should list the drivers that exist: %v", err)
	}
}

func TestCostRunIsAudited(t *testing.T) {
	h, costing := costHarness(t, 14)
	ctx := h.as(auth.RoleCostController)

	saved, err := costing.Run(ctx, service.CostRunRequest{
		SeasonID: h.seeded.SeasonID, From: "2026-12-01", To: "2026-12-14",
		Save: true, Code: "AUDITED",
	})
	if err != nil {
		t.Fatalf("cost run: %v", err)
	}

	events, err := h.store.Audit().List(ctx, store.AuditFilter{
		Entity: "cost_run", EntityID: saved.Run.ID,
	})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if events.Count != 1 {
		t.Fatalf("audit events = %d, want 1", events.Count)
	}
	if events.Items[0].Action != "COST_RUN" {
		t.Errorf("action = %s, want COST_RUN", events.Items[0].Action)
	}
	if !strings.Contains(events.Items[0].Reason, "USD") {
		t.Errorf("the audit record should say what was costed and in what: %q",
			events.Items[0].Reason)
	}
	if events.Items[0].After == "" {
		t.Error("the audit record must carry the totals it produced")
	}
}

// TestTheDefaultRangeEndsWhereTheRecordingEnds pins the interpretation the
// screen depends on. Costing a whole season's plan against a fortnight of
// actuals gives a variance of minus ninety per cent and a cost per ton of zero:
// arithmetically right, and useless, because it says only that the season has
// not finished.
func TestTheDefaultRangeEndsWhereTheRecordingEnds(t *testing.T) {
	h, costing := costHarness(t, 14)
	ctx := h.as(auth.RoleCostController)

	result, err := costing.Run(ctx, service.CostRunRequest{SeasonID: h.seeded.SeasonID})
	if err != nil {
		t.Fatalf("cost run: %v", err)
	}

	if result.To != "2026-12-14" {
		t.Errorf("the default range ends %s, want the last recorded day 2026-12-14", result.To)
	}
	// Both sides now cover the same fourteen days, so the comparison means
	// something: the actual cost per ton is a real figure near the planned one
	// rather than zero.
	if result.Totals.ActualUnitCost.LessThanOrEqual(domain.Zero) {
		t.Errorf("actual unit cost = %s, want a real figure", result.Totals.ActualUnitCost)
	}
	if result.Totals.VariancePct.Abs().GreaterThan(domain.D("25")) {
		t.Errorf("variance = %s%%, which suggests the two sides cover different days",
			result.Totals.VariancePct)
	}
	if !result.Planned.CalendarDays.Equal(result.Actual.CalendarDays) {
		t.Errorf("planned %s days against actual %s days; the range must cover both equally",
			result.Planned.CalendarDays, result.Actual.CalendarDays)
	}
}

// TestASeasonWithNoActualsStillCosts covers the other end: before anybody has
// posted, the plan is all there is to report and the run must still produce it.
func TestASeasonWithNoActualsStillCosts(t *testing.T) {
	h, costing := costHarness(t, 0)
	ctx := h.as(auth.RoleCostController)

	result, err := costing.Run(ctx, service.CostRunRequest{SeasonID: h.seeded.SeasonID})
	if err != nil {
		t.Fatalf("cost run: %v", err)
	}
	if result.To != "2027-04-16" {
		t.Errorf("with nothing recorded the range runs to the season end, got %s", result.To)
	}
	if result.Totals.PlannedCost.LessThanOrEqual(domain.Zero) {
		t.Error("the plan must still be costed")
	}
	if !result.Totals.ActualSugarTons.IsZero() {
		t.Errorf("nothing was produced, got %s t", result.Totals.ActualSugarTons)
	}
}
