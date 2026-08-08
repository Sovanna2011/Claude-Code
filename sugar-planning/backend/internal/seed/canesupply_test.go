package seed_test

import (
	"context"
	"testing"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/seed"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
	"github.com/kss/sugarplan/internal/store/memory"
)

// A year of cane supply behind the 2,300,000 t target.
//
// The figures are asserted rather than described, because the point of the seed
// is that somebody can quote a number off a screen. If the schedule sums to
// 2,319,997 t nobody can.

type supplyHarness struct {
	store    store.Store
	planning *service.Planning
	res      seed.Result
	ctx      context.Context
}

func loadSupply(t *testing.T, actualDays int) *supplyHarness {
	t.Helper()
	s := memory.New()
	planning := service.NewPlanning(s, func() time.Time {
		return time.Date(2026, 12, 15, 6, 0, 0, 0, time.UTC)
	})
	res, err := seed.LoadWithActuals(context.Background(), s, planning, actualDays)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	ctx := auth.WithPrincipal(context.Background(),
		auth.NewPrincipal("t", "t", "Test", "",
			[]string{auth.RoleProductionPlanner, auth.RoleMasterDataAdmin},
			[]string{res.CompanyID}, []string{res.FactoryID}))
	return &supplyHarness{store: s, planning: planning, res: res, ctx: ctx}
}

func (h *supplyHarness) schedule(t *testing.T, versionID string, series domain.Series) []domain.DailyCaneSupply {
	t.Helper()
	rows, err := h.planning.ListCaneSupply(h.ctx, store.PlanFilter{
		VersionIDs: []string{versionID}, Series: series, Top: 100000,
	})
	if err != nil {
		t.Fatalf("read the %s schedule: %v", series, err)
	}
	return rows
}

func TestTheSeasonHasSourcesBehindIt(t *testing.T) {
	h := loadSupply(t, 14)

	if h.res.Supply.Sources != 8 {
		t.Errorf("cane sources = %d, want 8", h.res.Supply.Sources)
	}
	if h.res.Supply.Entries != 8 {
		t.Errorf("supply commitments = %d, want 8", h.res.Supply.Entries)
	}
	// A margin above the crushing target, which is what a mill contracts: cane
	// standing in a field is not cane at the gate.
	if !h.res.Supply.CommittedTons.Equal(domain.D("2320000")) {
		t.Errorf("committed = %s, want 2,320,000", h.res.Supply.CommittedTons)
	}

	plan, err := h.planning.SupplyPlan(h.ctx, h.res.BudgetID)
	if err != nil {
		t.Fatalf("supply plan: %v", err)
	}
	if !plan.Reconcile.TargetTons.Equal(domain.D("2300000.000")) {
		t.Errorf("target = %s, want 2,300,000", plan.Reconcile.TargetTons)
	}
	if !plan.Reconcile.CoveragePct.Equal(domain.D("100.870")) {
		t.Errorf("coverage = %s %%, want 100.870", plan.Reconcile.CoveragePct)
	}
	// 0.87 % over is inside the 2 % tolerance, so the surplus is reported and
	// not complained about.
	for _, w := range plan.Warnings {
		if w.Code == "SUPPLY_COVERAGE" {
			t.Errorf("0.87 %% over tolerance-2 %% must be silent, got %s", w.Detail)
		}
	}

	// Exactly one source is short of lorries, and it is the smallholder zone
	// with the 12 t trucks. The seed is built that way so the warning on the
	// screen has something real behind it.
	haulage := 0
	for _, w := range plan.Warnings {
		if w.Code == "SUPPLY_OVER_HAULAGE" {
			haulage++
		}
	}
	if haulage != 1 {
		t.Errorf("haulage warnings = %d, want exactly 1", haulage)
	}

	// No source is committed to more than its own land grows: the areas and
	// yields were chosen to cover the commitments, not the other way round.
	for _, w := range plan.Warnings {
		if w.Code == "SUPPLY_OVER_YIELD" {
			t.Errorf("a seeded source promises more than it grows: %s", w.Detail)
		}
	}
}

func TestTheDeliveryScheduleSumsToTheCommitments(t *testing.T) {
	h := loadSupply(t, 14)
	rows := h.schedule(t, h.res.BudgetID, domain.SeriesPlan)

	if len(rows) != h.res.Supply.ScheduleRows {
		t.Errorf("read %d schedule rows, the seed reported %d",
			len(rows), h.res.Supply.ScheduleRows)
	}

	total, trips := domain.Zero, 0
	days := map[domain.BusinessDate]bool{}
	for _, r := range rows {
		total = total.Add(r.Tons)
		trips += r.Trips
		days[r.BusinessDate] = true
	}
	// Exactly, not approximately. The generator puts the rounding remainder on
	// the last day of each window precisely so this holds.
	if !total.Equal(domain.D("2320000.000")) {
		t.Errorf("the schedule totals %s t, the commitments are 2,320,000", total)
	}
	if trips == 0 {
		t.Error("a delivery schedule with no lorry movements on it is not a schedule")
	}
	// The windows between them cover the whole 137-day season.
	if len(days) != 137 {
		t.Errorf("the schedule covers %d days, the season is 137", len(days))
	}
}

func TestTheScheduleIsReadInDateOrder(t *testing.T) {
	// The regression this file exists for. The seed reads "the first fourteen
	// days" as a prefix of the schedule, so a store that returns the rows
	// grouped by source instead of by date hands it a different fortnight -
	// which is exactly what happened, and it was silent.
	h := loadSupply(t, 14)
	rows := h.schedule(t, h.res.BudgetID, domain.SeriesPlan)
	for i := 1; i < len(rows); i++ {
		if rows[i].BusinessDate < rows[i-1].BusinessDate {
			t.Fatalf("row %d is %s, after %s", i, rows[i].BusinessDate, rows[i-1].BusinessDate)
		}
	}
	if rows[0].BusinessDate != "2026-12-01" {
		t.Errorf("the schedule starts %s, the season starts 2026-12-01", rows[0].BusinessDate)
	}
}

func TestTheGateRecordsTheSameFortnightAsTheRestOfTheDemonstration(t *testing.T) {
	h := loadSupply(t, 14)
	rows := h.schedule(t, h.res.ActualID, domain.SeriesActual)
	if len(rows) == 0 {
		t.Fatal("no deliveries were recorded at the gate")
	}
	if len(rows) != h.res.Supply.DeliveryRows {
		t.Errorf("read %d deliveries, the seed reported %d", len(rows), h.res.Supply.DeliveryRows)
	}

	days := map[domain.BusinessDate]bool{}
	for _, r := range rows {
		days[r.BusinessDate] = true
	}
	if len(days) != 14 {
		t.Errorf("deliveries cover %d days, the demonstration records 14", len(days))
	}
	if rows[0].BusinessDate != "2026-12-01" {
		t.Errorf("deliveries start %s; they must start with the season, on 2026-12-01",
			rows[0].BusinessDate)
	}
	last := rows[len(rows)-1].BusinessDate
	if last != "2026-12-14" {
		t.Errorf("deliveries end %s, want 2026-12-14", last)
	}

	// The gate does not match the schedule, which is the whole reason the
	// screen comparing them exists.
	planned := domain.Zero
	for _, r := range h.schedule(t, h.res.BudgetID, domain.SeriesPlan) {
		if days[r.BusinessDate] {
			planned = planned.Add(r.Tons)
		}
	}
	actual := domain.Zero
	for _, r := range rows {
		actual = actual.Add(r.Tons)
		if r.Trips == 0 {
			t.Errorf("%s delivered %s t on no lorries", r.BusinessDate, r.Tons)
		}
	}
	if actual.Equal(planned) {
		t.Error("the gate matched the schedule to the tonne; a demonstration " +
			"where nothing ever varies teaches nobody what the variance screen is for")
	}
}
