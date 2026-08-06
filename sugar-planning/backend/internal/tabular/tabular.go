// Package tabular reads a spreadsheet or a delimited file into rows of text.
//
// It deliberately stops at text. What a column means, how a date in it is
// written and whether a value is valid are questions for the import mapping and
// the domain; this package's whole job is to get the cells out of the file
// without losing any, and to say clearly when it cannot.
//
// The .xlsx reader is the mirror of the writer in internal/report: an .xlsx is a
// zip of XML parts, and reading one needs the worksheet, the shared string table
// it points into, and nothing else.
package tabular

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

// MaxRows bounds a single import. A file larger than this is almost always a
// mistake - a whole season is 137 days - and reading it would hold a request
// open long enough to matter.
const MaxRows = 20000

// MaxColumns bounds the width, for the same reason.
const MaxColumns = 200

// Read parses a file into rows of cells.
//
// The format is taken from the file name rather than sniffed, because a caller
// who uploads a .csv named .xlsx has made a mistake worth being told about
// rather than worked around.
func Read(name string, data []byte, delimiter rune) ([][]string, error) {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".xlsx"):
		return ReadXLSX(data)
	case strings.HasSuffix(lower, ".csv"), strings.HasSuffix(lower, ".txt"),
		strings.HasSuffix(lower, ".tsv"):
		if strings.HasSuffix(lower, ".tsv") && delimiter == 0 {
			delimiter = '\t'
		}
		return ReadCSV(data, delimiter)
	case name == "":
		return ReadCSV(data, delimiter)
	}
	return nil, fmt.Errorf("%q is not a file this system can read; use .csv, .tsv or .xlsx", name)
}

// ReadCSV parses delimited text.
//
// A zero delimiter means comma. The rows are returned ragged rather than padded
// to the widest: a short row is a fact about the file, and the mapping decides
// whether a missing column matters.
func ReadCSV(data []byte, delimiter rune) ([][]string, error) {
	data = stripBOM(data)
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("the file is not UTF-8 text; save it as UTF-8 and try again")
	}

	r := csv.NewReader(bytes.NewReader(data))
	if delimiter != 0 {
		r.Comma = delimiter
	}
	// Ragged rows are allowed here and reported by the mapping, which can say
	// which column was missing rather than only that the row was short.
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	var rows [][]string
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("row %d could not be read: %w", len(rows)+1, err)
		}
		if len(record) > MaxColumns {
			return nil, fmt.Errorf("row %d has %d columns, which is more than the %d this system reads",
				len(rows)+1, len(record), MaxColumns)
		}
		rows = append(rows, record)
		if len(rows) > MaxRows {
			return nil, fmt.Errorf("the file has more than %d rows", MaxRows)
		}
	}
	return rows, nil
}

// stripBOM removes the byte order mark Excel writes at the head of a CSV, which
// would otherwise become part of the first column's header and make it match
// nothing.
func stripBOM(data []byte) []byte {
	return bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
}

// ---------------------------------------------------------------------------
// .xlsx
// ---------------------------------------------------------------------------

// ReadXLSX parses the first worksheet of an Office Open XML workbook.
//
// Only the first sheet is read, and that is said out loud in the API rather
// than left to be discovered: a workbook whose data is on the third tab needs
// that tab exported, and silently reading the wrong sheet would be worse than
// refusing.
func ReadXLSX(data []byte) ([][]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("the file is not a readable .xlsx workbook: %w", err)
	}

	var sheet, shared *zip.File
	for _, f := range zr.File {
		switch f.Name {
		case "xl/worksheets/sheet1.xml":
			sheet = f
		case "xl/sharedStrings.xml":
			shared = f
		}
	}
	if sheet == nil {
		return nil, fmt.Errorf("the workbook has no first worksheet this system can read; " +
			"save the sheet you want as .csv instead")
	}

	strs, err := readSharedStrings(shared)
	if err != nil {
		return nil, err
	}
	return readSheet(sheet, strs)
}

// readSharedStrings reads the table most .xlsx writers put text in. A workbook
// written with inline strings has no such part, which is not an error.
func readSharedStrings(f *zip.File) ([]string, error) {
	if f == nil {
		return nil, nil
	}
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("reading the workbook's text: %w", err)
	}
	defer rc.Close()

	var doc struct {
		Items []struct {
			// A string can be split into runs by formatting, and every run's
			// text is part of the value.
			Text string   `xml:"t"`
			Runs []string `xml:"r>t"`
		} `xml:"si"`
	}
	if err := xml.NewDecoder(rc).Decode(&doc); err != nil {
		return nil, fmt.Errorf("reading the workbook's text: %w", err)
	}

	out := make([]string, len(doc.Items))
	for i, item := range doc.Items {
		if len(item.Runs) > 0 {
			out[i] = strings.Join(item.Runs, "")
			continue
		}
		out[i] = item.Text
	}
	return out, nil
}

func readSheet(f *zip.File, shared []string) ([][]string, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("reading the worksheet: %w", err)
	}
	defer rc.Close()

	var doc struct {
		Rows []struct {
			Cells []struct {
				Ref    string `xml:"r,attr"`
				Type   string `xml:"t,attr"`
				Value  string `xml:"v"`
				Inline string `xml:"is>t"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	if err := xml.NewDecoder(rc).Decode(&doc); err != nil {
		return nil, fmt.Errorf("the workbook's first worksheet could not be read (%w); "+
			"save the sheet you want as .csv instead", err)
	}
	if len(doc.Rows) > MaxRows {
		return nil, fmt.Errorf("the sheet has more than %d rows", MaxRows)
	}

	rows := make([][]string, 0, len(doc.Rows))
	for _, row := range doc.Rows {
		// Cells are placed by their reference rather than by their order,
		// because an empty cell is simply absent from the XML - and a row whose
		// second column is blank must not shift its third column left.
		width := 0
		positions := make([]int, len(row.Cells))
		for i, c := range row.Cells {
			col := columnIndex(c.Ref)
			if col < 0 {
				col = i
			}
			positions[i] = col
			if col+1 > width {
				width = col + 1
			}
		}
		if width > MaxColumns {
			return nil, fmt.Errorf("a row has %d columns, which is more than the %d this system reads",
				width, MaxColumns)
		}

		cells := make([]string, width)
		for i, c := range row.Cells {
			cells[positions[i]] = cellText(c.Type, c.Value, c.Inline, shared)
		}
		rows = append(rows, cells)
	}
	return rows, nil
}

func cellText(cellType, value, inline string, shared []string) string {
	switch cellType {
	case "s":
		i, err := strconv.Atoi(value)
		if err != nil || i < 0 || i >= len(shared) {
			return ""
		}
		return shared[i]
	case "inlineStr":
		return inline
	case "str", "n", "b", "":
		if inline != "" {
			return inline
		}
		return value
	}
	return value
}

// columnIndex turns a cell reference such as "AB7" into a zero-based column
// number. It returns -1 for a reference it cannot read.
func columnIndex(ref string) int {
	col := 0
	seen := false
	for _, r := range ref {
		switch {
		case r >= 'A' && r <= 'Z':
			col = col*26 + int(r-'A') + 1
			seen = true
		case r >= 'a' && r <= 'z':
			col = col*26 + int(r-'a') + 1
			seen = true
		case r >= '0' && r <= '9':
			if !seen {
				return -1
			}
			return col - 1
		default:
			return -1
		}
	}
	if !seen {
		return -1
	}
	return col - 1
}
