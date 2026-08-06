package storetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// The costing half of the conformance suite. The rules it pins down are the
// ones the cost run depends on: business keys that make a re-entered rate a
// correction rather than a competitor, effective dating left to the domain, and
// a saved run that reproduces exactly what was reported.

func testCostElements(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	seedFixture(t, ctx, s)
	repo := s.Costing()

	saved, err := repo.SaveElement(ctx, domain.CostElement{
		Code: "FUEL", Name: "Boiler fuel", Category: domain.CategoryEnergy,
		Driver: domain.DriverRunHour, Variable: true, Validity: domain.Validity{Active: true},
	}, "controller")
	must(t, err, "save element")
	if saved.ID == "" || saved.RowVersion != 1 {
		t.Fatalf("saved element = %+v", saved)
	}
	if saved.CreatedBy != "controller" || saved.CreatedAt.IsZero() {
		t.Error("audit fields must be populated on insert")
	}

	// The code is the business key: saving it again corrects the row rather
	// than creating a second element that competes with the first.
	again, err := repo.SaveElement(ctx, domain.CostElement{
		Code: "FUEL", Name: "Boiler fuel oil", Category: domain.CategoryEnergy,
		Driver: domain.DriverRunHour, Variable: true, Validity: domain.Validity{Active: true},
	}, "controller")
	must(t, err, "save element again")
	if again.ID != saved.ID || again.RowVersion != 2 {
		t.Errorf("re-saving by code must update in place, got %+v", again)
	}

	got, err := repo.GetElement(ctx, saved.ID)
	must(t, err, "get element")
	if got.Name != "Boiler fuel oil" || got.Driver != domain.DriverRunHour ||
		got.Category != domain.CategoryEnergy || !got.Variable {
		t.Errorf("round trip lost data: %+v", got)
	}

	if _, err := repo.SaveElement(ctx, domain.CostElement{
		Code: "STAFF", Name: "Salaried staff", Category: domain.CategoryLabour,
		Driver: domain.DriverCalendarDay, Validity: domain.Validity{Active: true},
	}, "controller"); err != nil {
		t.Fatalf("save second element: %v", err)
	}

	page, err := repo.ListElements(ctx, store.ListOptions{})
	must(t, err, "list elements")
	if page.Count != 2 {
		t.Errorf("elements = %d, want 2", page.Count)
	}
	// Ordered by category then code: ENERGY before LABOUR.
	if len(page.Items) == 2 && page.Items[0].Code != "FUEL" {
		t.Errorf("elements are ordered by category then code, got %s first", page.Items[0].Code)
	}

	found, err := repo.ListElements(ctx, store.ListOptions{Search: "salaried"})
	must(t, err, "search elements")
	if found.Count != 1 || found.Items[0].Code != "STAFF" {
		t.Errorf("search = %+v", found.Items)
	}

	if _, err := repo.GetElement(ctx, "3f1d1a1e-0000-4000-8000-000000000000"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("an unknown element must be not found, got %v", err)
	}
}

func testCostRates(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	f := seedFixture(t, ctx, s)
	repo := s.Costing()

	element, err := repo.SaveElement(ctx, domain.CostElement{
		Code: "FUEL", Name: "Boiler fuel", Category: domain.CategoryEnergy,
		Driver: domain.DriverRunHour, Variable: true, Validity: domain.Validity{Active: true},
	}, "controller")
	must(t, err, "save element")

	first, err := repo.SaveRate(ctx, domain.CostRate{
		ElementID: element.ID, FactoryID: f.factory, RateType: domain.RateStandard,
		Rate: domain.D("140.000000"), Currency: "USD", ValidFrom: "2026-12-01",
	}, "controller")
	must(t, err, "save rate")
	if first.ID == "" || first.RowVersion != 1 {
		t.Fatalf("saved rate = %+v", first)
	}

	// Element, factory, type and start date are the business key: re-entering
	// a rate for a date that already has one corrects it.
	corrected, err := repo.SaveRate(ctx, domain.CostRate{
		ElementID: element.ID, FactoryID: f.factory, RateType: domain.RateStandard,
		Rate: domain.D("142.500000"), Currency: "USD", ValidFrom: "2026-12-01",
	}, "controller")
	must(t, err, "correct rate")
	if corrected.ID != first.ID || corrected.RowVersion != 2 {
		t.Errorf("re-entering a rate for the same date must correct it, got %+v", corrected)
	}

	// A different start date is a different rate: a price rise, not a
	// correction.
	if _, err := repo.SaveRate(ctx, domain.CostRate{
		ElementID: element.ID, FactoryID: f.factory, RateType: domain.RateStandard,
		Rate: domain.D("152.000000"), Currency: "USD", ValidFrom: "2027-01-15",
	}, "controller"); err != nil {
		t.Fatalf("save the price rise: %v", err)
	}
	// An actual rate does not collide with a standard one for the same date.
	if _, err := repo.SaveRate(ctx, domain.CostRate{
		ElementID: element.ID, FactoryID: f.factory, RateType: domain.RateActual,
		Rate: domain.D("155.000000"), Currency: "USD", ValidFrom: "2026-12-01",
	}, "controller"); err != nil {
		t.Fatalf("save the actual rate: %v", err)
	}

	all, err := repo.ListRates(ctx, store.CostFilter{FactoryID: f.factory})
	must(t, err, "list rates")
	if len(all) != 3 {
		t.Fatalf("rates = %d, want 3", len(all))
	}

	// The repository hands over everything in force and lets the domain pick,
	// so the two implementations cannot disagree about which rate applies.
	inForce, err := repo.ListRates(ctx, store.CostFilter{
		FactoryID: f.factory, RateType: string(domain.RateStandard), On: "2027-02-01",
	})
	must(t, err, "list rates in force")
	if len(inForce) != 2 {
		t.Fatalf("standard rates in force on 1 Feb = %d, want both open-ended ones", len(inForce))
	}
	picked, ok := domain.RateOn(inForce, element.ID, domain.RateStandard, "2027-02-01")
	if !ok || !picked.Rate.Equal(domain.D("152")) {
		t.Errorf("the later rate wins, got %+v", picked)
	}

	byType, err := repo.ListRates(ctx, store.CostFilter{
		FactoryID: f.factory, RateType: string(domain.RateActual),
	})
	must(t, err, "list actual rates")
	if len(byType) != 1 || !byType[0].Rate.Equal(domain.D("155")) {
		t.Errorf("actual rates = %+v", byType)
	}

	must(t, repo.DeleteRate(ctx, first.ID), "delete rate")
	after, err := repo.ListRates(ctx, store.CostFilter{FactoryID: f.factory})
	must(t, err, "list after delete")
	if len(after) != 2 {
		t.Errorf("rates after the delete = %d, want 2", len(after))
	}
	if err := repo.DeleteRate(ctx, first.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("deleting twice must be not found, got %v", err)
	}
}

func testExchangeRates(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	seedFixture(t, ctx, s)
	repo := s.Costing()

	saved, err := repo.SaveExchangeRate(ctx, domain.ExchangeRate{
		FromCurrency: "USD", ToCurrency: "KHR", Rate: domain.D("4100"), ValidFrom: "2026-12-01",
	}, "controller")
	must(t, err, "save exchange rate")
	if saved.ID == "" || saved.RowVersion != 1 {
		t.Fatalf("saved exchange rate = %+v", saved)
	}

	// The pair and the start date are the business key.
	corrected, err := repo.SaveExchangeRate(ctx, domain.ExchangeRate{
		FromCurrency: "USD", ToCurrency: "KHR", Rate: domain.D("4105"), ValidFrom: "2026-12-01",
	}, "controller")
	must(t, err, "correct exchange rate")
	if corrected.ID != saved.ID || corrected.RowVersion != 2 {
		t.Errorf("re-entering a quotation for the same date must correct it, got %+v", corrected)
	}

	if _, err := repo.SaveExchangeRate(ctx, domain.ExchangeRate{
		FromCurrency: "USD", ToCurrency: "KHR", Rate: domain.D("4150"), ValidFrom: "2027-02-01",
	}, "controller"); err != nil {
		t.Fatalf("save the later quotation: %v", err)
	}

	rates, err := repo.ListExchangeRates(ctx)
	must(t, err, "list exchange rates")
	if len(rates) != 2 {
		t.Fatalf("exchange rates = %d, want 2", len(rates))
	}
	converted, err := domain.Convert(domain.D("100"), "USD", "KHR", rates, "2027-03-01")
	must(t, err, "convert")
	if !converted.Equal(domain.D("415000")) {
		t.Errorf("100 USD in March = %s, want 415,000 KHR", converted)
	}
}

func testCostRuns(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	f := seedFixture(t, ctx, s)
	repo := s.Costing()

	season, err := s.Planning().SaveSeason(ctx, domain.Season{
		CompanyID: f.company, FactoryID: f.factory, Code: "2026-2027",
		Name: "Crushing season 2026-2027", StartDate: "2026-12-01", PlannedDays: 137,
		Status: "OPEN",
	}, "planner")
	must(t, err, "save season")

	element, err := repo.SaveElement(ctx, domain.CostElement{
		Code: "FUEL", Name: "Boiler fuel", Category: domain.CategoryEnergy,
		Driver: domain.DriverRunHour, Variable: true, Validity: domain.Validity{Active: true},
	}, "controller")
	must(t, err, "save element")

	run := domain.CostRun{
		SeasonID: season.ID, FactoryID: f.factory, Code: "DEC-2026",
		From: "2026-12-01", To: "2026-12-31", Currency: "USD",
		Totals: domain.CostTotals{
			PlannedCost: domain.D("100800.00"), ActualCost: domain.D("124000.00"),
			RateVariance: domain.D("12000.00"), UsageVariance: domain.D("11200.00"),
			TotalVariance: domain.D("23200.00"), ActualUnitCost: domain.D("11.27"),
		},
		Lines: []domain.CostLine{{
			ElementID: element.ID, ElementCode: "FUEL", ElementName: "Boiler fuel",
			Category: domain.CategoryEnergy, Driver: domain.DriverRunHour, Variable: true,
			StandardRate: domain.D("140.000000"), ActualRate: domain.D("155.000000"),
			PlannedQty: domain.D("720.000"), ActualQty: domain.D("800.000"),
			PlannedCost: domain.D("100800.00"), ActualCost: domain.D("124000.00"),
			RateVariance: domain.D("12000.00"), UsageVariance: domain.D("11200.00"),
			TotalVariance: domain.D("23200.00"),
			Missing:       []string{"actual rate, charged at standard"},
		}},
	}

	saved, err := repo.SaveRun(ctx, run, "controller")
	must(t, err, "save run")
	if saved.ID == "" || saved.RowVersion != 1 {
		t.Fatalf("saved run = %+v", saved)
	}

	// A saved run has to reproduce exactly what was reported, months later,
	// after the rates have moved on.
	got, err := repo.GetRun(ctx, saved.ID)
	must(t, err, "get run")
	if !got.Totals.TotalVariance.Equal(domain.D("23200")) {
		t.Errorf("total variance = %s, want 23,200", got.Totals.TotalVariance)
	}
	if !got.Totals.ActualUnitCost.Equal(domain.D("11.27")) {
		t.Errorf("unit cost = %s, want 11.27", got.Totals.ActualUnitCost)
	}
	if len(got.Lines) != 1 {
		t.Fatalf("lines = %d, want 1", len(got.Lines))
	}
	line := got.Lines[0]
	if !line.StandardRate.Equal(domain.D("140")) || !line.ActualRate.Equal(domain.D("155")) {
		t.Errorf("the run must keep the rates it used: %+v", line)
	}
	if !line.RateVariance.Add(line.UsageVariance).Equal(line.TotalVariance) {
		t.Error("a stored line must still reconcile")
	}
	if len(line.Missing) != 1 || line.Missing[0] != "actual rate, charged at standard" {
		t.Errorf("the run must keep what it could not find: %+v", line.Missing)
	}
	if line.ElementCode != "FUEL" || line.DriverUnit != "h" {
		t.Errorf("a stored line must name its element and unit: %+v", line)
	}

	// Season and code are the business key: re-running a period replaces it
	// rather than leaving two answers to the same question.
	run.Lines = nil
	run.Totals.TotalVariance = domain.D("0.00")
	rerun, err := repo.SaveRun(ctx, run, "controller")
	must(t, err, "re-run the period")
	if rerun.ID != saved.ID || rerun.RowVersion != 2 {
		t.Errorf("re-running a period must replace it, got %+v", rerun)
	}
	replaced, err := repo.GetRun(ctx, saved.ID)
	must(t, err, "get the replaced run")
	if len(replaced.Lines) != 0 {
		t.Errorf("an element dropped from the run must disappear, got %+v", replaced.Lines)
	}

	page, err := repo.ListRuns(ctx, store.CostFilter{SeasonID: season.ID})
	must(t, err, "list runs")
	if page.Count != 1 {
		t.Errorf("runs = %d, want 1", page.Count)
	}
	if len(page.Items) == 1 && len(page.Items[0].Lines) != 0 {
		t.Error("a list carries the totals only; the lines belong to the detail view")
	}

	if _, err := repo.GetRun(ctx, "3f1d1a1e-0000-4000-8000-000000000000"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("an unknown run must be not found, got %v", err)
	}
}
