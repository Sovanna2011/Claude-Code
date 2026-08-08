package tabular_test

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/kss/sugarplan/internal/tabular"
)

func TestCSVKeepsEveryCell(t *testing.T) {
	// Excel writes a byte order mark, quotes anything containing its delimiter,
	// and is perfectly happy to leave a row short.
	data := []byte("\xEF\xBB\xBFDate,Cane crushed,Note\n" +
		"2026-12-01,\"16,788.321\",\"first day, wet\"\n" +
		"2026-12-02,16788.321\n")

	rows, err := tabular.ReadCSV(data, 0)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("three rows, got %d", len(rows))
	}
	if rows[0][0] != "Date" {
		t.Errorf("the byte order mark must not become part of the first header, got %q", rows[0][0])
	}
	if rows[1][1] != "16,788.321" || rows[1][2] != "first day, wet" {
		t.Errorf("a quoted field containing the delimiter must survive: %q", rows[1])
	}
	// A short row is reported as short rather than padded, so the mapping can
	// say which column was missing.
	if len(rows[2]) != 2 {
		t.Errorf("a short row must stay short, got %d cells", len(rows[2]))
	}
}

func TestATabSeparatedFileIsReadByItsExtension(t *testing.T) {
	rows, err := tabular.Read("plan.tsv", []byte("Date\tTons\n2026-12-01\t100\n"), 0)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(rows) != 2 || rows[1][1] != "100" {
		t.Errorf("a .tsv must be split on tabs, got %v", rows)
	}
}

func TestAFileThatIsNotTextIsRefusedRatherThanMangled(t *testing.T) {
	if _, err := tabular.ReadCSV([]byte{0xff, 0xfe, 0x00, 0x41}, 0); err == nil {
		t.Error("a file that is not UTF-8 must be refused, not read as rubbish")
	}
	if _, err := tabular.Read("plan.pdf", []byte("x"), 0); err == nil {
		t.Error("a format this system cannot read must be named, not guessed at")
	}
}

// buildXLSX writes the minimum workbook a reader has to cope with: shared
// strings, a numeric cell, an inline string, and a gap where a blank cell is.
func buildXLSX(t *testing.T, sheet, shared string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	write := func(name, body string) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip %s: %v", name, err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if sheet != "" {
		write("xl/worksheets/sheet1.xml", sheet)
	} else {
		// A workbook whose data is on a later tab has no sheet1 this reader
		// recognises, which is the case worth refusing clearly.
		write("xl/worksheets/sheet2.xml", `<worksheet/>`)
	}
	if shared != "" {
		write("xl/sharedStrings.xml", shared)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

func TestXLSXPlacesCellsByReferenceNotByOrder(t *testing.T) {
	shared := `<?xml version="1.0"?><sst><si><t>Date</t></si><si><t>Cane</t></si>` +
		`<si><t>Note</t></si><si><r><t>wet </t></r><r><t>morning</t></r></si></sst>`
	// Row 2 has no cell B: a blank cell is simply absent from the XML, and its
	// column must not shift left.
	sheet := `<?xml version="1.0"?><worksheet><sheetData>` +
		`<row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c><c r="C1" t="s"><v>2</v></c></row>` +
		`<row r="2"><c r="A2" t="inlineStr"><is><t>2026-12-01</t></is></c>` +
		`<c r="C2" t="s"><v>3</v></c></row>` +
		`<row r="3"><c r="A3" t="inlineStr"><is><t>2026-12-02</t></is></c>` +
		`<c r="B3"><v>16788.321</v></c></row>` +
		`</sheetData></worksheet>`

	rows, err := tabular.ReadXLSX(buildXLSX(t, sheet, shared))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("three rows, got %d", len(rows))
	}
	if strings.Join(rows[0], "|") != "Date|Cane|Note" {
		t.Errorf("headers = %v", rows[0])
	}
	if len(rows[1]) != 3 || rows[1][1] != "" || rows[1][2] != "wet morning" {
		t.Errorf("a blank cell must hold its place and a split string must be joined: %q", rows[1])
	}
	if rows[2][1] != "16788.321" {
		t.Errorf("a numeric cell must come through unrounded, got %q", rows[2][1])
	}
}

func TestAWorkbookWithNoFirstSheetIsRefusedClearly(t *testing.T) {
	_, err := tabular.ReadXLSX(buildXLSX(t, "", ""))
	if err == nil {
		t.Fatal("a workbook with no readable sheet must be refused")
	}
	// The message has to tell somebody what to do about it.
	if !strings.Contains(err.Error(), ".csv") {
		t.Errorf("the message must say what to do instead, got %q", err)
	}
	if _, err := tabular.ReadXLSX([]byte("not a zip")); err == nil {
		t.Error("a file that is not a workbook must be refused")
	}
}
