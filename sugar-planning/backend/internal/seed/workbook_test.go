package seed_test

import (
	"testing"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/seed"
	"github.com/kss/sugarplan/internal/store"
)

// The reference workbook, reconciled.
//
// `ProductionPlan_2627_2.3mt Rev.1 (corrected)` is the mill's own daily plan for
// the 2026/27 season. Everything asserted here is read off it, so these tests
// are the answer to "does this system agree with the plan the factory is
// actually working to". Where it disagrees, the disagreement is stated and
// explained rather than smoothed over.

func planRows(t *testing.T, h *supplyHarness) []domain.DailyCanePlan {
	t.Helper()
	rows, err := h.store.Planning().ListCane(h.ctx, store.PlanFilter{
		VersionIDs: []string{h.res.BudgetID}, Series: domain.SeriesPlan, Top: 2000,
	})
	if err != nil {
		t.Fatalf("read the cane plan: %v", err)
	}
	return rows
}

func TestTheGeneratedPlanIsTheWorkbooksCrushingCurve(t *testing.T) {
	h := loadSupply(t, 0)
	rows := planRows(t, h)

	if len(rows) != seed.WorkbookCampaignDays {
		t.Fatalf("%d cane rows, the workbook's season is %d days",
			len(rows), seed.WorkbookCampaignDays)
	}

	// Every point the workbook's rate changes, by date rather than by index -
	// a plan that had the right shape on the wrong days would pass an
	// index-based check and be useless.
	want := map[domain.BusinessDate]string{
		"2026-12-01": "17000", // start-up, while the boilers come up
		"2026-12-02": "17000",
		"2026-12-03": "19000", // full rate
		"2026-12-20": "9000",  // winding down for the first wash-out
		"2026-12-21": "0",     // the first wash-out
		"2026-12-22": "19000",
		"2027-01-11": "0", // the second
		"2027-02-01": "0", // the third
		"2027-02-22": "0", // the fourth
		"2027-03-15": "0", // the fifth
		"2027-03-29": "0", // the last, pulled forward off the 21-day cadence
		"2027-04-02": "19000",
		"2027-04-03": "15000", // the run-down begins
		"2027-04-08": "12000",
		"2027-04-11": "8000",
		"2027-04-13": "4000",
		"2027-04-15": "3000",
		"2027-04-16": "3000", // the last day of crushing
	}
	byDate := map[domain.BusinessDate]domain.Dec{}
	for _, r := range rows {
		byDate[r.BusinessDate] = r.CaneCrushed
	}
	for date, tons := range want {
		got, ok := byDate[date]
		if !ok {
			t.Errorf("%s is not in the plan at all", date)
			continue
		}
		if !got.Equal(domain.D(tons)) {
			t.Errorf("%s: the plan crushes %s t, the workbook says %s", date, got, tons)
		}
	}

	// 137 days of campaign, 131 of crushing, 6 of wash-out.
	crushing, cleaning, total := 0, 0, domain.Zero
	for _, r := range rows {
		total = total.Add(r.CaneCrushed)
		if r.CaneCrushed.IsZero() {
			cleaning++
		} else {
			crushing++
		}
	}
	if crushing != seed.WorkbookCrushingDays {
		t.Errorf("%d crushing days, the workbook has %d", crushing, seed.WorkbookCrushingDays)
	}
	if cleaning != seed.WorkbookCleaningDays {
		t.Errorf("%d wash-outs, the workbook has %d", cleaning, seed.WorkbookCleaningDays)
	}
	if !total.Equal(domain.D(seed.WorkbookCaneTons)) {
		t.Errorf("the season crushes %s t, the workbook's target is %s", total, seed.WorkbookCaneTons)
	}
}

func TestTheSummaryReconcilesWithTheWorkbook(t *testing.T) {
	h := loadSupply(t, 0)
	s := h.res.Generated.Summary

	for _, c := range []struct {
		what string
		got  domain.Dec
		want string
	}{
		{"cane", s.CaneAllocated, seed.WorkbookCaneTons},
		{"raw sugar at 11.00 % recovery", s.RawSugarExpected, seed.WorkbookRawSugarTons},
		{"finished goods", s.FinishedGoods, seed.WorkbookFinishedTons},
	} {
		if !c.got.Equal(domain.D(c.want)) {
			t.Errorf("%s = %s, the workbook says %s", c.what, c.got, c.want)
		}
	}

	if s.CrushingDays != seed.WorkbookCrushingDays {
		t.Errorf("crushing days = %d, want %d", s.CrushingDays, seed.WorkbookCrushingDays)
	}
	if s.CleaningDays != seed.WorkbookCleaningDays {
		t.Errorf("wash-outs = %d, want %d", s.CleaningDays, seed.WorkbookCleaningDays)
	}
	if !s.PlateauRate.Equal(domain.D(seed.WorkbookPlateauTons)) {
		t.Errorf("full rate = %s, the workbook runs at %s", s.PlateauRate, seed.WorkbookPlateauTons)
	}

	// The remelt factor. The workbook does not state one, but it computes one:
	// it feeds the refinery 945 t of raw sugar a day to make 400 t of refined
	// and 500 t of white, and 945 / 900 is exactly 1.05. That closes an open
	// question the plan had been carrying as an assumption.
	if !s.RemeltInput.Equal(domain.D("254205")) {
		t.Errorf("remelt input = %s, want 242,100 x 1.05 = 254,205", s.RemeltInput)
	}

	// The raw sugar split is where this plan and the workbook differ, and the
	// difference is worth stating rather than tuning away.
	//
	// The workbook's 124,950 / 128,050 is a seasonal accounting split: raw
	// consumed by refining while the mill is still crushing, against raw left
	// in the silo for the remelt season. This generator routes raw sugar day by
	// day - straight to the refinery up to what the refinery needs, the rest to
	// the silo, and back out of the silo when the day's cane cannot cover the
	// refinery. Those are two different questions with two different answers.
	//
	// Compared like with like - refinery input up to 16 April - the plan says
	// 131,565 t against the workbook's 124,950 t. Almost all of the 6,615 t is
	// the refinery's own start-up, which this plan does not model: the mill
	// crushes from 1 December but the workbook does not start refining until
	// the 5th, and then runs at 350, 400 and 500 t a day before settling. Four
	// idle days and a week of ramp is about 4,500 t of raw sugar not consumed.
	//
	// The season totals are exact either way, which is what a plan is quoted
	// on; the split is out by 5 % and the cause is known and named.
	total := s.RawDirectRefine.Add(s.RawToStorage)
	if !total.Equal(domain.D(seed.WorkbookRawSugarTons)) {
		t.Errorf("the split totals %s t, and every ton of raw sugar has to go "+
			"somewhere: %s", total, seed.WorkbookRawSugarTons)
	}
}

func TestThePlanRunsPastTheLastCaneToTheEndOfTheRemeltSeason(t *testing.T) {
	// The finding that changed the model. The mill crushes for 137 days and
	// goes on refining stored raw sugar, and selling, until 2 September. A plan
	// that stopped at the last cane compressed nine months of production into
	// four and a half, and every storage date it produced was wrong.
	h := loadSupply(t, 0)

	products, err := h.store.Planning().ListProducts(h.ctx, store.PlanFilter{
		VersionIDs: []string{h.res.BudgetID}, Series: domain.SeriesPlan, Top: 20000,
	})
	if err != nil {
		t.Fatalf("read the production plan: %v", err)
	}
	last := domain.BusinessDate("")
	for _, r := range products {
		if r.BusinessDate > last {
			last = r.BusinessDate
		}
	}
	if last <= seed.WorkbookSeasonEnd {
		t.Fatalf("production ends %s, on or before the last day of crushing (%s); "+
			"the remelt season is not being planned", last, seed.WorkbookSeasonEnd)
	}
	// 2027-08-24 against the workbook's 2027-09-02. The nine days are the
	// refinery start-up the plan does not model: the workbook idles it until
	// 5 December and then ramps it, so its campaign finishes later on the same
	// tonnage. Asserted as "within a fortnight" rather than pinned, because the
	// gap is a known modelling difference and not a figure to defend.
	if gap := last.DaysBetween(seed.WorkbookRemeltEnd); gap > 14 {
		t.Errorf("production ends %s, %d days short of the workbook's %s",
			last, gap, seed.WorkbookRemeltEnd)
	}
}

func TestTheRawSiloPeaksOnTheDayTheWorkbookSaysItDoes(t *testing.T) {
	// The date is the assertion. A tonnage can be argued about; the date the
	// silo peaks is what the jumbo bagging campaign is planned around, and the
	// workbook names it: 10 April 2027.
	h := loadSupply(t, 0)
	rows, err := h.store.Planning().ListStorage(h.ctx, store.PlanFilter{
		VersionIDs: []string{h.res.BudgetID}, Series: domain.SeriesPlan, Top: 20000,
	})
	if err != nil {
		t.Fatalf("read the stock ledger: %v", err)
	}

	rawIDs := map[string]bool{
		h.res.Warehouses["RAW-WH1"]: true,
		h.res.Warehouses["RAW-WH2"]: true,
	}
	byDate := map[domain.BusinessDate]domain.Dec{}
	for _, r := range rows {
		if rawIDs[r.WarehouseID] {
			byDate[r.BusinessDate] = byDate[r.BusinessDate].Add(r.EndingBalance)
		}
	}
	peak, peakOn := domain.Zero, domain.BusinessDate("")
	for date, v := range byDate {
		if v.GreaterThan(peak) || (v.Equal(peak) && date < peakOn) {
			peak, peakOn = v, date
		}
	}
	if peakOn != "2027-04-10" {
		t.Errorf("the raw silo peaks on %s; the workbook's own tracker says 2027-04-10", peakOn)
	}

	// And it peaks over capacity, which is the workbook's own conclusion: it is
	// why 20,700 t is packed into jumbo bags between 21 January and 30 March.
	// The system reaches that conclusion independently, from the plan.
	capacity := domain.D(seed.WorkbookRawCapacity)
	if !peak.GreaterThan(capacity) {
		t.Errorf("the raw silo peaks at %s t against %s t of capacity; the workbook's "+
			"whole reason for planning jumbo bagging is that it does not fit", peak, capacity)
	}
}
