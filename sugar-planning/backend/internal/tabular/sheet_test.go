package tabular_test

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/kss/sugarplan/internal/tabular"
)

// Reading a named worksheet.
//
// This exists because of what a real workbook turned out to look like. The
// importer read `xl/worksheets/sheet1.xml` and nothing else, which is right for
// a file exported for the purpose and wrong for the file a mill actually keeps
// its plan in: that one opens on a summary tab and holds three hundred days of
// figures on the second. Asked to import it, the system read the summary,
// interpreted its labels as dates and tonnages, and reported twenty-three rows
// without a hint that it had read the wrong tab.

// workbook builds a minimal .xlsx with the given sheets, in order.
//
// Deliberately writes the worksheet parts in an order that does not match the
// tab order, and numbers them so that the second tab is sheet1.xml. A reader
// that resolves a name by counting, or by trusting the file number, passes on a
// tidy workbook and fails on a real one.
func workbook(t *testing.T, sheets []struct{ name, rows string }) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	add := func(name, body string) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	var book, rels strings.Builder
	book.WriteString(`<workbook xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`)
	rels.WriteString(`<Relationships>`)
	for i, s := range sheets {
		// Tab i is stored as sheet(N-i).xml: reversed, so file order and tab
		// order disagree.
		file := len(sheets) - i
		book.WriteString(`<sheet name="` + s.name + `" sheetId="` +
			string(rune('1'+i)) + `" r:id="rId` + string(rune('1'+i)) + `"/>`)
		rels.WriteString(`<Relationship Id="rId` + string(rune('1'+i)) +
			`" Target="worksheets/sheet` + string(rune('0'+file)) + `.xml"/>`)
	}
	book.WriteString(`</sheets></workbook>`)
	rels.WriteString(`</Relationships>`)

	add("xl/workbook.xml", book.String())
	add("xl/_rels/workbook.xml.rels", rels.String())
	for i, s := range sheets {
		file := len(sheets) - i
		add("xl/worksheets/sheet"+string(rune('0'+file))+".xml",
			`<worksheet><sheetData>`+s.rows+`</sheetData></worksheet>`)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return buf.Bytes()
}

func row(cells ...string) string {
	var b strings.Builder
	b.WriteString("<row>")
	for _, c := range cells {
		b.WriteString(`<c t="inlineStr"><is><t>` + c + `</t></is></c>`)
	}
	b.WriteString("</row>")
	return b.String()
}

func twoTabs(t *testing.T) []byte {
	return workbook(t, []struct{ name, rows string }{
		{"Summary Report", row("Total cane crushed", "2300000")},
		{"RW'2627(2.3mt)Rev1(re)", row("2026-12-01", "17000") + row("2026-12-02", "17000")},
	})
}

func TestAnEmptySheetNameTakesTheFirstTab(t *testing.T) {
	rows, err := tabular.ReadSheet("plan.xlsx", twoTabs(t), 0, "")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(rows) != 1 || rows[0][0] != "Total cane crushed" {
		t.Errorf("the first tab is the summary, got %v", rows)
	}
}

func TestANamedSheetIsFoundWhereverItsFileSits(t *testing.T) {
	// The daily plan is the second tab and is stored as sheet1.xml. A reader
	// that trusted the file number would return the summary and say nothing.
	rows, err := tabular.ReadSheet("plan.xlsx", twoTabs(t), 0, "RW'2627(2.3mt)Rev1(re)")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("%d rows, want the two days of plan: %v", len(rows), rows)
	}
	if rows[0][0] != "2026-12-01" || rows[0][1] != "17000" {
		t.Errorf("first row = %v, want the 1 December opening day at 17,000 t", rows[0])
	}
}

func TestASheetNameThatIsNotThereSaysWhichOnesAre(t *testing.T) {
	// The failure a site hits when somebody renames a tab. Naming the sheets
	// that do exist turns a support call into a correction.
	_, err := tabular.ReadSheet("plan.xlsx", twoTabs(t), 0, "Daily Plan")
	if err == nil {
		t.Fatal("a missing sheet must be refused, not silently replaced by the first one")
	}
	for _, want := range []string{"Daily Plan", "Summary Report", "RW'2627"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not mention %q: %v", want, err)
		}
	}
}
