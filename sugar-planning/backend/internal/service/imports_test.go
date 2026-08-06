package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
)

// These cover the controlled import: what a file has to survive before it is
// allowed anywhere near the plan, and what happens when somebody commits it.

func importHarness(t *testing.T) (*harness, *service.Imports) {
	t.Helper()
	h := newHarness(t, 0)
	return h, service.NewImports(h.store, h.planning, fixedClock())
}

// caneSheet is a file in the shape the seeded CANE-DAILY mapping reads.
const caneSheet = "Date,Delivered (MT),Accepted (MT),Crushed (MT),Rate (TPH),Remarks\n" +
	"01/12/2026,\"17,000.000\",\"16,900.000\",\"16,788.321\",700,first day\n" +
	"02/12/2026,17000,16900,16788.321,700,\n"

func stage(t *testing.T, ctx context.Context, imp *service.Imports, h *harness,
	body string, series domain.Series,
) service.Preview {

	t.Helper()
	preview, err := imp.Stage(ctx, service.StageRequest{
		MappingCode: "CANE-DAILY", VersionID: h.seeded.BudgetID,
		FileName: "cane.csv", Content: []byte(body), Series: series,
	})
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	return preview
}

func TestAStagedFileWritesNothingUntilItIsCommitted(t *testing.T) {
	h, imp := importHarness(t)
	ctx := h.as(auth.RoleProductionPlanner)

	preview := stage(t, ctx, imp, h, caneSheet, domain.SeriesPlan)
	if preview.Job.Status != domain.ImportValidated {
		t.Fatalf("status = %s, want VALIDATED", preview.Job.Status)
	}
	if preview.Job.TotalRows != 2 || preview.Job.ValidRows != 2 || preview.Job.ErrorRows != 0 {
		t.Fatalf("counts = %+v", preview.Job)
	}
	// The figures come back parsed: the thousands separator is gone and the
	// day-first date is ISO.
	if preview.Rows[0].Values["businessDate"] != "2026-12-01" {
		t.Errorf("date = %q", preview.Rows[0].Values["businessDate"])
	}
	if preview.Rows[0].Values["caneCrushed"] != "16788.321" {
		t.Errorf("crushed = %q", preview.Rows[0].Values["caneCrushed"])
	}

	// Nothing has been written. This is the whole point of staging.
	rows, err := h.store.Planning().ListCane(context.Background(), store.PlanFilter{
		VersionIDs: []string{h.seeded.BudgetID}, From: "2026-12-01", To: "2026-12-01",
		Series: domain.SeriesPlan,
	})
	if err != nil {
		t.Fatalf("list cane: %v", err)
	}
	if len(rows) != 1 || rows[0].CaneCrushed.Equal(domain.D("16788.321")) &&
		rows[0].CaneDelivered.Equal(domain.D("17000")) {
		// The generated plan already has a row for that date; what must not have
		// happened is the file's delivered figure appearing on it.
		t.Errorf("staging must not have written the file's figures: %+v", rows)
	}

	result, err := imp.Commit(ctx, preview.Job.ID, service.CommitRequest{})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if result.Written != 2 {
		t.Errorf("written = %d, want 2", result.Written)
	}
	// The plan already had rows for those dates, so the import replaced them.
	if result.Replaced != 2 {
		t.Errorf("replaced = %d, want 2; the generated plan already covers those days",
			result.Replaced)
	}
	if result.Job.Status != domain.ImportCommitted || result.Job.CommittedAt == nil {
		t.Errorf("the job must be committed: %+v", result.Job)
	}

	rows, err = h.store.Planning().ListCane(context.Background(), store.PlanFilter{
		VersionIDs: []string{h.seeded.BudgetID}, From: "2026-12-01", To: "2026-12-01",
		Series: domain.SeriesPlan,
	})
	if err != nil {
		t.Fatalf("list cane: %v", err)
	}
	if len(rows) != 1 || !rows[0].CaneDelivered.Equal(domain.D("17000")) {
		t.Fatalf("the committed figures must be in the plan: %+v", rows)
	}

	// A committed import cannot be committed again.
	if _, err := imp.Commit(ctx, preview.Job.ID, service.CommitRequest{}); !errors.Is(err, domain.ErrStateTransition) {
		t.Errorf("committing twice must be refused, got %v", err)
	}
}

func TestABadRowStopsTheWholeFileUnlessPartialIsAskedFor(t *testing.T) {
	h, imp := importHarness(t)
	ctx := h.as(auth.RoleProductionPlanner)

	sheet := caneSheet + "03/12/2026,17000,16900,not a number,700,\n"
	preview := stage(t, ctx, imp, h, sheet, domain.SeriesPlan)
	if preview.Job.ErrorRows != 1 || preview.Job.ValidRows != 2 {
		t.Fatalf("counts = %+v", preview.Job)
	}
	bad := preview.Rows[2]
	if bad.OK() || bad.RowNo != 4 {
		t.Fatalf("the bad row is line 4 of the file: %+v", bad)
	}
	if !strings.Contains(bad.Errors[0].Message, "not a number") {
		t.Errorf("the message must quote the cell: %q", bad.Errors[0].Message)
	}

	// All or nothing is the default: section 22 says never post a partially
	// valid document unless the caller asks for it.
	if _, err := imp.Commit(ctx, preview.Job.ID, service.CommitRequest{}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("a file with a bad row must be refused whole, got %v", err)
	}

	result, err := imp.Commit(ctx, preview.Job.ID, service.CommitRequest{Partial: true})
	if err != nil {
		t.Fatalf("partial commit: %v", err)
	}
	if result.Written != 2 || result.Skipped != 1 {
		t.Errorf("written %d skipped %d, want 2 and 1", result.Written, result.Skipped)
	}
}

func TestACodeThatNamesNothingFailsItsRow(t *testing.T) {
	h, imp := importHarness(t)
	ctx := h.as(auth.RoleProductionPlanner)

	sheet := "Date,Product,Pack,Good output (MT)\n" +
		"01/12/2026,REF,P50KG,500\n" +
		"01/12/2026,NOSUCH,P50KG,500\n"
	preview, err := imp.Stage(ctx, service.StageRequest{
		MappingCode: "PROD-DAILY", VersionID: h.seeded.BudgetID,
		FileName: "production.csv", Content: []byte(sheet), Series: domain.SeriesPlan,
	})
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	if preview.Job.ValidRows != 1 || preview.Job.ErrorRows != 1 {
		t.Fatalf("counts = %+v", preview.Job)
	}
	// The sound row resolved its codes to ids.
	if preview.Rows[0].Values["productId"] != h.seeded.Products["REF"] {
		t.Errorf("the product code must resolve to its id, got %q",
			preview.Rows[0].Values["productId"])
	}
	// The bad one says which code it was, which is what somebody needs to fix it.
	if msg := preview.Rows[1].Errors[0].Message; !strings.Contains(msg, "NOSUCH") {
		t.Errorf("the message must name the code, got %q", msg)
	}
}

func TestTheColumnsThatDidNotLineUpAreReported(t *testing.T) {
	h, imp := importHarness(t)
	ctx := h.as(auth.RoleProductionPlanner)

	// "Cane crushed" is not the heading the mapping looks for, and "Brix" is a
	// column nobody mapped. Both are worth saying: this is what an import that
	// produced nothing looks like from the inside.
	sheet := "Date,Cane crushed,Brix\n01/12/2026,16788.321,18.2\n"
	preview := stage(t, ctx, imp, h, sheet, domain.SeriesPlan)

	if !contains(preview.Missing, "Crushed (MT)") {
		t.Errorf("the mapped heading the file lacks must be reported, got %v", preview.Missing)
	}
	if !contains(preview.Unmapped, "Brix") || !contains(preview.Unmapped, "Cane crushed") {
		t.Errorf("the file's unmapped columns must be reported, got %v", preview.Unmapped)
	}
	// The job carries them too, so they survive to the screen that reads it back.
	if len(preview.Job.Errors) == 0 {
		t.Error("the job must record what did not line up")
	}
}

func TestAnImportCannotPostWhatThePersonCouldNot(t *testing.T) {
	h, imp := importHarness(t)

	// A warehouse keeper may post stock, not plan rows.
	if _, err := imp.Stage(h.keeper(), service.StageRequest{
		MappingCode: "CANE-DAILY", VersionID: h.seeded.BudgetID,
		FileName: "cane.csv", Content: []byte(caneSheet), Series: domain.SeriesPlan,
	}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("a keeper must not import plan rows, got %v", err)
	}

	// And a planner may not post actuals, whatever they upload.
	if _, err := imp.Stage(h.as(auth.RoleProductionPlanner), service.StageRequest{
		MappingCode: "CANE-DAILY", VersionID: h.seeded.ActualID,
		FileName: "cane.csv", Content: []byte(caneSheet), Series: domain.SeriesActual,
	}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("a planner must not import actual cane, got %v", err)
	}

	// The supervisor who may post actual cane can.
	preview, err := imp.Stage(h.as(auth.RoleShiftSupervisor), service.StageRequest{
		MappingCode: "CANE-DAILY", VersionID: h.seeded.ActualID,
		FileName: "cane.csv", Content: []byte(caneSheet), Series: domain.SeriesActual,
	})
	if err != nil {
		t.Fatalf("a supervisor importing actual cane: %v", err)
	}
	if preview.Job.ValidRows != 2 {
		t.Errorf("counts = %+v", preview.Job)
	}
}

func TestACancelledImportWritesNothingAndStays(t *testing.T) {
	h, imp := importHarness(t)
	ctx := h.as(auth.RoleProductionPlanner)

	preview := stage(t, ctx, imp, h, caneSheet, domain.SeriesPlan)
	cancelled, err := imp.Cancel(ctx, preview.Job.ID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if cancelled.Status != domain.ImportCancelled {
		t.Errorf("status = %s", cancelled.Status)
	}
	if _, err := imp.Commit(ctx, preview.Job.ID, service.CommitRequest{}); !errors.Is(err, domain.ErrStateTransition) {
		t.Errorf("a cancelled import must not commit, got %v", err)
	}

	// What was uploaded and refused is part of the record.
	back, err := imp.GetJob(ctx, preview.Job.ID, false, 0, 100)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if len(back.Rows) != 2 {
		t.Errorf("the staged rows must survive a cancellation, got %d", len(back.Rows))
	}
}

func TestAMappingIsMasterDataAndIsCheckedWhenItIsSaved(t *testing.T) {
	h, imp := importHarness(t)

	m := domain.ImportMapping{
		Code: "my-sheet", Name: "Mine", Kind: domain.ImportCane, HeaderRow: 1,
		Columns:  []domain.ColumnMapping{{Field: "businessDate", Header: "Date"}},
		Validity: domain.Validity{Active: true},
	}

	// A planner reads mappings; changing what a column means is master data.
	if _, err := imp.SaveMapping(h.as(auth.RoleProductionPlanner), m); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("a planner must not change a mapping, got %v", err)
	}

	admin := h.as(auth.RoleMasterDataAdmin)
	saved, err := imp.SaveMapping(admin, m)
	if err != nil {
		t.Fatalf("save mapping: %v", err)
	}
	if saved.Code != "MY-SHEET" {
		t.Errorf("the code is normalised, got %q", saved.Code)
	}

	broken := m
	broken.Code = "BROKEN"
	broken.Columns = []domain.ColumnMapping{{Field: "caneCrushed", Header: "Crushed"}}
	if _, err := imp.SaveMapping(admin, broken); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("a mapping with no date column must be refused when it is saved, got %v", err)
	}

	list, err := imp.ListMappings(h.as(auth.RoleProductionPlanner), string(domain.ImportCane))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) < 2 {
		t.Errorf("the seeded template and the new one, got %d", len(list))
	}
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
