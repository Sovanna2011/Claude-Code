package report

import (
	"bytes"
	"fmt"
	"strings"
)

// PDF renders the report as a printable A4 landscape document.
//
// The generator writes PDF 1.4 directly: a catalogue, a page tree, one content
// stream per page and the standard Helvetica fonts, which every reader has, so
// no font needs to be embedded. Landscape A4 is 842 x 595 points.
//
// It is deliberately modest - tabular reports with a header block, column
// headings repeated on every page, and page numbers - because that is what the
// specification asks a printed production report to carry.
const (
	pageWidth   = 842.0
	pageHeight  = 595.0
	marginLeft  = 32.0
	marginRight = 32.0
	marginTop   = 40.0
	marginBot   = 40.0
	titleSize   = 14.0
	metaSize    = 8.0
	headerSize  = 8.5
	bodySize    = 8.0
	lineHeight  = 11.5
)

// PDF renders the table.
func PDF(t Table) ([]byte, error) {
	widths := columnWidths(t)
	pages := paginate(t, widths)

	var objects [][]byte
	// Object numbering: 1 catalogue, 2 page tree, 3 Helvetica, 4 Helvetica-Bold,
	// then for each page a page object and its content stream.
	pageObjIDs := make([]int, len(pages))
	firstPageObj := 5
	for i := range pages {
		pageObjIDs[i] = firstPageObj + i*2
	}

	kids := make([]string, len(pageObjIDs))
	for i, id := range pageObjIDs {
		kids[i] = fmt.Sprintf("%d 0 R", id)
	}

	objects = append(objects, []byte("<< /Type /Catalog /Pages 2 0 R >>"))
	objects = append(objects, []byte(fmt.Sprintf(
		"<< /Type /Pages /Count %d /Kids [%s] >>", len(pages), strings.Join(kids, " "))))
	objects = append(objects, []byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>"))
	objects = append(objects, []byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>"))

	for i, content := range pages {
		contentID := pageObjIDs[i] + 1
		objects = append(objects, []byte(fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.0f %.0f] "+
				"/Resources << /Font << /F1 3 0 R /F2 4 0 R >> >> /Contents %d 0 R >>",
			pageWidth, pageHeight, contentID)))
		objects = append(objects, []byte(fmt.Sprintf(
			"<< /Length %d >>\nstream\n%s\nendstream", len(content), content)))
	}

	return assemblePDF(objects), nil
}

// assemblePDF writes the objects with a cross-reference table and trailer.
func assemblePDF(objects [][]byte) []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	// A binary comment marks the file as binary for transfer tools.
	buf.Write([]byte{'%', 0xE2, 0xE3, 0xCF, 0xD3, '\n'})

	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n", i+1)
		buf.Write(obj)
		buf.WriteString("\nendobj\n")
	}

	xrefStart := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n",
		len(objects)+1, xrefStart)
	return buf.Bytes()
}

// columnWidths distributes the printable width across the columns, honouring
// the width hints in proportion.
func columnWidths(t Table) []float64 {
	printable := pageWidth - marginLeft - marginRight
	if len(t.Columns) == 0 {
		return nil
	}
	total := 0
	for _, c := range t.Columns {
		w := c.Width
		if w <= 0 {
			w = 14
		}
		total += w
	}
	out := make([]float64, len(t.Columns))
	for i, c := range t.Columns {
		w := c.Width
		if w <= 0 {
			w = 14
		}
		out[i] = printable * float64(w) / float64(total)
	}
	return out
}

// paginate lays the table out and returns one content stream per page.
func paginate(t Table, widths []float64) []string {
	var pages []string
	rows := t.Rows
	if len(t.Totals) > 0 {
		rows = append(append([][]Cell{}, rows...), t.Totals)
	}

	rowIndex := 0
	pageNo := 1
	for {
		var b strings.Builder
		y := pageHeight - marginTop

		// Title and metadata, on every page so a loose sheet is still traceable.
		writeText(&b, "F2", titleSize, marginLeft, y, t.Title)
		y -= titleSize + 6

		meta := t.MetadataLines()
		for _, line := range meta {
			if y < marginBot+60 {
				break
			}
			writeText(&b, "F1", metaSize, marginLeft, y, line)
			y -= metaSize + 2
		}
		y -= 6

		// Column headings.
		writeRow(&b, t, widths, headerCells(t), y, "F2", headerSize)
		y -= lineHeight
		writeLine(&b, marginLeft, y+3, pageWidth-marginRight, y+3)
		y -= 2

		// Body rows until the page is full.
		for rowIndex < len(rows) && y > marginBot+lineHeight {
			isTotals := len(t.Totals) > 0 && rowIndex == len(rows)-1
			font := "F1"
			if isTotals {
				font = "F2"
				writeLine(&b, marginLeft, y+lineHeight-2, pageWidth-marginRight, y+lineHeight-2)
			}
			writeRow(&b, t, widths, rows[rowIndex], y, font, bodySize)
			y -= lineHeight
			rowIndex++
		}

		// Notes on the last page only.
		if rowIndex >= len(rows) {
			for _, n := range t.Notes {
				if y < marginBot {
					break
				}
				y -= 4
				writeText(&b, "F1", metaSize, marginLeft, y, n)
				y -= metaSize + 2
			}
		}

		writeText(&b, "F1", metaSize, pageWidth-marginRight-70, marginBot-14,
			fmt.Sprintf("Page %d", pageNo))

		pages = append(pages, b.String())
		if rowIndex >= len(rows) {
			break
		}
		pageNo++
		if pageNo > 2000 { // guard against a pathological layout
			break
		}
	}

	// Now that the total is known, stamp "Page n of m" by rewriting the label.
	for i := range pages {
		pages[i] = strings.Replace(pages[i],
			fmt.Sprintf("(Page %d) Tj", i+1),
			fmt.Sprintf("(Page %d of %d) Tj", i+1, len(pages)), 1)
	}
	return pages
}

func headerCells(t Table) []Cell {
	out := make([]Cell, len(t.Columns))
	for i, c := range t.Columns {
		out[i] = Text(c.Header)
	}
	return out
}

// writeRow places the cells of one row, right-aligning numeric columns.
func writeRow(b *strings.Builder, t Table, widths []float64, cells []Cell, y float64, font string, size float64) {
	x := marginLeft
	for i := range t.Columns {
		w := widths[i]
		if i < len(cells) {
			text := truncate(cells[i].Text, w, size)
			switch t.Columns[i].Align {
			case AlignRight:
				writeText(b, font, size, x+w-textWidth(text, size)-4, y, text)
			case AlignCentre:
				writeText(b, font, size, x+(w-textWidth(text, size))/2, y, text)
			default:
				writeText(b, font, size, x+2, y, text)
			}
		}
		x += w
	}
}

func writeText(b *strings.Builder, font string, size, x, y float64, text string) {
	if text == "" {
		return
	}
	fmt.Fprintf(b, "BT /%s %.1f Tf %.2f %.2f Td (%s) Tj ET\n", font, size, x, y, escapePDF(text))
}

func writeLine(b *strings.Builder, x1, y1, x2, y2 float64) {
	fmt.Fprintf(b, "0.5 w %.2f %.2f m %.2f %.2f l S\n", x1, y1, x2, y2)
}

// textWidth approximates the rendered width. Helvetica averages about 0.5 em
// per character across mixed text, which is close enough for column alignment
// in a tabular report.
func textWidth(s string, size float64) float64 { return float64(len([]rune(s))) * size * 0.5 }

func truncate(s string, width, size float64) string {
	max := int((width - 6) / (size * 0.5))
	if max < 1 {
		max = 1
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 1 {
		return string(runes[:max])
	}
	return string(runes[:max-1]) + "."
}

// escapePDF escapes the characters that are special inside a PDF string and
// drops anything outside WinAnsi, which the standard fonts cannot render.
func escapePDF(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '(', ')', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n', '\r', '\t':
			b.WriteByte(' ')
		default:
			if r < 32 {
				continue
			}
			if r > 255 {
				// Khmer and Thai text needs an embedded font; until that is
				// added, a placeholder is honest about what was dropped.
				b.WriteByte('?')
				continue
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}
