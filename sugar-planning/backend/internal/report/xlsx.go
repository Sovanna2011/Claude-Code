package report

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

// XLSX renders the report as an Office Open XML workbook.
//
// An .xlsx file is a zip archive of XML parts. The minimum set that Excel,
// LibreOffice and Numbers all accept is: the package relationships, the content
// types, a workbook, one worksheet, a styles part and (here) an inline-string
// worksheet so no shared-string table is needed.
//
// Numeric cells are written as numbers, not text, so the recipient can total a
// column without retyping it - which is the whole point of exporting to Excel
// rather than to CSV.
func XLSX(t Table) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	parts := []struct {
		name    string
		content string
	}{
		{"[Content_Types].xml", contentTypesXML},
		{"_rels/.rels", packageRelsXML},
		{"xl/workbook.xml", workbookXML},
		{"xl/_rels/workbook.xml.rels", workbookRelsXML},
		{"xl/styles.xml", stylesXML},
		{"xl/worksheets/sheet1.xml", sheetXML(t)},
	}
	for _, p := range parts {
		w, err := zw.Create(p.name)
		if err != nil {
			return nil, fmt.Errorf("create %s: %w", p.name, err)
		}
		if _, err := w.Write([]byte(p.content)); err != nil {
			return nil, fmt.Errorf("write %s: %w", p.name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("close workbook: %w", err)
	}
	return buf.Bytes(), nil
}

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>
</Types>`

const packageRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`

const workbookXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"
          xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets><sheet name="Report" sheetId="1" r:id="rId1"/></sheets>
</workbook>`

const workbookRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`

// stylesXML defines four cell formats:
//
//	0  normal text
//	1  bold (report title and column headers)
//	2  three-decimal number, thousands separated (tonnages)
//	3  bold three-decimal number (totals)
const stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<numFmts count="1"><numFmt numFmtId="164" formatCode="#,##0.000"/></numFmts>
<fonts count="2">
<font><sz val="11"/><name val="Calibri"/></font>
<font><b/><sz val="11"/><name val="Calibri"/></font>
</fonts>
<fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills>
<borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>
<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>
<cellXfs count="4">
<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>
<xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="1"/>
<xf numFmtId="164" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/>
<xf numFmtId="164" fontId="1" fillId="0" borderId="0" xfId="0" applyNumberFormat="1" applyFont="1"/>
</cellXfs>
<cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles>
</styleSheet>`

// sheetXML lays out the worksheet: title, metadata block, header row, data,
// totals and notes.
func sheetXML(t Table) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
	b.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)

	// Column widths from the column hints.
	if len(t.Columns) > 0 {
		b.WriteString("<cols>")
		for i, c := range t.Columns {
			width := c.Width
			if width <= 0 {
				width = 18
			}
			fmt.Fprintf(&b, `<col min="%d" max="%d" width="%d" customWidth="1"/>`, i+1, i+1, width)
		}
		b.WriteString("</cols>")
	}
	b.WriteString("<sheetData>")

	row := 1
	writeTextRow := func(values []string, style int) {
		fmt.Fprintf(&b, `<row r="%d">`, row)
		for i, v := range values {
			if v == "" {
				continue
			}
			fmt.Fprintf(&b, `<c r="%s%d" t="inlineStr" s="%d"><is><t xml:space="preserve">%s</t></is></c>`,
				columnName(i), row, style, escapeXML(v))
		}
		b.WriteString("</row>")
		row++
	}

	writeTextRow([]string{t.Title}, 1)
	for _, line := range t.MetadataLines() {
		writeTextRow([]string{line}, 0)
	}
	writeTextRow(nil, 0) // spacer

	headers := make([]string, len(t.Columns))
	for i, c := range t.Columns {
		headers[i] = c.Header
	}
	headerRow := row
	writeTextRow(headers, 1)

	writeCells := func(cells []Cell, textStyle, numberStyle int) {
		fmt.Fprintf(&b, `<row r="%d">`, row)
		for i := range t.Columns {
			if i >= len(cells) {
				break
			}
			cell := cells[i]
			ref := fmt.Sprintf("%s%d", columnName(i), row)
			switch {
			case cell.Number != nil && t.Columns[i].Numeric:
				fmt.Fprintf(&b, `<c r="%s" s="%d"><v>%s</v></c>`, ref, numberStyle, cell.Number.String())
			case cell.Text != "":
				fmt.Fprintf(&b, `<c r="%s" t="inlineStr" s="%d"><is><t xml:space="preserve">%s</t></is></c>`,
					ref, textStyle, escapeXML(cell.Text))
			}
		}
		b.WriteString("</row>")
		row++
	}

	for _, r := range t.Rows {
		writeCells(r, 0, 2)
	}
	if len(t.Totals) > 0 {
		writeCells(t.Totals, 1, 3)
	}
	if len(t.Notes) > 0 {
		writeTextRow(nil, 0)
		for _, n := range t.Notes {
			writeTextRow([]string{n}, 0)
		}
	}

	b.WriteString("</sheetData>")
	// Freeze the header row so a 137-day report stays readable while scrolling.
	fmt.Fprintf(&b, `<sheetViews><sheetView workbookViewId="0">`+
		`<pane ySplit="%d" topLeftCell="A%d" activePane="bottomLeft" state="frozen"/>`+
		`</sheetView></sheetViews>`, headerRow, headerRow+1)
	b.WriteString("</worksheet>")

	// sheetViews must precede sheetData in the schema; reorder before returning.
	return reorderSheetViews(b.String())
}

// reorderSheetViews moves the sheetViews element ahead of sheetData, which the
// schema requires and which is easier to do once than to thread through the
// writer above.
func reorderSheetViews(doc string) string {
	start := strings.Index(doc, "<sheetViews>")
	if start < 0 {
		return doc
	}
	end := strings.Index(doc, "</sheetViews>")
	if end < 0 {
		return doc
	}
	end += len("</sheetViews>")
	views := doc[start:end]
	without := doc[:start] + doc[end:]

	anchor := strings.Index(without, "<cols>")
	if anchor < 0 {
		anchor = strings.Index(without, "<sheetData>")
	}
	if anchor < 0 {
		return doc
	}
	return without[:anchor] + views + without[anchor:]
}

// columnName converts a zero-based index to a spreadsheet column name
// (0 -> A, 25 -> Z, 26 -> AA).
func columnName(i int) string {
	name := ""
	for i >= 0 {
		name = string(rune('A'+i%26)) + name
		i = i/26 - 1
	}
	return name
}

func escapeXML(s string) string {
	var b bytes.Buffer
	if err := xml.EscapeText(&b, []byte(s)); err != nil {
		return ""
	}
	return b.String()
}
