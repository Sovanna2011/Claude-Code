package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/sovanna2011/sugarcane-go/backend/internal/database"
	"github.com/sovanna2011/sugarcane-go/backend/internal/domain"
)

// PlanRepository owns the activity plan: the header, its tasks, and the three inputs the generator
// reads — the projection's lines, the activity master with its dependencies, and the calendar.
type PlanRepository struct{ db *database.DB }

func NewPlanRepository(db *database.DB) *PlanRepository { return &PlanRepository{db: db} }

func (r *PlanRepository) q(tx database.Querier) database.Querier {
	if tx != nil {
		return tx
	}
	return r.db.Pool()
}

// ---------------------------------------------------------------- the generator's inputs

// PlanningActivities reads the activity master with each activity's predecessors, in the order the
// engine walks it.
//
// Every dependency dates the schedule, advisory ones included: a dependency says this work follows
// that work, and a programme that ignored half of them would put activities in an order nobody
// intends. What is_blocking governs is what happens when the field departs from the plan — the
// execution module refuses a blocking one and warns about the rest — not how the plan is drawn.
func (r *PlanRepository) PlanningActivities(ctx context.Context, tx database.Querier, companyID int) ([]domain.PlanActivity, error) {
	rows, err := r.q(tx).Query(ctx, `
		SELECT a.id, a.code, a.name, a.category::text, a.applicable_crop_type::text, a.sequence_no,
		       a.standard_start_day_offset, a.standard_capacity_per_day,
		       a.standard_duration_per_ha, a.standard_labour_days_per_ha,
		       a.is_mandatory, a.requires_tractor, a.requires_equipment,
		       a.requires_material, a.requires_labour,
		       COALESCE(json_agg(json_build_object(
		           'dependsOnId', d.depends_on_id, 'lagDays', d.lag_days, 'isBlocking', d.is_blocking)
		           ORDER BY d.depends_on_id) FILTER (WHERE d.id IS NOT NULL), '[]')
		  FROM planting_activity a
		  LEFT JOIN activity_dependency d ON d.activity_id = a.id
		 WHERE a.company_id = $1 AND a.active
		 GROUP BY a.id
		 ORDER BY a.sequence_no, a.id`, companyID)
	if err != nil {
		return nil, fmt.Errorf("planning activities: %w", err)
	}
	defer rows.Close()

	out := []domain.PlanActivity{}
	for rows.Next() {
		var a domain.PlanActivity
		var deps []byte
		if err := rows.Scan(&a.ID, &a.Code, &a.Name, &a.Category, &a.CropType, &a.SequenceNo,
			&a.StartDayOffset, &a.CapacityPerDay, &a.DurationPerHa, &a.LabourDaysPerHa,
			&a.IsMandatory, &a.RequiresTractor, &a.RequiresEquipment,
			&a.RequiresMaterial, &a.RequiresLabour, &deps); err != nil {
			return nil, err
		}
		if err := unmarshalDependencies(deps, &a.DependsOn); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// PlanLines reads the blocks an approved projection covers, as the generator needs them.
func (r *PlanRepository) PlanLines(ctx context.Context, tx database.Querier, projectionID int) ([]domain.PlanLine, error) {
	rows, err := r.q(tx).Query(ctx, `
		SELECT l.id, l.block_id, b.code, l.projected_planting_area_ha,
		       l.planting_type::text, l.planned_planting_start
		  FROM projection_line l
		  JOIN block b ON b.id = l.block_id
		 WHERE l.projection_id = $1
		 ORDER BY l.priority, b.code, l.id`, projectionID)
	if err != nil {
		return nil, fmt.Errorf("projection lines for planning: %w", err)
	}
	defer rows.Close()

	out := []domain.PlanLine{}
	for rows.Next() {
		var l domain.PlanLine
		if err := rows.Scan(&l.ProjectionLineID, &l.BlockID, &l.BlockCode,
			&l.AreaHa, &l.PlantingType, &l.PlantingStart); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------- the plan

const planSelect = `
	SELECT p.id, p.projection_id, pr.projection_no, pr.version, pr.status::text,
	       p.plan_no, p.status::text,
	       p.work_on_saturday, p.work_on_sunday, p.holidays,
	       p.starts_on::text, p.ends_on::text,
	       p.task_count, p.total_area_ha, p.total_working_hours, p.total_labour_days,
	       p.generated_at::text, p.generated_by, p.released_at::text, p.released_by,
	       p.closed_at::text, p.closed_by,
	       p.remark, p.row_version, p.created_at, p.created_by, p.updated_at, p.updated_by
	  FROM activity_plan p
	  JOIN planting_projection pr ON pr.id = p.projection_id`

func scanPlan(row pgx.Row) (domain.ActivityPlan, error) {
	var p domain.ActivityPlan
	var holidays []time.Time
	err := row.Scan(&p.ID, &p.ProjectionID, &p.ProjectionNo, &p.ProjectionRevision, &p.ProjectionStatus,
		&p.PlanNo, &p.Status,
		&p.WorkOnSaturday, &p.WorkOnSunday, &holidays,
		&p.StartsOn, &p.EndsOn,
		&p.TaskCount, &p.TotalAreaHa, &p.TotalWorkingHours, &p.TotalLabourDays,
		&p.GeneratedAt, &p.GeneratedBy, &p.ReleasedAt, &p.ReleasedBy,
		&p.ClosedAt, &p.ClosedBy,
		&p.Remark, &p.Version, &p.CreatedAt, &p.CreatedBy, &p.UpdatedAt, &p.UpdatedBy)
	p.Holidays = []string{}
	for _, h := range holidays {
		p.Holidays = append(p.Holidays, h.Format("2006-01-02"))
	}
	return p, err
}

func (r *PlanRepository) ListPlans(ctx context.Context, f domain.Filter, page domain.Page) ([]domain.ActivityPlan, int, error) {
	b := &builder{}
	if f.ProjectionID != nil {
		b.eq("p.projection_id", *f.ProjectionID)
	}
	if f.SeasonID != nil {
		b.eq("pr.crop_season_id", *f.SeasonID)
	}
	if f.PlanStatus != nil {
		b.where = append(b.where, fmt.Sprintf("p.status = %s::activity_plan_status", b.add(*f.PlanStatus)))
	}
	if land := planLandConditions(b, f); len(land) > 0 {
		b.raw(fmt.Sprintf(`EXISTS (SELECT 1 FROM activity_plan_task t
			JOIN block bk ON bk.id = t.block_id
			JOIN zone z   ON z.id = bk.zone_id
			 WHERE t.plan_id = p.id AND %s)`, strings.Join(land, " AND ")))
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		key := b.add("%" + strings.ToLower(s) + "%")
		b.raw(fmt.Sprintf("(lower(p.plan_no) LIKE %s OR lower(pr.projection_no) LIKE %s)", key, key))
	}

	var total int
	countSQL := `SELECT count(*) FROM activity_plan p
	              JOIN planting_projection pr ON pr.id = p.projection_id` + b.whereSQL()
	if err := r.db.Pool().QueryRow(ctx, countSQL, b.args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count plans: %w", err)
	}

	query := planSelect + b.whereSQL() +
		fmt.Sprintf(" ORDER BY p.starts_on DESC NULLS LAST, p.plan_no LIMIT %s OFFSET %s",
			b.add(page.Size), b.add(page.Offset()))
	rows, err := r.db.Pool().Query(ctx, query, b.args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list plans: %w", err)
	}
	defer rows.Close()

	out := []domain.ActivityPlan{}
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

func planLandConditions(b *builder, f domain.Filter) []string {
	var out []string
	if f.FarmID != nil {
		out = append(out, "z.farm_id = "+b.add(*f.FarmID))
	}
	if f.ZoneID != nil {
		out = append(out, "bk.zone_id = "+b.add(*f.ZoneID))
	}
	if f.BlockID != nil {
		out = append(out, "t.block_id = "+b.add(*f.BlockID))
	}
	if f.ActivityID != nil {
		out = append(out, "t.activity_id = "+b.add(*f.ActivityID))
	}
	return out
}

func (r *PlanRepository) GetPlan(ctx context.Context, tx database.Querier, id int) (domain.ActivityPlan, error) {
	p, err := scanPlan(r.q(tx).QueryRow(ctx, planSelect+" WHERE p.id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ActivityPlan{}, domain.NotFound("Activity plan", id)
	}
	return p, err
}

func (r *PlanRepository) PlanForProjection(ctx context.Context, tx database.Querier, projectionID int) (domain.ActivityPlan, error) {
	p, err := scanPlan(r.q(tx).QueryRow(ctx, planSelect+" WHERE p.projection_id = $1", projectionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ActivityPlan{}, domain.NotFound("Activity plan for projection", projectionID)
	}
	return p, err
}

// Tasks reads a plan's programme, optionally narrowed to one block or activity — the Gantt asks for
// everything, a block's own page asks for its rows only.
func (r *PlanRepository) Tasks(ctx context.Context, tx database.Querier, planID int, f domain.Filter) ([]domain.PlanTaskRow, error) {
	b := &builder{}
	b.eq("t.plan_id", planID)
	if f.BlockID != nil {
		b.eq("t.block_id", *f.BlockID)
	}
	if f.ActivityID != nil {
		b.eq("t.activity_id", *f.ActivityID)
	}
	if f.ZoneID != nil {
		b.eq("bk.zone_id", *f.ZoneID)
	}
	if f.FarmID != nil {
		b.eq("z.farm_id", *f.FarmID)
	}
	if f.ActivityCategory != nil {
		b.where = append(b.where, fmt.Sprintf("a.category = %s::activity_category", b.add(*f.ActivityCategory)))
	}

	rows, err := r.q(tx).Query(ctx, `
		SELECT t.id, t.plan_id, t.projection_line_id, t.block_id, bk.code, bk.name,
		       z.id, z.code, f.id, f.code,
		       t.activity_id, a.code, a.name, a.category::text, t.sequence_no,
		       t.planned_area_ha, t.planned_start_date::text, t.planned_end_date::text,
		       t.duration_days, t.daily_target_ha,
		       t.planned_working_hours, t.required_labour_days, t.required_workers,
		       a.requires_tractor, a.requires_equipment, a.requires_material, a.requires_labour,
		       t.status::text, t.completion_percent,
		       t.actual_start_date::text, t.actual_end_date::text, t.actual_area_ha, t.remark
		  FROM activity_plan_task t
		  JOIN block bk           ON bk.id = t.block_id
		  JOIN zone z             ON z.id = bk.zone_id
		  JOIN farm f             ON f.id = z.farm_id
		  JOIN planting_activity a ON a.id = t.activity_id`+b.whereSQL()+`
		 ORDER BY bk.code, t.sequence_no, t.id`, b.args...)
	if err != nil {
		return nil, fmt.Errorf("plan tasks: %w", err)
	}
	defer rows.Close()

	out := []domain.PlanTaskRow{}
	for rows.Next() {
		var t domain.PlanTaskRow
		if err := rows.Scan(&t.ID, &t.PlanID, &t.ProjectionLineID, &t.BlockID, &t.BlockCode, &t.BlockName,
			&t.ZoneID, &t.ZoneCode, &t.FarmID, &t.FarmCode,
			&t.ActivityID, &t.ActivityCode, &t.ActivityName, &t.Category, &t.SequenceNo,
			&t.PlannedAreaHa, &t.PlannedStartDate, &t.PlannedEndDate,
			&t.DurationDays, &t.DailyTargetHa,
			&t.PlannedWorkingHours, &t.RequiredLabourDays, &t.RequiredWorkers,
			&t.RequiresTractor, &t.RequiresEquipment, &t.RequiresMaterial, &t.RequiresLabour,
			&t.Status, &t.CompletionPercent,
			&t.ActualStartDate, &t.ActualEndDate, &t.ActualAreaHa, &t.Remark); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------- writing

func (r *PlanRepository) CreatePlan(ctx context.Context, tx database.Querier,
	projectionID int, planNo string, cal domain.WorkingCalendar, remark *string) (int, error) {

	var id int
	err := tx.QueryRow(ctx, `
		INSERT INTO activity_plan(projection_id, plan_no, work_on_saturday, work_on_sunday, holidays, remark)
		VALUES ($1, $2, $3, $4, $5::date[], $6) RETURNING id`,
		projectionID, planNo, cal.WorkOnSaturday, cal.WorkOnSunday, holidayList(cal), remark).Scan(&id)
	return id, mapPlanError(err, planNo)
}

func (r *PlanRepository) UpdateCalendar(ctx context.Context, tx database.Querier,
	id int, cal domain.WorkingCalendar, rowVersion int) error {

	tag, err := tx.Exec(ctx, `
		UPDATE activity_plan
		   SET work_on_saturday = $2, work_on_sunday = $3, holidays = $4::date[],
		       row_version = row_version + 1
		 WHERE id = $1 AND row_version = $5`,
		id, cal.WorkOnSaturday, cal.WorkOnSunday, holidayList(cal), rowVersion)
	if err != nil {
		return mapPlanError(err, "calendar")
	}
	if tag.RowsAffected() == 0 {
		return explainMiss(ctx, r.q(tx), "activity_plan", "Activity plan", id)
	}
	return nil
}

// ReplaceTasks writes a freshly generated programme. Regeneration is a replacement rather than a
// merge: the engine is deterministic, so anything still in the table that the engine did not
// produce is a leftover from a previous shape of the projection.
func (r *PlanRepository) ReplaceTasks(ctx context.Context, tx database.Querier, planID int, tasks []domain.PlanTask) error {
	if _, err := tx.Exec(ctx, `DELETE FROM activity_plan_task WHERE plan_id = $1`, planID); err != nil {
		return fmt.Errorf("clear plan tasks: %w", err)
	}

	for _, t := range tasks {
		_, err := tx.Exec(ctx, `
			INSERT INTO activity_plan_task(plan_id, projection_line_id, block_id, activity_id, sequence_no,
			        planned_area_ha, planned_start_date, planned_end_date, duration_days, daily_target_ha,
			        planned_working_hours, required_labour_days, required_workers)
			VALUES ($1, $2, $3, $4, $5, $6, $7::date, $8::date, $9, $10, $11, $12, $13)`,
			planID, t.ProjectionLineID, t.BlockID, t.ActivityID, t.SequenceNo,
			t.PlannedAreaHa, t.PlannedStartDate.Format("2006-01-02"), t.PlannedEndDate.Format("2006-01-02"),
			t.DurationDays, t.DailyTargetHa,
			t.PlannedWorkingHours, t.RequiredLabourDays, t.RequiredWorkers)
		if err != nil {
			return mapPlanError(err, t.ActivityCode+" on "+t.BlockCode)
		}
	}

	_, err := tx.Exec(ctx, `
		UPDATE activity_plan SET generated_at = now(), generated_by = COALESCE(
		    NULLIF(current_setting('app.actor', true), ''), 'system'), row_version = row_version + 1
		 WHERE id = $1`, planID)
	return err
}

// SetStatus moves a plan through Draft → Released → Closed, stamping who did it.
func (r *PlanRepository) SetStatus(ctx context.Context, tx database.Querier,
	id int, to string, actor string, rowVersion int) error {

	// The arguments are numbered as they are appended. Writing the indexes by hand left $3 unused
	// when reopening, and PostgreSQL will not bind a statement with a hole in its parameters.
	args := []any{id, to}
	sets := []string{"status = $2::activity_plan_status", "row_version = row_version + 1"}

	switch to {
	case domain.PlanReleased:
		args = append(args, actor)
		sets = append(sets, "released_at = now()", fmt.Sprintf("released_by = $%d", len(args)))
	case domain.PlanClosed:
		args = append(args, actor)
		sets = append(sets, "closed_at = now()", fmt.Sprintf("closed_by = $%d", len(args)))
	default:
		// Reopening clears the stamps of the states it has come back from.
		sets = append(sets, "released_at = NULL", "released_by = NULL", "closed_at = NULL", "closed_by = NULL")
	}

	args = append(args, rowVersion)
	query := fmt.Sprintf("UPDATE activity_plan SET %s WHERE id = $1 AND row_version = $%d",
		strings.Join(sets, ", "), len(args))
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return mapPlanError(err, to)
	}
	if tag.RowsAffected() == 0 {
		return explainMiss(ctx, r.q(tx), "activity_plan", "Activity plan", id)
	}
	return nil
}

func (r *PlanRepository) DeletePlan(ctx context.Context, tx database.Querier, id int) error {
	tag, err := tx.Exec(ctx, `DELETE FROM activity_plan WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.NotFound("Activity plan", id)
	}
	return nil
}

// NextPlanNo numbers a plan after the projection it came from, which is how a planner refers to it.
func (r *PlanRepository) NextPlanNo(ctx context.Context, tx database.Querier, projectionID int) (string, error) {
	var projectionNo string
	var revision int
	err := r.q(tx).QueryRow(ctx,
		`SELECT projection_no, version FROM planting_projection WHERE id = $1`, projectionID).
		Scan(&projectionNo, &revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.NotFound("Planting projection", projectionID)
	}
	return fmt.Sprintf("PLAN-%s-v%d", projectionNo, revision), err
}

func holidayList(cal domain.WorkingCalendar) []string {
	out := make([]string, 0, len(cal.Holidays))
	for d := range cal.Holidays {
		out = append(out, d)
	}
	sortStrings(out)
	return out
}

func mapPlanError(err error, key string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "activity_plan_projection_id_key"):
		return domain.Conflict("PLAN_EXISTS",
			"This projection already has an activity plan. Regenerate that one rather than making a second.")
	case strings.Contains(msg, "activity_plan_plan_no_key"):
		return domain.Conflict("DUPLICATE_CODE", fmt.Sprintf("Plan %q already exists.", key))
	case strings.Contains(msg, "task_dates_ordered"):
		return domain.Invalid([]domain.FieldError{{
			Field: "plannedEndDate", Message: "A task cannot end before it starts."}})
	case strings.Contains(msg, "task_duration_positive"):
		return domain.Invalid([]domain.FieldError{{
			Field: "durationDays", Message: "A task occupies at least one working day."}})
	case strings.Contains(msg, "task_area_positive"):
		return domain.Invalid([]domain.FieldError{{
			Field: "plannedAreaHa", Message: "A task must have an area to work."}})
	case strings.Contains(msg, "plan_window"):
		return domain.Invalid([]domain.FieldError{{
			Field: "endsOn", Message: "The plan cannot end before it starts."}})
	}
	return mapWriteError(err, "Activity plan", key)
}
