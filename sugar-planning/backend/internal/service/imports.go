package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
	"github.com/kss/sugarplan/internal/tabular"
)

// Imports is the controlled import of section 22.
//
// The shape of it is set by one decision, and everything else follows: a file is
// never posted as it arrives. It is read through a mapping template, staged,
// validated row by row, shown to somebody, and committed only when they say so.
// An import that wrote straight through would be a way past every rule the rest
// of the system enforces - and the spreadsheet this application replaces is
// exactly where the bad data comes from.
//
// The commit itself goes through the ordinary planning service, so an imported
// row is subject to the same validation, the same period locking, the same
// authorisation and the same audit trail as a row somebody types in.
type Imports struct {
	store    store.Store
	planning *Planning
	now      func() time.Time
}

// NewImports builds the service.
func NewImports(s store.Store, planning *Planning, now func() time.Time) *Imports {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Imports{store: s, planning: planning, now: now}
}

// ---------------------------------------------------------------------------
// Mapping templates
// ---------------------------------------------------------------------------

// ListMappings returns the templates a site has set up.
func (i *Imports) ListMappings(ctx context.Context, kind string) ([]domain.ImportMapping, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return nil, err
	}
	return i.store.Imports().ListMappings(ctx, kind)
}

// SaveMapping stores a template.
//
// It is master data, behind the master-data permission: a mapping decides what
// a column means, and somebody who can change that can change what every future
// import records without touching a single figure.
func (i *Imports) SaveMapping(ctx context.Context, m domain.ImportMapping) (domain.ImportMapping, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermMasterDataWrite); err != nil {
		return domain.ImportMapping{}, err
	}
	m.Code = strings.ToUpper(strings.TrimSpace(m.Code))
	if err := m.Validate(); err != nil {
		return domain.ImportMapping{}, err
	}

	var saved domain.ImportMapping
	err := i.store.InTx(ctx, func(tx store.Store) error {
		var err error
		saved, err = tx.Imports().SaveMapping(ctx, m, caller.Username)
		if err != nil {
			return err
		}
		return writeAudit(ctx, tx, i.now, auditEntry{
			action: "SAVE", entity: "import_mapping", entityID: saved.ID, after: saved,
		})
	})
	return saved, err
}

// Fields returns the catalogue a mapping screen builds itself from: what each
// kind of file may contain, and which of those are required.
func (i *Imports) Fields(ctx context.Context) (map[domain.ImportKind][]domain.ImportField, error) {
	if err := auth.FromContext(ctx).Require(domain.PermPlanRead); err != nil {
		return nil, err
	}
	return domain.ImportFields, nil
}

// ---------------------------------------------------------------------------
// Staging
// ---------------------------------------------------------------------------

// StageRequest is an upload.
type StageRequest struct {
	// MappingCode or MappingID names the template. The code is what a script
	// uses; the screen has the id to hand.
	MappingCode string `json:"mappingCode,omitempty"`
	MappingID   string `json:"mappingId,omitempty"`
	// VersionID is the plan version the rows will be written to.
	VersionID string `json:"versionId"`
	FileName  string `json:"fileName"`
	// Content is the file. It is passed as bytes rather than a path because
	// nothing here writes to disk: the parsed rows are the record.
	Content []byte `json:"-"`
	// Series says whether the file holds plan or actual figures. It is asked
	// for rather than inferred, because a file of actuals loaded into the plan
	// would overwrite the baseline somebody is measuring against.
	Series domain.Series `json:"series"`
}

// Preview is what an upload produced: the job, the rows and what is wrong.
type Preview struct {
	Job  domain.ImportJob   `json:"job"`
	Rows []domain.ImportRow `json:"rows"`
	// Unmapped and Missing name the columns that did not line up, which is the
	// first thing to check when an import produces nothing.
	Unmapped []string `json:"unmapped,omitempty"`
	Missing  []string `json:"missing,omitempty"`
}

// Stage reads a file through its mapping and holds the result for review.
//
// Nothing is written to the plan here. What the caller gets back is what would
// be written, together with every problem found, and the job stays open until
// somebody commits or cancels it.
func (i *Imports) Stage(ctx context.Context, req StageRequest) (Preview, error) {
	caller := auth.FromContext(ctx)

	mapping, err := i.mappingFor(ctx, req)
	if err != nil {
		return Preview{}, err
	}
	version, season, err := i.planning.writeTarget(ctx, req.VersionID)
	if err != nil {
		return Preview{}, err
	}
	series := req.Series
	if series == "" {
		series = domain.SeriesPlan
	}
	if series != domain.SeriesPlan && series != domain.SeriesActual {
		return Preview{}, fmt.Errorf("%w: %q is not a series; use PLAN or ACTUAL",
			domain.ErrValidation, series)
	}
	// The permission the commit will need is checked now rather than after
	// somebody has uploaded a file and read a preview of it.
	if err := caller.Require(permForKind(mapping.Kind, series)); err != nil {
		return Preview{}, err
	}
	if len(req.Content) == 0 {
		return Preview{}, fmt.Errorf("%w: the file is empty", domain.ErrValidation)
	}

	cells, err := tabular.ReadSheet(req.FileName, req.Content, mapping.Separator(), mapping.Sheet)
	if err != nil {
		return Preview{}, fmt.Errorf("%w: %s", domain.ErrValidation, err)
	}
	read, err := domain.ReadRows(mapping, cells)
	if err != nil {
		return Preview{}, err
	}

	// Codes are resolved here rather than in the domain, which has no
	// repository to look one up in.
	if err := i.resolveCodes(ctx, mapping.Kind, season.FactoryID, read.Rows); err != nil {
		return Preview{}, err
	}
	if err := i.markReplacements(ctx, mapping.Kind, version.ID, series, read.Rows); err != nil {
		return Preview{}, err
	}

	job := domain.ImportJob{
		Kind: mapping.Kind, MappingID: mapping.ID, VersionID: version.ID,
		FactoryID: season.FactoryID, FileName: req.FileName,
		Status: domain.ImportValidated, TotalRows: len(read.Rows),
		Note: string(series),
	}
	for _, r := range read.Rows {
		if r.OK() {
			job.ValidRows++
			continue
		}
		job.ErrorRows++
	}
	for _, h := range read.Missing {
		job.Errors = append(job.Errors, domain.FieldError{
			Field: "columns", Code: "MISSING_COLUMN",
			Message: fmt.Sprintf("the file has no %q column", h),
		})
	}
	for _, h := range read.Unmapped {
		job.Errors = append(job.Errors, domain.FieldError{
			Field: "columns", Code: "UNMAPPED_COLUMN",
			Message: fmt.Sprintf("the file's %q column is not in the mapping and will be ignored", h),
		})
	}

	var saved domain.ImportJob
	err = i.store.InTx(ctx, func(tx store.Store) error {
		var err error
		saved, err = tx.Imports().SaveJob(ctx, job, caller.Username)
		if err != nil {
			return err
		}
		if err := tx.Imports().SaveRows(ctx, saved.ID, read.Rows); err != nil {
			return err
		}
		return writeAudit(ctx, tx, i.now, auditEntry{
			action: "IMPORT_STAGE", entity: "import_job", entityID: saved.ID, after: saved,
			reason: fmt.Sprintf("%s: %d rows, %d with errors", req.FileName,
				saved.TotalRows, saved.ErrorRows),
		})
	})
	if err != nil {
		return Preview{}, err
	}
	return Preview{Job: saved, Rows: read.Rows, Unmapped: read.Unmapped, Missing: read.Missing}, nil
}

func (i *Imports) mappingFor(ctx context.Context, req StageRequest) (domain.ImportMapping, error) {
	switch {
	case req.MappingID != "":
		return i.store.Imports().GetMapping(ctx, req.MappingID)
	case req.MappingCode != "":
		return i.store.Imports().MappingByCode(ctx,
			strings.ToUpper(strings.TrimSpace(req.MappingCode)))
	}
	return domain.ImportMapping{}, fmt.Errorf(
		"%w: the upload must name the mapping that reads it", domain.ErrValidation)
}

// permForKind is the permission the commit will need. An import is not a way of
// posting figures somebody may not post by hand.
func permForKind(kind domain.ImportKind, series domain.Series) string {
	if series != domain.SeriesActual {
		return domain.PermPlanWrite
	}
	switch kind {
	case domain.ImportCane:
		return domain.PermActualCane
	case domain.ImportProduction:
		return domain.PermActualProduce
	case domain.ImportShipment:
		return domain.PermActualShip
	}
	return domain.PermActualStock
}

// ---------------------------------------------------------------------------
// Resolving codes and spotting replacements
// ---------------------------------------------------------------------------

// resolveCodes turns the business keys in a file - product codes, warehouse
// codes - into ids, and fails the row that names something that does not exist.
//
// Every code in the file is looked up once rather than once per row: a season of
// daily rows names the same handful of products 137 times each.
func (i *Imports) resolveCodes(ctx context.Context, kind domain.ImportKind,
	factoryID string, rows []domain.ImportRow,
) error {

	fields := []domain.ImportField{}
	for _, f := range domain.FieldsFor(kind) {
		if f.Type == domain.FieldCode {
			fields = append(fields, f)
		}
	}
	if len(fields) == 0 {
		return nil
	}

	for _, field := range fields {
		wanted := map[string]bool{}
		for _, r := range rows {
			if code := r.Values[field.Name]; code != "" {
				wanted[code] = true
			}
		}
		if len(wanted) == 0 {
			continue
		}

		resolved := map[string]string{}
		for code := range wanted {
			id, err := i.lookup(ctx, field.Entity, code)
			if err != nil {
				if errors.Is(err, domain.ErrNotFound) {
					continue
				}
				return err
			}
			resolved[code] = id
		}

		for idx := range rows {
			code := rows[idx].Values[field.Name]
			if code == "" {
				continue
			}
			id, ok := resolved[code]
			if !ok {
				rows[idx].Errors = append(rows[idx].Errors, domain.FieldError{
					Field: field.Name, Code: "NOT_FOUND",
					Message: fmt.Sprintf("%s %q is not %s in this system",
						field.Label, code, article(field.Entity)),
				})
				continue
			}
			// The id replaces the code under the field's own name plus "Id", so
			// the row keeps the code somebody typed as well as what it resolved
			// to - which is what an error message needs to quote.
			rows[idx].Values[strings.TrimSuffix(field.Name, "Code")+"Id"] = id
		}
	}
	_ = factoryID
	return nil
}

func article(entity string) string {
	switch entity {
	case "product", "packaging", "line", "shift", "channel", "warehouse":
		return "a " + entity
	}
	return "a known " + entity
}

// lookup resolves one business key against the repository that owns it.
func (i *Imports) lookup(ctx context.Context, entity, code string) (string, error) {
	md := i.store.MasterData()
	switch entity {
	case "product":
		p, err := md.Products().GetByCode(ctx, code)
		return p.ID, err
	case "packaging":
		p, err := md.PackagingTypes().GetByCode(ctx, code)
		return p.ID, err
	case "line":
		l, err := md.Lines().GetByCode(ctx, code)
		return l.ID, err
	case "shift":
		s, err := md.Shifts().GetByCode(ctx, code)
		return s.ID, err
	case "channel":
		c, err := md.Channels().GetByCode(ctx, code)
		return c.ID, err
	case "warehouse":
		w, err := md.Warehouses().GetByCode(ctx, code)
		return w.ID, err
	}
	return "", fmt.Errorf("%w: %q is not an entity this system resolves codes against",
		domain.ErrValidation, entity)
}

// markReplacements flags the rows whose key already exists in the plan.
//
// This is not an error - re-importing a corrected sheet is the normal case and
// an upsert is exactly what should happen - but somebody about to overwrite a
// fortnight of recorded figures should be told before they press the button
// rather than after.
func (i *Imports) markReplacements(ctx context.Context, kind domain.ImportKind,
	versionID string, series domain.Series, rows []domain.ImportRow,
) error {

	from, to := domain.BusinessDate(""), domain.BusinessDate("")
	for _, r := range rows {
		d := domain.BusinessDate(r.Values["businessDate"])
		if !d.Valid() {
			continue
		}
		if from == "" || d < from {
			from = d
		}
		if to == "" || d > to {
			to = d
		}
	}
	if from == "" {
		return nil
	}

	f := store.PlanFilter{VersionIDs: []string{versionID}, From: from, To: to, Series: series}
	existing := map[string]bool{}

	switch kind {
	case domain.ImportCane:
		found, err := i.store.Planning().ListCane(ctx, f)
		if err != nil {
			return err
		}
		for _, r := range found {
			existing[key(string(r.BusinessDate), r.ShiftID)] = true
		}
	case domain.ImportProduction:
		found, err := i.store.Planning().ListProducts(ctx, f)
		if err != nil {
			return err
		}
		for _, r := range found {
			existing[key(string(r.BusinessDate), r.ProductID, r.PackagingID, r.LineID, r.ShiftID)] = true
		}
	case domain.ImportShipment:
		found, err := i.store.Planning().ListShipments(ctx, f)
		if err != nil {
			return err
		}
		for _, r := range found {
			existing[key(string(r.BusinessDate), r.ProductID, r.ChannelID, r.WarehouseID)] = true
		}
	case domain.ImportStorage:
		found, err := i.store.Planning().ListStorage(ctx, f)
		if err != nil {
			return err
		}
		for _, r := range found {
			existing[key(string(r.BusinessDate), r.WarehouseID, r.ProductID)] = true
		}
	}

	for idx := range rows {
		rows[idx].Replaces = existing[i.storedKey(kind, rows[idx])]
	}
	return nil
}

// storedKey builds the key of a staged row in the same shape as a stored row's,
// so the two can be compared.
func (i *Imports) storedKey(kind domain.ImportKind, row domain.ImportRow) string {
	parts := make([]string, 0, 5)
	for _, name := range domain.KeyFields(kind) {
		if strings.HasSuffix(name, "Code") {
			parts = append(parts, row.Values[strings.TrimSuffix(name, "Code")+"Id"])
			continue
		}
		parts = append(parts, row.Values[name])
	}
	return key(parts...)
}

func key(parts ...string) string { return strings.Join(parts, "|") }

// ---------------------------------------------------------------------------
// Reading a staged job back
// ---------------------------------------------------------------------------

// ListJobs returns the import history.
func (i *Imports) ListJobs(ctx context.Context, f store.ImportFilter) (store.Page[domain.ImportJob], error) {
	if err := auth.FromContext(ctx).Require(domain.PermPlanRead); err != nil {
		return store.Page[domain.ImportJob]{}, err
	}
	return i.store.Imports().ListJobs(ctx, f)
}

// GetJob returns one job with a page of its staged rows.
func (i *Imports) GetJob(ctx context.Context, id string, errorsOnly bool, skip, top int) (Preview, error) {
	if err := auth.FromContext(ctx).Require(domain.PermPlanRead); err != nil {
		return Preview{}, err
	}
	job, err := i.store.Imports().GetJob(ctx, id)
	if err != nil {
		return Preview{}, err
	}
	page, err := i.store.Imports().Rows(ctx, id, errorsOnly, skip, top)
	if err != nil {
		return Preview{}, err
	}
	return Preview{Job: job, Rows: page.Items}, nil
}

// ---------------------------------------------------------------------------
// Commit
// ---------------------------------------------------------------------------

// CommitRequest applies a staged import.
type CommitRequest struct {
	// Partial commits the sound rows and leaves the rest. The default is all or
	// nothing, which is what section 22 asks for: never post a partially valid
	// document unless the business transaction supports it.
	Partial bool `json:"partial,omitempty"`
}

// CommitResult reports what was written.
type CommitResult struct {
	Job      domain.ImportJob `json:"job"`
	Written  int              `json:"written"`
	Skipped  int              `json:"skipped"`
	Replaced int              `json:"replaced"`
}

// Commit writes the staged rows into the plan.
//
// It goes through the ordinary planning service rather than the repository, so
// an imported row meets the same validation, the same period locking, the same
// authorisation and the same audit trail as a row somebody types into the grid.
// An import that could write what a person could not would be a hole in every
// rule above it.
func (i *Imports) Commit(ctx context.Context, id string, req CommitRequest) (CommitResult, error) {
	caller := auth.FromContext(ctx)

	job, err := i.store.Imports().GetJob(ctx, id)
	if err != nil {
		return CommitResult{}, err
	}
	if !job.IsOpen() {
		return CommitResult{}, fmt.Errorf("%w: this import is %s and cannot be committed again",
			domain.ErrStateTransition, strings.ToLower(job.Status))
	}
	series := domain.Series(job.Note)
	if series == "" {
		series = domain.SeriesPlan
	}
	if err := caller.Require(permForKind(job.Kind, series)); err != nil {
		return CommitResult{}, err
	}
	if job.ErrorRows > 0 && !req.Partial {
		return CommitResult{}, fmt.Errorf(
			"%w: %d of %d rows have errors; fix the file and upload it again, "+
				"or commit the sound rows explicitly",
			domain.ErrValidation, job.ErrorRows, job.TotalRows)
	}
	if !job.CanCommit() {
		return CommitResult{}, fmt.Errorf("%w: no row in this import can be written",
			domain.ErrValidation)
	}

	staged, err := i.store.Imports().Rows(ctx, id, false, 0, tabular.MaxRows)
	if err != nil {
		return CommitResult{}, err
	}

	result := CommitResult{}
	sound := make([]domain.ImportRow, 0, len(staged.Items))
	for _, r := range staged.Items {
		if !r.OK() {
			result.Skipped++
			continue
		}
		if r.Replaces {
			result.Replaced++
		}
		sound = append(sound, r)
	}
	sort.Slice(sound, func(a, b int) bool { return sound[a].RowNo < sound[b].RowNo })

	// The write goes through the planning service, which opens its own
	// transaction and writes its own audit record.
	written, err := i.write(ctx, job, series, sound)
	if err != nil {
		return CommitResult{}, err
	}
	result.Written = written

	now := i.now()
	job.Status, job.CommittedAt, job.CommittedBy = domain.ImportCommitted, &now, caller.Username
	err = i.store.InTx(ctx, func(tx store.Store) error {
		var err error
		result.Job, err = tx.Imports().SaveJob(ctx, job, caller.Username)
		if err != nil {
			return err
		}
		return writeAudit(ctx, tx, i.now, auditEntry{
			action: "IMPORT_COMMIT", entity: "import_job", entityID: job.ID, after: result.Job,
			reason: fmt.Sprintf("%s: %d rows written, %d replaced existing, %d skipped",
				job.FileName, result.Written, result.Replaced, result.Skipped),
		})
	})
	if err != nil {
		return CommitResult{}, err
	}
	return result, nil
}

// write turns staged rows into the daily rows of their kind and hands them to
// the planning service.
func (i *Imports) write(ctx context.Context, job domain.ImportJob, series domain.Series,
	rows []domain.ImportRow,
) (int, error) {

	opts := UpsertOptions{}
	// A field the file did not carry is zero, not a parse error: a sheet that
	// records crushed cane and nothing else is a perfectly ordinary sheet.
	dec := func(r domain.ImportRow, field string) domain.Dec {
		raw := r.Values[field]
		if raw == "" {
			return domain.Zero
		}
		return domain.D(raw)
	}

	switch job.Kind {
	case domain.ImportCane:
		out := make([]domain.DailyCanePlan, 0, len(rows))
		for _, r := range rows {
			out = append(out, domain.DailyCanePlan{
				VersionID: job.VersionID, FactoryID: job.FactoryID,
				BusinessDate: domain.BusinessDate(r.Values["businessDate"]),
				ShiftID:      r.Values["shiftId"], Series: series,
				CaneAvailable: dec(r, "caneAvailable"), CaneDelivered: dec(r, "caneDelivered"),
				CaneAccepted: dec(r, "caneAccepted"), CaneRejected: dec(r, "caneRejected"),
				CaneDiverted: dec(r, "caneDiverted"), CaneCrushed: dec(r, "caneCrushed"),
				CrushRateTPH: dec(r, "crushRateTph"), AvailableHrs: dec(r, "availableHours"),
				StoppageHrs: dec(r, "stoppageHours"),
				ReasonCode:  r.Values["reasonCode"], Note: r.Values["note"],
			})
		}
		res, err := i.planning.UpsertCane(ctx, job.VersionID, out, opts)
		return res.Accepted, err

	case domain.ImportProduction:
		out := make([]domain.DailyProductPlan, 0, len(rows))
		for _, r := range rows {
			out = append(out, domain.DailyProductPlan{
				VersionID: job.VersionID, FactoryID: job.FactoryID,
				BusinessDate: domain.BusinessDate(r.Values["businessDate"]),
				LineID:       r.Values["lineId"], ShiftID: r.Values["shiftId"],
				ProductID: r.Values["productId"], PackagingID: r.Values["packagingId"],
				Series:   series,
				Quantity: dec(r, "quantity"), RemeltInput: dec(r, "remeltInput"),
				ProcessLoss: dec(r, "processLoss"), Rework: dec(r, "rework"),
				Rejected: dec(r, "rejected"), HoldQty: dec(r, "holdQty"),
				ReasonCode: r.Values["reasonCode"], Note: r.Values["note"],
			})
		}
		res, err := i.planning.UpsertProduction(ctx, job.VersionID, out, opts)
		return res.Accepted, err

	case domain.ImportShipment:
		out := make([]domain.DailyShipmentPlan, 0, len(rows))
		for _, r := range rows {
			out = append(out, domain.DailyShipmentPlan{
				VersionID:    job.VersionID,
				BusinessDate: domain.BusinessDate(r.Values["businessDate"]),
				ProductID:    r.Values["productId"], ChannelID: r.Values["channelId"],
				WarehouseID: r.Values["warehouseId"], Series: series,
				Quantity: dec(r, "quantity"), Note: r.Values["note"],
			})
		}
		res, err := i.planning.UpsertShipments(ctx, job.VersionID, out, opts)
		return res.Accepted, err

	case domain.ImportStorage:
		out := make([]domain.DailyStoragePlan, 0, len(rows))
		for _, r := range rows {
			row := domain.DailyStoragePlan{
				VersionID: job.VersionID, WarehouseID: r.Values["warehouseId"],
				ProductID:         r.Values["productId"],
				BusinessDate:      domain.BusinessDate(r.Values["businessDate"]),
				Series:            series,
				ProductionReceipt: dec(r, "productionReceipt"),
				TransferIn:        dec(r, "transferIn"), TransferOut: dec(r, "transferOut"),
				RepackIn: dec(r, "repackIn"), RepackOut: dec(r, "repackOut"),
				RemeltIssue: dec(r, "remeltIssue"), ShipmentQty: dec(r, "shipmentQty"),
				Adjustment: dec(r, "adjustment"), ProcessLoss: dec(r, "processLoss"),
				HoldQty: dec(r, "holdQty"),
			}
			// A count is optional, and an absent one must stay absent: a zero
			// would read as "we counted, and there was nothing there".
			if raw, ok := r.Values["physicalBalance"]; ok && raw != "" {
				counted := domain.D(raw)
				row.PhysicalBalance = &counted
			}
			out = append(out, row)
		}
		res, err := i.planning.UpsertStorage(ctx, job.VersionID, out, opts)
		return res.Accepted, err
	}
	return 0, fmt.Errorf("%w: %q is not a kind of file this system imports",
		domain.ErrValidation, job.Kind)
}

// Cancel abandons a staged import. The job and its rows stay, because what was
// uploaded and refused is part of the record.
func (i *Imports) Cancel(ctx context.Context, id string) (domain.ImportJob, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return domain.ImportJob{}, err
	}
	job, err := i.store.Imports().GetJob(ctx, id)
	if err != nil {
		return domain.ImportJob{}, err
	}
	if !job.IsOpen() {
		return domain.ImportJob{}, fmt.Errorf("%w: this import is %s already",
			domain.ErrStateTransition, strings.ToLower(job.Status))
	}
	job.Status = domain.ImportCancelled

	var saved domain.ImportJob
	err = i.store.InTx(ctx, func(tx store.Store) error {
		var err error
		saved, err = tx.Imports().SaveJob(ctx, job, caller.Username)
		if err != nil {
			return err
		}
		return writeAudit(ctx, tx, i.now, auditEntry{
			action: "IMPORT_CANCEL", entity: "import_job", entityID: job.ID, after: saved,
		})
	})
	return saved, err
}
