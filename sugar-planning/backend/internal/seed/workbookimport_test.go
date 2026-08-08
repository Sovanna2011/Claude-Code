package seed_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/seed"
	"github.com/kss/sugarplan/internal/service"
)

// Importing the mill's own plan.
//
// `testdata/production-plan-2627-cane.csv` is a trimmed extract of the daily
// sheet of `ProductionPlan_2627_2.3mt Rev.1`: only the date and the two cane
// columns, with every row and column *position* preserved. That matters more
// than it sounds. The shipped `KSS-PLAN-CANE` mapping reads this file with no
// change at all - same first data row, same column indexes - so what is proved
// here is the mapping a site would actually use, not one written for a test.
//
// The workbook itself is not in the repository. It is the mill's own production
// plan, and the figures that matter are asserted either way.

func importFixture(t *testing.T) []byte {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", "production-plan-2627-cane.csv"))
	if err != nil {
		t.Fatalf("read the fixture: %v", err)
	}
	return content
}

func stageFixture(t *testing.T, h *supplyHarness, name string) service.Preview {
	t.Helper()
	imports := service.NewImports(h.store, h.planning,
		func() time.Time { return time.Date(2026, 12, 15, 6, 0, 0, 0, time.UTC) })
	ctx := auth.WithPrincipal(h.ctx, auth.NewPrincipal("t", "planner", "Planner", "",
		[]string{auth.RoleProductionPlanner},
		[]string{h.res.CompanyID}, []string{h.res.FactoryID}))

	preview, err := imports.Stage(ctx, service.StageRequest{
		MappingCode: "KSS-PLAN-CANE",
		VersionID:   h.res.BudgetID,
		FileName:    name,
		Content:     importFixture(t),
		Series:      domain.SeriesPlan,
	})
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	return preview
}

func TestTheShippedMappingReadsTheMillsOwnPlan(t *testing.T) {
	h := loadSupply(t, 0)
	preview := stageFixture(t, h, "production-plan-2627-cane.csv")

	// Nothing was left unmapped or missing. A file that lines up but reports a
	// column nobody claimed is the first sign the mapping has drifted from the
	// sheet.
	if len(preview.Unmapped) > 0 || len(preview.Missing) > 0 {
		t.Errorf("unmapped %v, missing %v", preview.Unmapped, preview.Missing)
	}

	byDate := map[domain.BusinessDate]string{}
	for _, row := range preview.Rows {
		if len(row.Errors) > 0 {
			continue
		}
		byDate[domain.BusinessDate(row.Values["businessDate"])] = row.Values["caneCrushed"]
	}

	// The same curve the generator is held to, arriving this time through the
	// import path instead of the seed. If these two ever disagree, one of them
	// is not reading the mill's plan.
	for _, c := range []struct {
		date domain.BusinessDate
		tons string
		why  string
	}{
		{"2026-12-01", "17000", "the opening day, while the boilers come up"},
		{"2026-12-03", "19000", "full rate"},
		{"2026-12-20", "9000", "half rate, the day before the first wash-out"},
		{"2026-12-21", "0", "the first wash-out"},
		{"2027-03-29", "0", "the last wash-out"},
		{"2027-04-03", "15000", "the run-down begins"},
		{"2027-04-16", "3000", "the last day of crushing"},
	} {
		if got := byDate[c.date]; got != c.tons {
			t.Errorf("%s: the file says %q, want %q — %s", c.date, got, c.tons, c.why)
		}
	}

	// The sheet carries a row for every day of the campaign out to September,
	// but cane only on the days the mill crushes. A wash-out day holds an
	// explicit 0 - the mill was there and crushed nothing - while 17 April
	// onward is blank, because there is no cane to have an opinion about. The
	// two are different facts and the file writes them differently.
	crushingDays, total := 0, domain.Zero
	for _, tons := range byDate {
		if tons == "" {
			continue
		}
		crushingDays++
		total = total.Add(domain.D(tons))
	}
	if crushingDays != seed.WorkbookCampaignDays {
		t.Errorf("%d days carry a cane figure, the campaign is %d days",
			crushingDays, seed.WorkbookCampaignDays)
	}
	if !total.Equal(domain.D(seed.WorkbookCaneTons)) {
		t.Errorf("the file totals %s t, the season target is %s", total, seed.WorkbookCaneTons)
	}
}

func TestTheSheetsOwnTotalsRowIsRefusedAndNamed(t *testing.T) {
	// The sheet ends with a `SUM` line holding 2,300,000. Importing it would
	// double the season in a single row, so it has to be refused - and refused
	// against the line number somebody is looking at in Excel, or they cannot
	// find it.
	h := loadSupply(t, 0)
	preview := stageFixture(t, h, "production-plan-2627-cane.csv")

	var rejected []domain.ImportRow
	for _, row := range preview.Rows {
		if len(row.Errors) > 0 {
			rejected = append(rejected, row)
		}
	}
	if len(rejected) != 1 {
		t.Fatalf("%d rows rejected, want just the SUM line: %v", len(rejected), rejected)
	}
	row := rejected[0]
	if row.Values["businessDate"] != "SUM" {
		t.Errorf("the rejected row is %v; the SUM line is the one that should fail", row.Values)
	}
	if row.RowNo != 325 {
		t.Errorf("the rejection is reported against row %d; the SUM line is row 325 of the sheet",
			row.RowNo)
	}
	if preview.Job.ValidRows != 304 || preview.Job.ErrorRows != 1 {
		t.Errorf("job counted %d valid and %d in error, want 304 and 1",
			preview.Job.ValidRows, preview.Job.ErrorRows)
	}

	// And the whole file is still held, not half-written: staging never touches
	// the plan.
	if !preview.Job.IsOpen() {
		t.Errorf("the job is %s; staging leaves it open for somebody to look at",
			preview.Job.Status)
	}
}
