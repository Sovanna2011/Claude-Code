package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/kss/sugarplan/internal/domain"
)

// A spreadsheet writes numbers in whatever way the person who made it had their
// machine set to, and getting one of these wrong turns a thousand tons into one.

func TestSpreadsheetNumbersAreReadTheWayTheyAreWritten(t *testing.T) {
	cases := []struct {
		in    string
		comma bool
		want  string
	}{
		{"16788.321", false, "16788.321"},
		{"16,788.321", false, "16788.321"},
		{"16 788.321", false, "16788.321"}, // Excel's non-breaking thousands space
		{"16.788,321", true, "16788.321"},  // European convention
		{"16788,321", true, "16788.321"},   // European, no thousands separator
		{"$1,205.00", false, "1205"},       // a currency column
		{"(1,205)", false, "-1205"},        // accounting notation for a negative
		{"  2300000  ", false, "2300000"},  // padded by the export
		{"11.00%", false, "11"},            // a percentage column
		{"0", false, "0"},
		{"", false, "0"},
	}
	for _, c := range cases {
		got, err := domain.ParseImportDecimal(c.in, c.comma)
		if err != nil {
			t.Errorf("%q: %v", c.in, err)
			continue
		}
		if got.String() != c.want {
			t.Errorf("%q (decimalComma=%v) = %s, want %s", c.in, c.comma, got, c.want)
		}
	}

	for _, bad := range []string{"n/a", "-", "sixteen", "1.2.3.4x"} {
		if _, err := domain.ParseImportDecimal(bad, false); err == nil {
			t.Errorf("%q must be refused rather than read as a number", bad)
		}
	}
}

func TestDatesAreReadInTheMappingsFormat(t *testing.T) {
	// With a format given there is no ambiguity: 01/12/2026 is the first of
	// December, not the twelfth of January.
	d, err := domain.ParseImportDate("01/12/2026", "02/01/2006")
	if err != nil || d != "2026-12-01" {
		t.Errorf("day-first = %q, %v; want 2026-12-01", d, err)
	}
	d, err = domain.ParseImportDate("01/12/2026", "01/02/2006")
	if err != nil || d != "2026-01-12" {
		t.Errorf("month-first = %q, %v; want 2026-01-12", d, err)
	}

	// A column formatted as a number gives Excel's day serial rather than a date.
	// 46357 is 1 December 2026.
	d, err = domain.ParseImportDate("46357", "")
	if err != nil || d != "2026-12-01" {
		t.Errorf("serial = %q, %v; want 2026-12-01", d, err)
	}

	if _, err := domain.ParseImportDate("not a date", ""); err == nil {
		t.Error("a cell that is not a date must be refused")
	}
}

// caneMapping is the shape a site's daily sheet usually has.
func caneMapping() domain.ImportMapping {
	return domain.ImportMapping{
		Code: "CANE-DAILY", Name: "Daily cane sheet", Kind: domain.ImportCane,
		HeaderRow: 1, DateFormat: "02/01/2006",
		Columns: []domain.ColumnMapping{
			{Field: "businessDate", Header: "Date"},
			{Field: "caneCrushed", Header: "Cane crushed (MT)"},
			{Field: "availableHours", Header: "Hours", Default: "24"},
			{Field: "note", Header: "Remarks"},
		},
	}
}

func TestAMappingReadsTheColumnsItNamesAndReportsTheRest(t *testing.T) {
	m := caneMapping()
	cells := [][]string{
		{"Date", "Cane crushed (MT)", "Remarks", "Brix"},
		{"01/12/2026", "16,788.321", "first day", "18.2"},
		{"02/12/2026", "16788.321", "", "18.4"},
		{"", "", "", ""},
	}

	result, err := domain.ReadRows(m, cells)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("two data rows, got %d", len(result.Rows))
	}
	// The row number is the line in the file, so an error names what somebody
	// sees in Excel rather than an offset into an array.
	if result.Rows[0].RowNo != 2 {
		t.Errorf("first data row is line 2, got %d", result.Rows[0].RowNo)
	}
	if result.Rows[0].Values["businessDate"] != "2026-12-01" {
		t.Errorf("date = %q", result.Rows[0].Values["businessDate"])
	}
	if result.Rows[0].Values["caneCrushed"] != "16788.321" {
		t.Errorf("crushed = %q", result.Rows[0].Values["caneCrushed"])
	}
	// A default fills a column the file does not have at all.
	if result.Rows[0].Values["availableHours"] != "24" {
		t.Errorf("the default must fill an absent column, got %q",
			result.Rows[0].Values["availableHours"])
	}
	// A column nobody mapped is reported rather than ignored: a figure somebody
	// expected to import and did not is the failure that goes unnoticed.
	if len(result.Unmapped) != 1 || result.Unmapped[0] != "Brix" {
		t.Errorf("unmapped = %v, want [Brix]", result.Unmapped)
	}
	if len(result.Missing) != 1 || result.Missing[0] != "Hours" {
		t.Errorf("missing = %v, want [Hours]", result.Missing)
	}
}

func TestABadCellFailsItsOwnRowAndNoOther(t *testing.T) {
	m := caneMapping()
	cells := [][]string{
		{"Date", "Cane crushed (MT)", "Remarks"},
		{"01/12/2026", "16788.321", "fine"},
		{"02/12/2026", "n/a", "the mill was down"},
		{"not a date", "100", ""},
		{"", "500", "no date at all"},
	}

	result, err := domain.ReadRows(m, cells)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(result.Rows) != 4 {
		t.Fatalf("four data rows, got %d", len(result.Rows))
	}
	if !result.Rows[0].OK() {
		t.Errorf("the sound row must be sound: %+v", result.Rows[0].Errors)
	}
	for i, want := range map[int]string{1: "caneCrushed", 2: "businessDate", 3: "businessDate"} {
		row := result.Rows[i]
		if row.OK() {
			t.Errorf("row %d must have failed", row.RowNo)
			continue
		}
		if row.Errors[0].Field != want {
			t.Errorf("row %d failed on %q, want %q", row.RowNo, row.Errors[0].Field, want)
		}
	}
	// The message has to be readable by the person who has the file open.
	if msg := result.Rows[1].Errors[0].Message; !strings.Contains(msg, "n/a") {
		t.Errorf("the message must quote the cell, got %q", msg)
	}
}

func TestTheSameRowTwiceInOneFileIsCaught(t *testing.T) {
	m := caneMapping()
	cells := [][]string{
		{"Date", "Cane crushed (MT)"},
		{"01/12/2026", "16788.321"},
		{"02/12/2026", "16788.321"},
		{"01/12/2026", "9999.000"},
	}

	result, err := domain.ReadRows(m, cells)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	last := result.Rows[2]
	if !last.Duplicate || last.OK() {
		t.Fatalf("the repeated date must be caught: %+v", last)
	}
	// Naming the earlier row is what makes it fixable.
	if !strings.Contains(last.Errors[0].Message, "row 2") {
		t.Errorf("the message must name the row it repeats, got %q", last.Errors[0].Message)
	}
	if result.Rows[0].Duplicate {
		t.Error("the first occurrence is not the duplicate")
	}
}

func TestAFileWithNoDataIsRefusedRatherThanCommittedEmpty(t *testing.T) {
	m := caneMapping()
	if _, err := domain.ReadRows(m, [][]string{{"Date", "Cane crushed (MT)"}}); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("a file of headings only must be refused, got %v", err)
	}
	if _, err := domain.ReadRows(m, nil); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("an empty file must be refused, got %v", err)
	}
	// The mapping says the headings are on row 3; the file has two rows.
	tooFar := caneMapping()
	tooFar.HeaderRow = 3
	if _, err := domain.ReadRows(tooFar, [][]string{{"a"}, {"b"}}); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("a heading row past the end of the file must be refused, got %v", err)
	}
}

func TestAMappingThatCouldNotWorkIsRefusedWhenItIsSaved(t *testing.T) {
	base := caneMapping()

	cases := map[string]func(*domain.ImportMapping){
		"no code":            func(m *domain.ImportMapping) { m.Code = "" },
		"unknown kind":       func(m *domain.ImportMapping) { m.Kind = "DAILY_WEATHER" },
		"no columns":         func(m *domain.ImportMapping) { m.Columns = nil },
		"unknown field":      func(m *domain.ImportMapping) { m.Columns[1].Field = "rainfall" },
		"column named twice": func(m *domain.ImportMapping) { m.Columns[1].Field = "businessDate" },
		"nothing to fill from": func(m *domain.ImportMapping) {
			m.Columns[1].Header, m.Columns[1].Default = "", ""
		},
		"data above the headings": func(m *domain.ImportMapping) { m.HeaderRow, m.FirstDataRow = 3, 2 },
		"unreadable date format":  func(m *domain.ImportMapping) { m.DateFormat = "dd/mm/yyyy" },
	}
	for name, break_ := range cases {
		m := base
		m.Columns = append([]domain.ColumnMapping(nil), base.Columns...)
		break_(&m)
		if err := m.Validate(); err == nil {
			t.Errorf("%s: must be refused when the mapping is saved, not when a file is uploaded", name)
		}
	}

	// A mapping missing a required field could never produce a row.
	m := base
	m.Columns = []domain.ColumnMapping{{Field: "caneCrushed", Header: "Cane"}}
	err := m.Validate()
	if err == nil {
		t.Fatal("a mapping with no date column must be refused")
	}
	if !strings.Contains(err.Error(), "Date") {
		t.Errorf("the message must name the field that is missing, got %q", err)
	}

	if err := base.Validate(); err != nil {
		t.Errorf("the ordinary mapping must be accepted: %v", err)
	}
}

func TestColumnsCanBeMatchedByPositionWhenTheSheetHasNoHeadings(t *testing.T) {
	first, second := 0, 1
	m := domain.ImportMapping{
		Code: "POSITIONAL", Kind: domain.ImportCane, DateFormat: "2006-01-02",
		Columns: []domain.ColumnMapping{
			{Field: "businessDate", Column: &first},
			{Field: "caneCrushed", Column: &second},
		},
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("a positional mapping is legitimate: %v", err)
	}

	result, err := domain.ReadRows(m, [][]string{{"2026-12-01", "16788.321"}})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(result.Rows) != 1 || result.Rows[0].Values["caneCrushed"] != "16788.321" {
		t.Errorf("positional read = %+v", result.Rows)
	}
}
