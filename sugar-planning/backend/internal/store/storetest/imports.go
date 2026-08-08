package storetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// The staging area is where an import lives between being read and being
// committed, so both stores have to agree about what survives the round trip:
// the column mapping, the parsed values, the errors attached to each row.

func testImports(t *testing.T, newStore Factory) {
	ctx := context.Background()
	s := newStore(t)
	imp := s.Imports()

	column := 3
	mapping, err := imp.SaveMapping(ctx, domain.ImportMapping{
		Code: "CANE-DAILY", Name: "Daily cane sheet", Kind: domain.ImportCane,
		HeaderRow: 2, FirstDataRow: 4, Sheet: "Daily plan", Delimiter: ";", DateFormat: "02/01/2006",
		DecimalComma: true, Note: "from the weighbridge office",
		Columns: []domain.ColumnMapping{
			{Field: "businessDate", Header: "Date"},
			{Field: "caneCrushed", Column: &column},
			{Field: "availableHours", Header: "Hours", Default: "24"},
		},
		Validity: domain.Validity{Active: true},
	}, "admin")
	must(t, err, "save mapping")

	back, err := imp.GetMapping(ctx, mapping.ID)
	must(t, err, "get mapping")
	if len(back.Columns) != 3 {
		t.Fatalf("the column mapping must survive the round trip, got %+v", back.Columns)
	}
	// Every part of the template matters: a lost delimiter or a lost decimal
	// convention silently reads the next file wrong.
	if back.Delimiter != ";" || !back.DecimalComma || back.DateFormat != "02/01/2006" {
		t.Errorf("the file conventions must survive: %+v", back)
	}
	// The worksheet name has to survive the round trip. A mapping that came
	// back without it would read the first sheet of the workbook and report the
	// summary tab as the plan, which is exactly what happened before the column
	// existed - and it did not look like a failure.
	if back.Sheet != "Daily plan" {
		t.Errorf("the worksheet name came back as %q, want \"Daily plan\"", back.Sheet)
	}
	if back.HeaderRow != 2 || back.FirstDataRow != 4 {
		t.Errorf("the row numbers must survive: header %d, data %d",
			back.HeaderRow, back.FirstDataRow)
	}
	if back.Columns[1].Column == nil || *back.Columns[1].Column != 3 {
		t.Errorf("a positional column must survive: %+v", back.Columns[1])
	}
	if back.Columns[2].Default != "24" {
		t.Errorf("a default must survive: %+v", back.Columns[2])
	}

	byCode, err := imp.MappingByCode(ctx, "CANE-DAILY")
	must(t, err, "mapping by code")
	if byCode.ID != mapping.ID {
		t.Errorf("the code must find the same mapping")
	}
	if _, err := imp.MappingByCode(ctx, "NOSUCH"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("an unknown code must be not found, got %v", err)
	}

	// The code is the business key.
	if _, err := imp.SaveMapping(ctx, domain.ImportMapping{
		Code: "CANE-DAILY", Kind: domain.ImportCane,
	}, "admin"); !errors.Is(err, domain.ErrDuplicate) {
		t.Errorf("a duplicate mapping code must be refused, got %v", err)
	}

	list, err := imp.ListMappings(ctx, string(domain.ImportCane))
	must(t, err, "list mappings")
	if len(list) != 1 {
		t.Errorf("one cane mapping, got %d", len(list))
	}
	list, err = imp.ListMappings(ctx, string(domain.ImportShipment))
	must(t, err, "list mappings of another kind")
	if len(list) != 0 {
		t.Errorf("no shipment mapping, got %d", len(list))
	}

	// --- the job and its staged rows ---------------------------------------

	job, err := imp.SaveJob(ctx, domain.ImportJob{
		Kind: domain.ImportCane, MappingID: mapping.ID, FileName: "cane.csv",
		Status: domain.ImportValidated, TotalRows: 3, ValidRows: 2, ErrorRows: 1,
		Errors: []domain.FieldError{{Field: "columns", Code: "UNMAPPED_COLUMN",
			Message: `the file's "Brix" column is not in the mapping`}},
		Note: string(domain.SeriesPlan),
	}, "planner")
	must(t, err, "save job")
	if job.ID == "" || job.CreatedBy != "planner" {
		t.Fatalf("the job must be stored with its actor: %+v", job)
	}

	rows := []domain.ImportRow{
		{RowNo: 4, Values: map[string]string{"businessDate": "2026-12-01", "caneCrushed": "16788.321"}},
		{RowNo: 5, Values: map[string]string{"businessDate": "2026-12-02", "caneCrushed": "16788.321"},
			Replaces: true},
		{RowNo: 6, Values: map[string]string{"businessDate": "2026-12-01"},
			Duplicate: true,
			Errors: []domain.FieldError{{Field: "businessDate", Code: "DUPLICATE",
				Message: "this row repeats row 4 of the file"}}},
	}
	must(t, imp.SaveRows(ctx, job.ID, rows), "save rows")

	page, err := imp.Rows(ctx, job.ID, false, 0, 100)
	must(t, err, "read rows")
	if len(page.Items) != 3 || page.Count != 3 {
		t.Fatalf("three staged rows, got %d of %d", len(page.Items), page.Count)
	}
	if page.Items[0].RowNo != 4 || page.Items[0].Values["caneCrushed"] != "16788.321" {
		t.Errorf("the parsed values must survive: %+v", page.Items[0])
	}
	if !page.Items[1].Replaces {
		t.Error("a row that would replace an existing one must stay marked")
	}

	// The error download reads only the rows that failed.
	page, err = imp.Rows(ctx, job.ID, true, 0, 100)
	must(t, err, "read error rows")
	if len(page.Items) != 1 || page.Items[0].RowNo != 6 {
		t.Fatalf("one failed row, got %v", page.Items)
	}
	if !page.Items[0].Duplicate || len(page.Items[0].Errors) != 1 {
		t.Errorf("the row's problems must survive: %+v", page.Items[0])
	}
	if page.Items[0].Errors[0].Field != "businessDate" {
		t.Errorf("the error must keep its field: %+v", page.Items[0].Errors[0])
	}

	// Re-staging replaces the rows rather than adding to them.
	must(t, imp.SaveRows(ctx, job.ID, rows[:1]), "re-save rows")
	page, err = imp.Rows(ctx, job.ID, false, 0, 100)
	must(t, err, "read rows again")
	if len(page.Items) != 1 {
		t.Errorf("re-staging must replace, got %d rows", len(page.Items))
	}

	job.Status = domain.ImportCommitted
	job.CommittedBy = "planner"
	committed, err := imp.SaveJob(ctx, job, "planner")
	must(t, err, "commit the job")
	if committed.Status != domain.ImportCommitted || committed.CommittedBy != "planner" {
		t.Errorf("the commit must be recorded: %+v", committed)
	}
	// The record of who first uploaded it is not overwritten by whoever
	// committed it.
	if committed.CreatedBy != "planner" || committed.CreatedAt != job.CreatedAt {
		t.Errorf("the upload's own record must survive the commit: %+v", committed)
	}

	jobs, err := imp.ListJobs(ctx, store.ImportFilter{Status: domain.ImportCommitted})
	must(t, err, "list jobs")
	if len(jobs.Items) != 1 {
		t.Errorf("one committed job, got %d", len(jobs.Items))
	}
	jobs, err = imp.ListJobs(ctx, store.ImportFilter{Status: domain.ImportCancelled})
	must(t, err, "list cancelled jobs")
	if len(jobs.Items) != 0 {
		t.Errorf("no cancelled job, got %d", len(jobs.Items))
	}

	if _, err := imp.GetJob(ctx, "not-a-uuid"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("a malformed job id must be not found, got %v", err)
	}
}
