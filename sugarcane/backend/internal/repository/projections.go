package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/sovanna2011/sugarcane-go/backend/internal/database"
	"github.com/sovanna2011/sugarcane-go/backend/internal/domain"
)

// ProjectionRepository owns the planting projections: the header, its lines, the approval trail and
// the revision chain. The derived totals are the database's own work — nothing here writes them.
type ProjectionRepository struct{ db *database.DB }

func NewProjectionRepository(db *database.DB) *ProjectionRepository {
	return &ProjectionRepository{db: db}
}

func (r *ProjectionRepository) q(tx database.Querier) database.Querier {
	if tx != nil {
		return tx
	}
	return r.db.Pool()
}

const projectionSelect = `
	SELECT p.id, p.company_id, c.name, p.plantation_id, pl.name,
	       p.crop_season_id, s.code, s.crop_year,
	       p.projection_no, p.version, p.supersedes_id,
	       (SELECT n.id FROM planting_projection n WHERE n.supersedes_id = p.id ORDER BY n.version LIMIT 1),
	       p.is_current,
	       p.projection_date::text, p.planning_start::text, p.planning_end::text,
	       p.total_projected_area_ha, p.total_harvestable_area_ha, p.total_expected_tons, p.line_count,
	       p.status::text, p.prepared_by,
	       p.submitted_by, p.submitted_at::text, p.reviewed_by, p.reviewed_at::text,
	       p.approved_by, p.approved_at::text, p.rejected_by, p.rejected_at::text,
	       p.rejection_reason, p.revision_reason, p.closed_by, p.closed_at::text,
	       p.remark, p.row_version, p.created_at, p.created_by, p.updated_at, p.updated_by
	  FROM planting_projection p
	  JOIN company c     ON c.id = p.company_id
	  JOIN plantation pl ON pl.id = p.plantation_id
	  JOIN crop_season s ON s.id = p.crop_season_id`

func scanProjection(row pgx.Row) (domain.Projection, error) {
	var p domain.Projection
	err := row.Scan(&p.ID, &p.CompanyID, &p.CompanyName, &p.PlantationID, &p.PlantationName,
		&p.CropSeasonID, &p.CropSeasonCode, &p.CropYear,
		&p.ProjectionNo, &p.Revision, &p.SupersedesID, &p.SupersededBy, &p.IsCurrent,
		&p.ProjectionDate, &p.PlanningStart, &p.PlanningEnd,
		&p.TotalProjectedAreaHa, &p.TotalHarvestableAreaHa, &p.TotalExpectedTons, &p.LineCount,
		&p.Status, &p.PreparedBy,
		&p.SubmittedBy, &p.SubmittedAt, &p.ReviewedBy, &p.ReviewedAt,
		&p.ApprovedBy, &p.ApprovedAt, &p.RejectedBy, &p.RejectedAt,
		&p.RejectionReason, &p.RevisionReason, &p.ClosedBy, &p.ClosedAt,
		&p.Remark, &p.Version, &p.CreatedAt, &p.CreatedBy, &p.UpdatedAt, &p.UpdatedBy)
	return p, err
}

func (r *ProjectionRepository) ListProjections(ctx context.Context, f domain.Filter, page domain.Page) ([]domain.Projection, int, error) {
	b := &builder{}
	if f.CompanyID != nil {
		b.eq("p.company_id", *f.CompanyID)
	}
	if f.PlantationID != nil {
		b.eq("p.plantation_id", *f.PlantationID)
	}
	if f.SeasonID != nil {
		b.eq("p.crop_season_id", *f.SeasonID)
	}
	if f.CropYear != nil {
		b.eq("s.crop_year", *f.CropYear)
	}
	if f.ProjectionStatus != nil {
		b.where = append(b.where, fmt.Sprintf("p.status = %s::projection_status", b.add(*f.ProjectionStatus)))
	}
	if f.CurrentOnly {
		b.raw("p.is_current")
	}
	// Filtering a projection by land means "which plans touch this ground?" — the natural question
	// when standing on a farm, a zone or a block. All three narrow the same subquery over the
	// lines rather than one subquery each, so asking for a zone and a block is a single pass that
	// answers with the plans matching both, not two independent scans.
	if land := landConditions(b, f); len(land) > 0 {
		b.raw(fmt.Sprintf(`EXISTS (SELECT 1 FROM projection_line l
			JOIN block bk ON bk.id = l.block_id
			JOIN zone z   ON z.id = bk.zone_id
			 WHERE l.projection_id = p.id AND %s)`, strings.Join(land, " AND ")))
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		key := b.add("%" + strings.ToLower(s) + "%")
		b.raw(fmt.Sprintf("(lower(p.projection_no) LIKE %s OR lower(COALESCE(p.remark, '')) LIKE %s)", key, key))
	}

	countSQL := `SELECT count(*) FROM planting_projection p JOIN crop_season s ON s.id = p.crop_season_id` + b.whereSQL()
	var total int
	if err := r.db.Pool().QueryRow(ctx, countSQL, b.args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count projections: %w", err)
	}

	query := projectionSelect + b.whereSQL() +
		fmt.Sprintf(" ORDER BY p.projection_date DESC, p.projection_no, p.version DESC LIMIT %s OFFSET %s",
			b.add(page.Size), b.add(page.Offset()))
	rows, err := r.db.Pool().Query(ctx, query, b.args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list projections: %w", err)
	}
	defer rows.Close()

	out := []domain.Projection{}
	for rows.Next() {
		p, err := scanProjection(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

// landConditions renders the farm, zone and block filters against the aliases the lines subquery
// above establishes: l for the line, bk for its block, z for the block's zone.
func landConditions(b *builder, f domain.Filter) []string {
	var out []string
	if f.FarmID != nil {
		out = append(out, "z.farm_id = "+b.add(*f.FarmID))
	}
	if f.ZoneID != nil {
		out = append(out, "bk.zone_id = "+b.add(*f.ZoneID))
	}
	if f.BlockID != nil {
		out = append(out, "l.block_id = "+b.add(*f.BlockID))
	}
	return out
}

func (r *ProjectionRepository) GetProjection(ctx context.Context, tx database.Querier, id int) (domain.Projection, error) {
	p, err := scanProjection(r.q(tx).QueryRow(ctx, projectionSelect+" WHERE p.id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Projection{}, domain.NotFound("Planting projection", id)
	}
	return p, err
}

// ---------------------------------------------------------------- lines

const lineSelect = `
	SELECT l.id, l.projection_id, l.block_id, b.code, b.name,
	       z.id, z.code, f.id, f.code, f.name,
	       l.cane_variety_id, v.code, l.planting_type::text,
	       l.available_area_ha, l.projected_planting_area_ha,
	       l.planned_planting_start::text, l.planned_planting_end::text, l.expected_harvest_date::text,
	       l.expected_yield_per_ha, l.expected_loss_percent,
	       l.harvestable_area_ha, l.expected_production_tons,
	       COALESCE(v.seed_rate_per_ha, 0), l.priority, l.remark,
	       b.latitude, b.longitude
	  FROM projection_line l
	  JOIN block b        ON b.id = l.block_id
	  JOIN zone z         ON z.id = b.zone_id
	  JOIN farm f         ON f.id = z.farm_id
	  JOIN cane_variety v ON v.id = l.cane_variety_id`

func scanLine(row pgx.Row) (domain.ProjectionLine, error) {
	var l domain.ProjectionLine
	var seedRate float64
	var lat, lng *float64
	err := row.Scan(&l.ID, &l.ProjectionID, &l.BlockID, &l.BlockCode, &l.BlockName,
		&l.ZoneID, &l.ZoneCode, &l.FarmID, &l.FarmCode, &l.FarmName,
		&l.CaneVarietyID, &l.CaneVarietyCode, &l.PlantingType,
		&l.AvailableAreaHa, &l.ProjectedPlantingAreaHa,
		&l.PlannedPlantingStart, &l.PlannedPlantingEnd, &l.ExpectedHarvestDate,
		&l.ExpectedYieldPerHa, &l.ExpectedLossPct,
		&l.HarvestableAreaHa, &l.ExpectedProductionTons,
		&seedRate, &l.Priority, &l.Remark, &lat, &lng)
	if err != nil {
		return l, err
	}
	// Seed cane is only cut for a fresh planting; a ratoon crop regrows from the stubble.
	if l.PlantingType == "NewPlanting" {
		l.RequiredSeedCaneTons = domain.RequiredSeedCane(l.ProjectedPlantingAreaHa, seedRate)
	}
	l.MapURL = domain.GoogleMapsURL(lat, lng)
	return l, nil
}

func (r *ProjectionRepository) Lines(ctx context.Context, tx database.Querier, projectionID int) ([]domain.ProjectionLine, error) {
	rows, err := r.q(tx).Query(ctx, lineSelect+
		" WHERE l.projection_id = $1 ORDER BY l.priority, f.code, z.code, b.code", projectionID)
	if err != nil {
		return nil, fmt.Errorf("projection lines: %w", err)
	}
	defer rows.Close()

	out := []domain.ProjectionLine{}
	for rows.Next() {
		l, err := scanLine(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *ProjectionRepository) GetLine(ctx context.Context, tx database.Querier, id int) (domain.ProjectionLine, error) {
	l, err := scanLine(r.q(tx).QueryRow(ctx, lineSelect+" WHERE l.id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ProjectionLine{}, domain.NotFound("Projection line", id)
	}
	return l, err
}

// BlockFacts is what the server needs to write a line without trusting the caller for it: the
// block's plantable area, which becomes the line's snapshot, and its code for the error messages.
type BlockFacts struct {
	Code        string
	Name        string
	PlantableHa float64
}

func (r *ProjectionRepository) BlockFacts(ctx context.Context, tx database.Querier, blockID int) (BlockFacts, error) {
	var f BlockFacts
	err := r.q(tx).QueryRow(ctx,
		`SELECT code, name, plantable_area_ha FROM block WHERE id = $1 AND active`, blockID).
		Scan(&f.Code, &f.Name, &f.PlantableHa)
	if errors.Is(err, pgx.ErrNoRows) {
		return f, domain.NotFound("Block", blockID)
	}
	return f, err
}

// VarietyDefaults supplies the yield and loss a line inherits when the caller does not override
// them, and the growing period that dates the expected harvest.
type VarietyDefaults struct {
	Code                string
	YieldPerHa          float64
	LossPercent         float64
	GrowingPeriodMonths int
}

func (r *ProjectionRepository) VarietyDefaults(ctx context.Context, tx database.Querier, id int) (VarietyDefaults, error) {
	var v VarietyDefaults
	err := r.q(tx).QueryRow(ctx,
		`SELECT code, expected_yield_per_ha, expected_loss_percent, growing_period_months
		   FROM cane_variety WHERE id = $1 AND active`, id).
		Scan(&v.Code, &v.YieldPerHa, &v.LossPercent, &v.GrowingPeriodMonths)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, domain.NotFound("Cane variety", id)
	}
	return v, err
}

// SeasonWindow is the planting window a line's dates must fall inside, when the season declares one.
func (r *ProjectionRepository) SeasonWindow(ctx context.Context, tx database.Querier, id int) (code string, start, end *string, err error) {
	err = r.q(tx).QueryRow(ctx,
		`SELECT code, planting_window_start::text, planting_window_end::text FROM crop_season WHERE id = $1`, id).
		Scan(&code, &start, &end)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, nil, domain.NotFound("Growing season", id)
	}
	return code, start, end, err
}

// ---------------------------------------------------------------- writing

func (r *ProjectionRepository) SaveProjection(ctx context.Context, tx database.Querier, id *int, in domain.ProjectionInput, actor string) (int, error) {
	if id == nil {
		var newID int
		err := tx.QueryRow(ctx, `
			INSERT INTO planting_projection(company_id, plantation_id, crop_season_id, projection_no,
			        projection_date, planning_start, planning_end, prepared_by, remark, created_by, updated_by)
			VALUES ($1, $2, $3, $4, $5::date, $6::date, $7::date, $8, $9, $10, $10)
			RETURNING id`,
			in.CompanyID, in.PlantationID, in.CropSeasonID, in.ProjectionNo,
			in.ProjectionDate, in.PlanningStart, in.PlanningEnd, in.PreparedBy, in.Remark, actor).Scan(&newID)
		return newID, mapProjectionError(err, in.ProjectionNo)
	}

	tag, err := tx.Exec(ctx, `
		UPDATE planting_projection
		   SET crop_season_id = $2, projection_no = $3, projection_date = $4::date,
		       planning_start = $5::date, planning_end = $6::date,
		       prepared_by = $7, remark = $8, row_version = row_version + 1,
		       updated_at = now(), updated_by = $9
		 WHERE id = $1 AND row_version = $10`,
		*id, in.CropSeasonID, in.ProjectionNo, in.ProjectionDate,
		in.PlanningStart, in.PlanningEnd, in.PreparedBy, in.Remark, actor, in.Version)
	if err != nil {
		return 0, mapProjectionError(err, in.ProjectionNo)
	}
	if tag.RowsAffected() == 0 {
		return 0, explainMiss(ctx, r.q(tx), "planting_projection", "Planting projection", *id)
	}
	return *id, nil
}

// LineValues is a line as the server has settled it — the caller's input with the snapshot, the
// variety defaults and the derived harvest date already applied.
type LineValues struct {
	BlockID       int
	CaneVarietyID int
	PlantingType  string
	AvailableHa   float64
	ProjectedHa   float64
	Start, End    string
	HarvestDate   *string
	YieldPerHa    float64
	LossPercent   float64
	Priority      int
	Remark        *string
}

func (r *ProjectionRepository) SaveLine(ctx context.Context, tx database.Querier, projectionID int, lineID *int, v LineValues) (int, error) {
	if lineID == nil {
		var newID int
		err := tx.QueryRow(ctx, `
			INSERT INTO projection_line(projection_id, block_id, cane_variety_id, planting_type,
			        available_area_ha, projected_planting_area_ha,
			        planned_planting_start, planned_planting_end, expected_harvest_date,
			        expected_yield_per_ha, expected_loss_percent, priority, remark)
			VALUES ($1, $2, $3, $4::planting_type, $5, $6, $7::date, $8::date, $9::date, $10, $11, $12, $13)
			RETURNING id`,
			projectionID, v.BlockID, v.CaneVarietyID, v.PlantingType,
			v.AvailableHa, v.ProjectedHa, v.Start, v.End, v.HarvestDate,
			v.YieldPerHa, v.LossPercent, v.Priority, v.Remark).Scan(&newID)
		return newID, mapProjectionError(err, "line")
	}

	tag, err := tx.Exec(ctx, `
		UPDATE projection_line
		   SET block_id = $3, cane_variety_id = $4, planting_type = $5::planting_type,
		       available_area_ha = $6, projected_planting_area_ha = $7,
		       planned_planting_start = $8::date, planned_planting_end = $9::date,
		       expected_harvest_date = $10::date,
		       expected_yield_per_ha = $11, expected_loss_percent = $12, priority = $13, remark = $14
		 WHERE id = $1 AND projection_id = $2`,
		*lineID, projectionID, v.BlockID, v.CaneVarietyID, v.PlantingType,
		v.AvailableHa, v.ProjectedHa, v.Start, v.End, v.HarvestDate,
		v.YieldPerHa, v.LossPercent, v.Priority, v.Remark)
	if err != nil {
		return 0, mapProjectionError(err, "line")
	}
	if tag.RowsAffected() == 0 {
		return 0, domain.NotFound("Projection line", *lineID)
	}
	return *lineID, nil
}

func (r *ProjectionRepository) DeleteLine(ctx context.Context, tx database.Querier, projectionID, lineID int) error {
	tag, err := tx.Exec(ctx,
		`DELETE FROM projection_line WHERE id = $1 AND projection_id = $2`, lineID, projectionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.NotFound("Projection line", lineID)
	}
	return nil
}

// ---------------------------------------------------------------- the workflow

// stampFor is the column pair each action writes. Approve stamps approved_by and approved_at, and
// nothing else touches those columns, so the header always says who decided and when.
var stampFor = map[string]struct{ by, at string }{
	domain.ActionSubmit:  {"submitted_by", "submitted_at"},
	domain.ActionReview:  {"reviewed_by", "reviewed_at"},
	domain.ActionApprove: {"approved_by", "approved_at"},
	domain.ActionReject:  {"rejected_by", "rejected_at"},
	domain.ActionClose:   {"closed_by", "closed_at"},
}

// ApplyTransition moves the status, stamps the action and writes the trail row, all under the
// caller's row version so two managers deciding at once cannot both win.
func (r *ProjectionRepository) ApplyTransition(ctx context.Context, tx database.Querier,
	id int, from, to, action, actor string, comments *string, rowVersion int) error {

	sets := []string{"status = $2::projection_status", "row_version = row_version + 1",
		"updated_at = now()", "updated_by = $3"}
	args := []any{id, to, actor}

	if stamp, ok := stampFor[action]; ok {
		sets = append(sets, fmt.Sprintf("%s = $3", stamp.by), fmt.Sprintf("%s = now()", stamp.at))
	}
	if action == domain.ActionReject {
		args = append(args, comments)
		sets = append(sets, fmt.Sprintf("rejection_reason = $%d", len(args)))
	}
	if action == domain.ActionRevise {
		args = append(args, comments)
		sets = append(sets, fmt.Sprintf("revision_reason = $%d", len(args)), "is_current = false")
	}

	args = append(args, rowVersion)
	query := fmt.Sprintf("UPDATE planting_projection SET %s WHERE id = $1 AND row_version = $%d",
		strings.Join(sets, ", "), len(args))

	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return mapProjectionError(err, action)
	}
	if tag.RowsAffected() == 0 {
		return explainMiss(ctx, r.q(tx), "planting_projection", "Planting projection", id)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO projection_approval(projection_id, action, from_status, to_status, actor, comments)
		VALUES ($1, $2, $3::projection_status, $4::projection_status, $5, $6)`,
		id, action, from, to, actor, comments)
	return err
}

// AssertNoCommittedOverlap asks the database whether approving this plan would double-book a block.
// The rule lives in SQL so a plan approved by any route is checked, and it is called here rather
// than by a trigger because it is a rule about approving, not about writing a row.
func (r *ProjectionRepository) AssertNoCommittedOverlap(ctx context.Context, tx database.Querier, id int) error {
	_, err := tx.Exec(ctx, `SELECT assert_no_committed_overlap($1)`, id)
	return mapProjectionError(err, "approve")
}

// CreateRevision opens the next version of a decided plan: a fresh Draft carrying the same header
// and a copy of every line, pointing back at the one it replaces.
func (r *ProjectionRepository) CreateRevision(ctx context.Context, tx database.Querier, id int, actor string) (int, error) {
	var newID int
	err := tx.QueryRow(ctx, `
		INSERT INTO planting_projection(company_id, plantation_id, crop_season_id, projection_no,
		        version, supersedes_id, is_current, projection_date, planning_start, planning_end,
		        status, prepared_by, remark, created_by, updated_by)
		SELECT company_id, plantation_id, crop_season_id, projection_no,
		       (SELECT COALESCE(max(version), 0) + 1 FROM planting_projection sib
		         WHERE sib.company_id = p.company_id AND sib.projection_no = p.projection_no),
		       p.id, true, projection_date, planning_start, planning_end,
		       'Draft', prepared_by, remark, $2, $2
		  FROM planting_projection p WHERE p.id = $1
		RETURNING id`, id, actor).Scan(&newID)
	if err != nil {
		return 0, mapProjectionError(err, "revision")
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO projection_line(projection_id, block_id, cane_variety_id, planting_type,
		        available_area_ha, projected_planting_area_ha,
		        planned_planting_start, planned_planting_end, expected_harvest_date,
		        expected_yield_per_ha, expected_loss_percent, priority, remark)
		SELECT $2, block_id, cane_variety_id, planting_type,
		       available_area_ha, projected_planting_area_ha,
		       planned_planting_start, planned_planting_end, expected_harvest_date,
		       expected_yield_per_ha, expected_loss_percent, priority, remark
		  FROM projection_line WHERE projection_id = $1`, id, newID)
	return newID, mapProjectionError(err, "revision")
}

func (r *ProjectionRepository) History(ctx context.Context, tx database.Querier, projectionID int) ([]domain.ProjectionAction, error) {
	rows, err := r.q(tx).Query(ctx, `
		SELECT id, action, from_status::text, to_status::text, actor, at::text, comments
		  FROM projection_approval WHERE projection_id = $1 ORDER BY at, id`, projectionID)
	if err != nil {
		return nil, fmt.Errorf("projection history: %w", err)
	}
	defer rows.Close()

	out := []domain.ProjectionAction{}
	for rows.Next() {
		var a domain.ProjectionAction
		if err := rows.Scan(&a.ID, &a.Action, &a.FromStatus, &a.ToStatus, &a.Actor, &a.At, &a.Comments); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CommittedAreaOnBlock is the area this projection already has on a block, ignoring one line — the
// one being replaced. It lets the service check the cross-row rule before the write, so the caller
// gets a message naming the block rather than the deferred trigger's exception at commit.
func (r *ProjectionRepository) CommittedAreaOnBlock(ctx context.Context, tx database.Querier,
	projectionID, blockID int, excludeLineID *int) (float64, error) {

	var total float64
	err := r.q(tx).QueryRow(ctx, `
		SELECT COALESCE(SUM(projected_planting_area_ha), 0) FROM projection_line
		 WHERE projection_id = $1 AND block_id = $2 AND ($3::int IS NULL OR id <> $3)`,
		projectionID, blockID, excludeLineID).Scan(&total)
	return total, err
}

// TranslateProjectionError maps an error that surfaced from COMMIT rather than from the statement
// that caused it. The deferred constraint triggers fire at commit, so their exceptions escape the
// repository entirely and would otherwise reach the caller as a 500.
func TranslateProjectionError(err error) error {
	if err == nil {
		return nil
	}
	// Something already translated is left alone: re-mapping a domain error would lose its fields.
	if _, ok := domain.AsError(err); ok {
		return err
	}
	return mapProjectionError(err, "projection")
}

// mapProjectionError turns this module's constraint names and RAISE messages into refusals a
// planner can act on, and leaves anything it does not recognise to the shared mapping.
func mapProjectionError(err error, key string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "AREA_EXCEEDS_BLOCK"):
		return domain.Invalid([]domain.FieldError{{
			Field: "projectedPlantingAreaHa", Message: afterMarker(msg, "AREA_EXCEEDS_BLOCK: ")}})
	case strings.Contains(msg, "BLOCK_ALREADY_COMMITTED"):
		return domain.Conflict("BLOCK_ALREADY_COMMITTED", afterMarker(msg, "BLOCK_ALREADY_COMMITTED: "))
	case strings.Contains(msg, "line_area_within_block"):
		return domain.Invalid([]domain.FieldError{{
			Field:   "projectedPlantingAreaHa",
			Message: "The projected area cannot exceed the block's plantable area."}})
	case strings.Contains(msg, "line_area_positive"):
		return domain.Invalid([]domain.FieldError{{
			Field: "projectedPlantingAreaHa", Message: "The projected area must be greater than zero."}})
	case strings.Contains(msg, "line_dates_ordered"):
		return domain.Invalid([]domain.FieldError{{
			Field: "plannedPlantingEnd", Message: "Planting cannot end before it starts."}})
	case strings.Contains(msg, "line_loss_range"):
		return domain.Invalid([]domain.FieldError{{
			Field: "expectedLossPercent", Message: "The expected loss must be between 0 and 100 per cent."}})
	case strings.Contains(msg, "line_yield_not_negative"):
		return domain.Invalid([]domain.FieldError{{
			Field: "expectedYieldPerHa", Message: "The expected yield cannot be negative."}})
	case strings.Contains(msg, "line_priority_range"):
		return domain.Invalid([]domain.FieldError{{
			Field: "priority", Message: "Priority runs from 1 (first) to 9 (last)."}})
	case strings.Contains(msg, "projection_planning_window"):
		return domain.Invalid([]domain.FieldError{{
			Field: "planningEnd", Message: "The planning window cannot end before it starts."}})
	case strings.Contains(msg, "projection_rejection_has_a_reason"):
		return domain.Invalid([]domain.FieldError{{
			Field: "comments", Message: "Say why the plan is being rejected."}})
	case strings.Contains(msg, "projection_line_projection_id_block_id_planting_type_key"):
		return domain.Conflict("DUPLICATE_LINE",
			"That block already has a line of this planting type on this projection; change the existing line instead.")
	case strings.Contains(msg, "planting_projection_company_id_projection_no_version_key"):
		return domain.Conflict("DUPLICATE_CODE",
			fmt.Sprintf("Projection %q already has a version with this number.", key))
	}
	return mapWriteError(err, "Planting projection", key)
}
