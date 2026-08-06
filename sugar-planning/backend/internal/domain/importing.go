package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// This file is the import model of section 22: a controlled mapping template
// that says which column of a file holds which field, and the parsing and
// validation that turn a sheet into rows this system will accept.
//
// The shape of it is set by one decision. A file is never posted as it arrives:
// it is parsed into a staging area, validated row by row, shown to somebody, and
// committed only when they say so. An import that wrote straight through would
// be a way of getting past every rule the rest of the system enforces, and the
// spreadsheet this application replaces is exactly where the bad data comes
// from.

// ImportKind is what a file is a file of.
//
// Each kind maps onto a service that already exists, so an imported row is
// subject to the same validation, the same authorisation and the same audit
// trail as a row somebody types into the planning grid.
type ImportKind string

const (
	ImportCane       ImportKind = "DAILY_CANE"
	ImportProduction ImportKind = "DAILY_PRODUCTION"
	ImportShipment   ImportKind = "DAILY_SHIPMENT"
	ImportStorage    ImportKind = "DAILY_STORAGE"
)

// ValidImportKind reports whether the value is a kind this system imports.
func ValidImportKind(k ImportKind) bool {
	switch k {
	case ImportCane, ImportProduction, ImportShipment, ImportStorage:
		return true
	}
	return false
}

// ImportFieldType says how a cell is read.
type ImportFieldType string

const (
	FieldDate    ImportFieldType = "DATE"
	FieldDecimal ImportFieldType = "DECIMAL"
	FieldText    ImportFieldType = "TEXT"
	// FieldCode is a business key - a product code, a warehouse code - which is
	// resolved to an id by the service rather than here. The domain has no
	// repository and cannot look one up.
	FieldCode ImportFieldType = "CODE"
)

// ImportField describes one field a kind can carry.
type ImportField struct {
	Name string          `json:"name"`
	Type ImportFieldType `json:"type"`
	// Label is what the mapping screen shows.
	Label string `json:"label"`
	// Required means a row without it cannot be built at all.
	Required bool `json:"required"`
	// Entity names what a CODE field resolves against, so the service knows
	// which repository to ask.
	Entity string `json:"entity,omitempty"`
	// Note explains a field whose meaning is not obvious from its label.
	Note string `json:"note,omitempty"`
}

// ImportFields is the catalogue: what each kind of file may contain.
//
// It is deliberately explicit rather than derived from the struct tags of the
// row types. A field appearing in an import is a decision - crushed cane may be
// imported, a row version may not - and reflection over a struct would make
// every column that exists importable the moment somebody added one.
var ImportFields = map[ImportKind][]ImportField{
	ImportCane: {
		{Name: "businessDate", Type: FieldDate, Label: "Date", Required: true},
		{Name: "shiftCode", Type: FieldCode, Label: "Shift", Entity: "shift"},
		{Name: "caneAvailable", Type: FieldDecimal, Label: "Cane available (t)"},
		{Name: "caneDelivered", Type: FieldDecimal, Label: "Cane delivered (t)"},
		{Name: "caneAccepted", Type: FieldDecimal, Label: "Cane accepted (t)"},
		{Name: "caneRejected", Type: FieldDecimal, Label: "Cane rejected (t)"},
		{Name: "caneDiverted", Type: FieldDecimal, Label: "Cane diverted (t)"},
		{Name: "caneCrushed", Type: FieldDecimal, Label: "Cane crushed (t)"},
		{Name: "crushRateTph", Type: FieldDecimal, Label: "Crush rate (t/h)"},
		{Name: "availableHours", Type: FieldDecimal, Label: "Available hours"},
		{Name: "stoppageHours", Type: FieldDecimal, Label: "Stoppage hours"},
		{Name: "reasonCode", Type: FieldText, Label: "Reason code"},
		{Name: "note", Type: FieldText, Label: "Note"},
	},
	ImportProduction: {
		{Name: "businessDate", Type: FieldDate, Label: "Date", Required: true},
		{Name: "productCode", Type: FieldCode, Label: "Product", Entity: "product", Required: true},
		{Name: "packagingCode", Type: FieldCode, Label: "Packaging", Entity: "packaging"},
		{Name: "lineCode", Type: FieldCode, Label: "Production line", Entity: "line"},
		{Name: "shiftCode", Type: FieldCode, Label: "Shift", Entity: "shift"},
		{Name: "quantity", Type: FieldDecimal, Label: "Good output (t)"},
		{Name: "remeltInput", Type: FieldDecimal, Label: "Remelt input (t)",
			Note: "Raw sugar consumed to make this output."},
		{Name: "processLoss", Type: FieldDecimal, Label: "Process loss (t)"},
		{Name: "rework", Type: FieldDecimal, Label: "Rework (t)"},
		{Name: "rejected", Type: FieldDecimal, Label: "Rejected (t)"},
		{Name: "holdQty", Type: FieldDecimal, Label: "Quality hold (t)"},
		{Name: "reasonCode", Type: FieldText, Label: "Reason code"},
		{Name: "note", Type: FieldText, Label: "Note"},
	},
	ImportShipment: {
		{Name: "businessDate", Type: FieldDate, Label: "Date", Required: true},
		{Name: "productCode", Type: FieldCode, Label: "Product", Entity: "product", Required: true},
		{Name: "channelCode", Type: FieldCode, Label: "Channel", Entity: "channel", Required: true},
		{Name: "warehouseCode", Type: FieldCode, Label: "Warehouse", Entity: "warehouse"},
		{Name: "quantity", Type: FieldDecimal, Label: "Shipped (t)"},
		{Name: "note", Type: FieldText, Label: "Note"},
	},
	// The stock ledger imports movements, never balances. The opening and
	// closing balances are calculated forward from the movements, and a file
	// that could set them directly would let a spreadsheet contradict the
	// ledger - which is the disagreement this whole application exists to end.
	// A physical count is a movement in that sense: it is a measurement, and the
	// ledger reports the difference rather than being overwritten by it.
	ImportStorage: {
		{Name: "businessDate", Type: FieldDate, Label: "Date", Required: true},
		{Name: "warehouseCode", Type: FieldCode, Label: "Warehouse", Entity: "warehouse", Required: true},
		{Name: "productCode", Type: FieldCode, Label: "Product", Entity: "product", Required: true},
		{Name: "productionReceipt", Type: FieldDecimal, Label: "Production receipt (t)"},
		{Name: "transferIn", Type: FieldDecimal, Label: "Transfer in (t)"},
		{Name: "transferOut", Type: FieldDecimal, Label: "Transfer out (t)"},
		{Name: "repackIn", Type: FieldDecimal, Label: "Repack in (t)"},
		{Name: "repackOut", Type: FieldDecimal, Label: "Repack out (t)"},
		{Name: "remeltIssue", Type: FieldDecimal, Label: "Remelt issue (t)"},
		{Name: "shipmentQty", Type: FieldDecimal, Label: "Shipped (t)"},
		{Name: "adjustment", Type: FieldDecimal, Label: "Adjustment (t)"},
		{Name: "processLoss", Type: FieldDecimal, Label: "Process loss (t)"},
		{Name: "holdQty", Type: FieldDecimal, Label: "Quality hold (t)"},
		{Name: "physicalBalance", Type: FieldDecimal, Label: "Physical count (t)",
			Note: "A count. The ledger reports the difference rather than being overwritten by it."},
	},
}

// FieldsFor returns the catalogue for a kind.
func FieldsFor(kind ImportKind) []ImportField { return ImportFields[kind] }

// FieldFor returns one field of a kind.
func FieldFor(kind ImportKind, name string) (ImportField, bool) {
	for _, f := range ImportFields[kind] {
		if f.Name == name {
			return f, true
		}
	}
	return ImportField{}, false
}

// KeyFields are the fields that identify a row within its kind. Two rows with
// the same key are the same row, which is what duplicate detection is about and
// what an upsert matches on.
func KeyFields(kind ImportKind) []string {
	switch kind {
	case ImportCane:
		return []string{"businessDate", "shiftCode"}
	case ImportProduction:
		return []string{"businessDate", "productCode", "packagingCode", "lineCode", "shiftCode"}
	case ImportShipment:
		return []string{"businessDate", "productCode", "channelCode", "warehouseCode"}
	case ImportStorage:
		return []string{"businessDate", "warehouseCode", "productCode"}
	}
	return []string{"businessDate"}
}

// ---------------------------------------------------------------------------
// The mapping template
// ---------------------------------------------------------------------------

// ColumnMapping ties one field to one column of the file.
type ColumnMapping struct {
	Field string `json:"field"`
	// Header is the column heading to look for, matched case-insensitively and
	// ignoring surrounding space. A site whose sheet says "Cane crushed (MT)"
	// maps it once and never thinks about it again.
	Header string `json:"header,omitempty"`
	// Column is a zero-based position, used when a sheet has no headings at all.
	// Header wins when both are given.
	Column *int `json:"column,omitempty"`
	// Default fills a cell the file leaves blank. It is applied before
	// validation, so a required field with a default is satisfied.
	Default string `json:"default,omitempty"`
}

// ImportMapping is the controlled template: which columns of which shape of file
// hold which fields, and how the numbers and dates in it are written.
//
// It is master data rather than a request parameter because it is the thing a
// site gets right once. An import that asked for the column positions every time
// would be re-entering the mapping on every upload, which is where a mistake
// gets made.
type ImportMapping struct {
	ID      string          `json:"id"`
	Code    string          `json:"code"`
	Name    string          `json:"name"`
	Kind    ImportKind      `json:"kind"`
	Columns []ColumnMapping `json:"columns"`
	// HeaderRow is the one-based row the headings are on. Zero means the file
	// has no headings and the columns are matched by position.
	HeaderRow int `json:"headerRow"`
	// FirstDataRow is the one-based row the data starts on. Zero means the row
	// after the headings.
	FirstDataRow int `json:"firstDataRow"`
	// Delimiter is the CSV separator, as a single character. Empty means comma.
	Delimiter string `json:"delimiter,omitempty"`
	// DateFormat is a Go layout - "2006-01-02", "02/01/2006". Empty accepts the
	// ISO form and the two common European and American ones.
	DateFormat string `json:"dateFormat,omitempty"`
	// DecimalComma says the file writes 1.234,56 rather than 1,234.56. Getting
	// this wrong turns a thousand tons into one, so it is a deliberate setting
	// rather than a guess made per cell.
	DecimalComma bool   `json:"decimalComma"`
	Note         string `json:"note,omitempty"`
	Validity
	AuditFields
}

// Validate checks a mapping before it is stored.
func (m ImportMapping) Validate() error {
	verr := &ValidationError{}
	if strings.TrimSpace(m.Code) == "" {
		verr.Add("code", "REQUIRED", "a mapping needs a code")
	}
	if !ValidImportKind(m.Kind) {
		verr.Add("kind", "INVALID", fmt.Sprintf("%q is not a kind of file this system imports", m.Kind))
	}
	if m.HeaderRow < 0 || m.FirstDataRow < 0 {
		verr.Add("headerRow", "OUT_OF_RANGE", "a row number cannot be negative")
	}
	if m.FirstDataRow > 0 && m.HeaderRow > 0 && m.FirstDataRow <= m.HeaderRow {
		verr.Add("firstDataRow", "OUT_OF_RANGE",
			"the data starts on or before the heading row, so the headings would be imported as data")
	}
	if len([]rune(m.Delimiter)) > 1 {
		verr.Add("delimiter", "INVALID", "the delimiter is a single character")
	}
	if m.DateFormat != "" && !validDateLayout(m.DateFormat) {
		verr.Add("dateFormat", "INVALID",
			fmt.Sprintf("%q is not a date format this system understands; "+
				"write it as an example of 2 January 2006, such as 02/01/2006", m.DateFormat))
	}
	if len(m.Columns) == 0 {
		verr.Add("columns", "EMPTY", "a mapping with no columns imports nothing")
	}

	seen := map[string]bool{}
	for i, c := range m.Columns {
		field, ok := FieldFor(m.Kind, c.Field)
		if !ok {
			verr.AddRow(i, "field", "UNKNOWN",
				fmt.Sprintf("%q is not a field of a %s file", c.Field, m.Kind))
			continue
		}
		if seen[c.Field] {
			verr.AddRow(i, "field", "DUPLICATE",
				fmt.Sprintf("%s is mapped twice; one field comes from one column", field.Label))
			continue
		}
		seen[c.Field] = true
		if c.Header == "" && c.Column == nil && c.Default == "" {
			verr.AddRow(i, "header", "REQUIRED",
				fmt.Sprintf("%s names neither a column nor a default, so it would never be filled",
					field.Label))
		}
		if c.Column != nil && *c.Column < 0 {
			verr.AddRow(i, "column", "OUT_OF_RANGE", "a column position cannot be negative")
		}
	}

	// A file that cannot supply the required fields is not worth uploading.
	for _, f := range FieldsFor(m.Kind) {
		if f.Required && !seen[f.Name] {
			verr.Add("columns", "MISSING_REQUIRED",
				fmt.Sprintf("%s is required for a %s file and is not mapped", f.Label, m.Kind))
		}
	}
	return verr.OrNil()
}

// validDateLayout reports whether a string is a Go time layout rather than the
// dd/mm/yyyy notation people expect to work.
//
// The test is that two different dates render differently through it. A layout
// with no reference components in it - "dd/mm/yyyy" has none - renders every
// date as itself, and would then "parse" only strings identical to itself, which
// is a mapping that silently rejects every date in the file.
func validDateLayout(layout string) bool {
	reference := time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)
	other := time.Date(2026, 12, 25, 9, 30, 0, 0, time.UTC)
	rendered := reference.Format(layout)
	if rendered == other.Format(layout) {
		return false
	}
	_, err := time.Parse(layout, rendered)
	return err == nil
}

// DataStartsAt is the one-based row the data begins on.
func (m ImportMapping) DataStartsAt() int {
	if m.FirstDataRow > 0 {
		return m.FirstDataRow
	}
	if m.HeaderRow > 0 {
		return m.HeaderRow + 1
	}
	return 1
}

// Separator is the CSV delimiter as a rune, or zero for the default.
func (m ImportMapping) Separator() rune {
	for _, r := range m.Delimiter {
		return r
	}
	return 0
}

// ---------------------------------------------------------------------------
// The job
// ---------------------------------------------------------------------------

// Import job statuses.
const (
	// ImportPending is a job whose file has been read but not yet judged. It is
	// a transient state; a read that finishes leaves the job VALIDATED whether
	// or not every row passed.
	ImportPending   = "PENDING"
	ImportValidated = "VALIDATED"
	ImportCommitted = "COMMITTED"
	ImportFailed    = "FAILED"
	ImportCancelled = "CANCELLED"
)

// ImportJob is one upload: what was read, what was wrong with it, and whether
// anybody has committed it.
type ImportJob struct {
	ID        string     `json:"id"`
	Kind      ImportKind `json:"kind"`
	MappingID string     `json:"mappingId,omitempty"`
	VersionID string     `json:"versionId,omitempty"`
	FactoryID string     `json:"factoryId,omitempty"`
	FileName  string     `json:"fileName"`
	Status    string     `json:"status"`
	TotalRows int        `json:"totalRows"`
	ValidRows int        `json:"validRows"`
	ErrorRows int        `json:"errorRows"`
	// Errors holds the problems that belong to the file rather than to a row:
	// a heading that was mapped and is not there, a column nobody mapped.
	Errors      []FieldError `json:"errors,omitempty"`
	Note        string       `json:"note,omitempty"`
	CommittedAt *time.Time   `json:"committedAt,omitempty"`
	CommittedBy string       `json:"committedBy,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
	CreatedBy   string       `json:"createdBy"`
}

// IsOpen reports whether the job can still be committed or cancelled.
func (j ImportJob) IsOpen() bool {
	return j.Status == ImportPending || j.Status == ImportValidated
}

// CanCommit reports whether there is anything to commit.
func (j ImportJob) CanCommit() bool { return j.IsOpen() && j.ValidRows > 0 }

// ---------------------------------------------------------------------------
// Reading a file through a mapping
// ---------------------------------------------------------------------------

// ImportRow is one row of a file after it has been read through the mapping and
// before it has been resolved against master data.
type ImportRow struct {
	// RowNo is the one-based row in the file, so an error names the line
	// somebody is looking at in Excel rather than an offset into an array.
	RowNo int `json:"rowNo"`
	// Values holds the mapped fields, still as text: the codes have not been
	// resolved and the decimals have not been rounded.
	Values map[string]string `json:"values"`
	// Errors are the problems found in this row.
	Errors []FieldError `json:"errors,omitempty"`
	// Duplicate marks a row whose key repeats a row earlier in the same file.
	Duplicate bool `json:"duplicate,omitempty"`
	// Replaces marks a row whose key already exists in the plan, so committing
	// it overwrites rather than adds. This is not an error - re-importing a
	// corrected sheet is the normal case - but somebody should see it before
	// they press the button.
	Replaces bool `json:"replaces,omitempty"`
}

// Key is the row's natural key, for duplicate detection and upsert matching.
func (r ImportRow) Key(kind ImportKind) string {
	parts := make([]string, 0, 4)
	for _, name := range KeyFields(kind) {
		parts = append(parts, strings.ToUpper(strings.TrimSpace(r.Values[name])))
	}
	return strings.Join(parts, "|")
}

// OK reports whether the row can be committed.
func (r ImportRow) OK() bool { return len(r.Errors) == 0 }

// ReadResult is what reading a whole file produced.
type ReadResult struct {
	Rows []ImportRow `json:"rows"`
	// Unmapped names the file's headings that the mapping ignores. It is not an
	// error and it is reported anyway: a column somebody expected to be
	// imported and is not is the failure that goes unnoticed.
	Unmapped []string `json:"unmapped,omitempty"`
	// Missing names mapped headings the file does not have. A field with a
	// default still appears here: a renamed column looks exactly like an absent
	// one, and a default quietly standing in for it is how a wrong figure gets
	// imported without anybody noticing.
	Missing []string `json:"missing,omitempty"`
}

// ReadRows applies a mapping to the cells of a file.
//
// It does not resolve codes and does not touch the database: what comes out is
// text keyed by field name, with the problems that can be seen from the file
// alone already attached. The service resolves the rest.
func ReadRows(m ImportMapping, cells [][]string) (ReadResult, error) {
	if !ValidImportKind(m.Kind) {
		return ReadResult{}, fmt.Errorf("%w: %q is not a kind of file this system imports",
			ErrValidation, m.Kind)
	}
	if len(cells) == 0 {
		return ReadResult{}, fmt.Errorf("%w: the file has no rows", ErrValidation)
	}

	index, result, err := resolveColumns(m, cells)
	if err != nil {
		return ReadResult{}, err
	}

	start := m.DataStartsAt()
	seen := map[string]int{}
	for i := start - 1; i < len(cells); i++ {
		row := readRow(m, index, cells[i], i+1)
		if row == nil {
			continue
		}
		if first, ok := seen[row.Key(m.Kind)]; ok {
			row.Duplicate = true
			row.Errors = append(row.Errors, FieldError{
				Field: "businessDate", Code: "DUPLICATE",
				Message: fmt.Sprintf("this row repeats row %d of the file; "+
					"the later one would silently overwrite the earlier", first),
			})
		} else {
			seen[row.Key(m.Kind)] = row.RowNo
		}
		result.Rows = append(result.Rows, *row)
	}
	if len(result.Rows) == 0 {
		return ReadResult{}, fmt.Errorf(
			"%w: the file has no data rows below row %d", ErrValidation, start-1)
	}
	return result, nil
}

// resolveColumns works out which column of the file each field comes from.
func resolveColumns(m ImportMapping, cells [][]string) (map[string]int, ReadResult, error) {
	var result ReadResult
	index := map[string]int{}

	var headers []string
	if m.HeaderRow > 0 {
		if m.HeaderRow > len(cells) {
			return nil, result, fmt.Errorf(
				"%w: the mapping says the headings are on row %d, and the file has only %d rows",
				ErrValidation, m.HeaderRow, len(cells))
		}
		headers = cells[m.HeaderRow-1]
	}

	used := map[int]bool{}
	for _, c := range m.Columns {
		switch {
		case c.Header != "" && headers != nil:
			at := findHeader(headers, c.Header)
			if at < 0 {
				// Reported whether or not a default covers it. A column that has
				// been renamed in the source sheet looks exactly like a column
				// that was never there, and a default quietly standing in for it
				// is how a wrong figure gets imported without anybody noticing.
				result.Missing = append(result.Missing, c.Header)
				continue
			}
			index[c.Field], used[at] = at, true
		case c.Column != nil:
			index[c.Field], used[*c.Column] = *c.Column, true
		}
	}

	for i, h := range headers {
		if h = strings.TrimSpace(h); h != "" && !used[i] {
			result.Unmapped = append(result.Unmapped, h)
		}
	}
	sort.Strings(result.Missing)
	return index, result, nil
}

func findHeader(headers []string, want string) int {
	want = strings.ToLower(strings.TrimSpace(want))
	for i, h := range headers {
		if strings.ToLower(strings.TrimSpace(h)) == want {
			return i
		}
	}
	return -1
}

// readRow builds one row, or nil when the line is blank.
func readRow(m ImportMapping, index map[string]int, cells []string, rowNo int) *ImportRow {
	row := ImportRow{RowNo: rowNo, Values: map[string]string{}}

	any := false
	for _, c := range m.Columns {
		value := c.Default
		if at, ok := index[c.Field]; ok && at < len(cells) {
			if cell := strings.TrimSpace(cells[at]); cell != "" {
				value, any = cell, true
			}
		}
		if value == "" {
			continue
		}
		row.Values[c.Field] = value
	}
	if !any {
		// A wholly blank line is the trailing rubbish every exported sheet has,
		// not a row somebody meant to import.
		return nil
	}

	for _, field := range FieldsFor(m.Kind) {
		value, present := row.Values[field.Name]
		if !present || value == "" {
			if field.Required {
				row.Errors = append(row.Errors, FieldError{
					Field: field.Name, Code: "REQUIRED",
					Message: field.Label + " is empty",
				})
			}
			continue
		}
		normalised, err := normalise(field, value, m)
		if err != nil {
			row.Errors = append(row.Errors, FieldError{
				Field: field.Name, Code: "INVALID", Message: err.Error(),
			})
			continue
		}
		row.Values[field.Name] = normalised
	}
	return &row
}

// normalise turns a cell into the canonical form the rest of the system uses:
// an ISO date, a plain decimal, trimmed text.
func normalise(field ImportField, value string, m ImportMapping) (string, error) {
	switch field.Type {
	case FieldDate:
		d, err := ParseImportDate(value, m.DateFormat)
		if err != nil {
			return "", fmt.Errorf("%s is %q, which is not a date this mapping can read (%s)",
				field.Label, value, describeFormat(m.DateFormat))
		}
		return string(d), nil

	case FieldDecimal:
		d, err := ParseImportDecimal(value, m.DecimalComma)
		if err != nil {
			return "", fmt.Errorf("%s is %q, which is not a number", field.Label, value)
		}
		if d.IsNegative() {
			return "", fmt.Errorf("%s is %s; a quantity cannot be negative", field.Label, d)
		}
		return d.String(), nil
	}
	return value, nil
}

func describeFormat(layout string) string {
	if layout == "" {
		return "expected 2026-12-01, 01/12/2026 or 12/01/2026"
	}
	return "expected " + time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC).Format(layout)
}

// ParseImportDate reads a date in the mapping's format.
//
// With no format given it accepts the ISO form and the two common written ones.
// Day-first is tried before month-first because this is a Cambodian site, and
// the ambiguity is reported by the mapping screen rather than resolved silently:
// a template that says which it is removes the guess entirely.
func ParseImportDate(value, layout string) (BusinessDate, error) {
	value = strings.TrimSpace(value)
	layouts := []string{layout}
	if layout == "" {
		layouts = []string{"2006-01-02", "02/01/2006", "01/02/2006", "2/1/2006",
			"02-01-2006", "2006/01/02", "02.01.2006"}
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, value); err == nil {
			return NewBusinessDate(t), nil
		}
	}
	// A sheet whose date column was formatted as a number gives a serial: the
	// count of days since 1899-12-30, which is Excel's epoch.
	if serial, err := ParseImportDecimal(value, false); err == nil && !serial.IsZero() {
		days := serial.IntPart()
		if days > 20000 && days < 80000 {
			epoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
			return NewBusinessDate(epoch.AddDate(0, 0, int(days))), nil
		}
	}
	return "", fmt.Errorf("%q is not a date", value)
}

// ParseImportDecimal reads a number as a spreadsheet writes it: with thousands
// separators, possibly with a currency symbol, possibly in parentheses for a
// negative, and in either the English or the European convention.
func ParseImportDecimal(value string, decimalComma bool) (Dec, error) {
	s := strings.TrimSpace(value)
	if s == "" {
		return Zero, nil
	}

	negative := false
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		// Accounting notation: (1,205) is minus 1,205.
		negative, s = true, strings.TrimSuffix(strings.TrimPrefix(s, "("), ")")
	}

	// Strip everything that is decoration rather than value: spaces of every
	// kind, currency symbols, a trailing per-cent sign.
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r == '-', r == '+':
			b.WriteRune(r)
		case r == '.' || r == ',':
			b.WriteRune(r)
		case r == ' ' || r == ' ' || r == '\'' || r == '$' || r == '%':
			// dropped
		default:
			return Zero, fmt.Errorf("%q is not a number", value)
		}
	}
	s = b.String()
	if s == "" {
		return Zero, fmt.Errorf("%q is not a number", value)
	}

	if decimalComma {
		s = strings.ReplaceAll(s, ".", "")
		s = strings.Replace(s, ",", ".", 1)
		s = strings.ReplaceAll(s, ",", "")
	} else {
		s = strings.ReplaceAll(s, ",", "")
	}

	d, err := ParseDec(s)
	if err != nil {
		return Zero, fmt.Errorf("%q is not a number", value)
	}
	if negative {
		d = d.Neg()
	}
	return d, nil
}
