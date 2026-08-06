package report

import (
	"bytes"
	"encoding/csv"
	"fmt"
)

// CSV renders the report as UTF-8 CSV with the metadata preserved as header
// lines above the table.
func CSV(t Table) ([]byte, error) {
	var buf bytes.Buffer
	// UTF-8 byte order mark: without it Excel on Windows opens the file in the
	// local code page and mangles Khmer and Thai names.
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	w := csv.NewWriter(&buf)
	if err := w.Write([]string{t.Title}); err != nil {
		return nil, fmt.Errorf("write csv title: %w", err)
	}
	for _, line := range t.MetadataLines() {
		if err := w.Write([]string{line}); err != nil {
			return nil, fmt.Errorf("write csv metadata: %w", err)
		}
	}
	if err := w.Write(nil); err != nil {
		return nil, err
	}

	headers := make([]string, len(t.Columns))
	for i, c := range t.Columns {
		headers[i] = c.Header
	}
	if err := w.Write(headers); err != nil {
		return nil, fmt.Errorf("write csv header: %w", err)
	}

	for _, row := range t.Rows {
		if err := w.Write(cellsToStrings(row, len(t.Columns))); err != nil {
			return nil, fmt.Errorf("write csv row: %w", err)
		}
	}
	if len(t.Totals) > 0 {
		if err := w.Write(cellsToStrings(t.Totals, len(t.Columns))); err != nil {
			return nil, fmt.Errorf("write csv totals: %w", err)
		}
	}
	for _, note := range t.Notes {
		if err := w.Write([]string{note}); err != nil {
			return nil, err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("flush csv: %w", err)
	}
	return buf.Bytes(), nil
}

// cellsToStrings pads or truncates a row to the column count so that a ragged
// row cannot corrupt the file.
func cellsToStrings(cells []Cell, width int) []string {
	out := make([]string, width)
	for i := range out {
		if i < len(cells) {
			out[i] = cells[i].Text
		}
	}
	return out
}
