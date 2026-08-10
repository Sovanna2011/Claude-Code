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

// ActivityRepository owns the growing seasons, the cane varieties and the planting activity master
// with its dependency chain — the master data the planning engines read before they can turn an
// area into a schedule.
type ActivityRepository struct{ db *database.DB }

func NewActivityRepository(db *database.DB) *ActivityRepository { return &ActivityRepository{db: db} }

func (r *ActivityRepository) q(tx database.Querier) database.Querier {
	if tx != nil {
		return tx
	}
	return r.db.Pool()
}

// ---------------------------------------------------------------- seasons

const seasonSelect = `
	SELECT id, code, name, crop_year, starts_on::text, ends_on::text,
	       planting_window_start::text, planting_window_end::text,
	       harvest_window_start::text, harvest_window_end::text,
	       status, remark, active, version
	  FROM crop_season`

func scanSeason(row pgx.Row) (domain.Season, error) {
	var s domain.Season
	err := row.Scan(&s.ID, &s.Code, &s.Name, &s.CropYear, &s.StartsOn, &s.EndsOn,
		&s.PlantingWindowStart, &s.PlantingWindowEnd, &s.HarvestWindowStart, &s.HarvestWindowEnd,
		&s.Status, &s.Remark, &s.Active, &s.Version)
	return s, err
}

func (r *ActivityRepository) ListSeasons(ctx context.Context, f domain.Filter, page domain.Page) ([]domain.Season, int, error) {
	b := &builder{}
	if f.SeasonID != nil {
		b.eq("id", *f.SeasonID)
	}
	if f.CropYear != nil {
		b.eq("crop_year", *f.CropYear)
	}
	if !f.IncludeInactive {
		b.raw("active")
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		p := b.add("%" + strings.ToLower(s) + "%")
		b.raw(fmt.Sprintf("(lower(code) LIKE %s OR lower(name) LIKE %s)", p, p))
	}

	var total int
	if err := r.db.Pool().QueryRow(ctx, "SELECT count(*) FROM crop_season"+b.whereSQL(), b.args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count seasons: %w", err)
	}

	query := seasonSelect + b.whereSQL() +
		fmt.Sprintf(" ORDER BY crop_year DESC, code LIMIT %s OFFSET %s", b.add(page.Size), b.add(page.Offset()))
	rows, err := r.db.Pool().Query(ctx, query, b.args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list seasons: %w", err)
	}
	defer rows.Close()

	out := []domain.Season{}
	for rows.Next() {
		s, err := scanSeason(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

func (r *ActivityRepository) GetSeason(ctx context.Context, tx database.Querier, id int) (domain.Season, error) {
	s, err := scanSeason(r.q(tx).QueryRow(ctx, seasonSelect+" WHERE id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Season{}, domain.NotFound("Growing season", id)
	}
	return s, err
}

func (r *ActivityRepository) SaveSeason(ctx context.Context, tx database.Querier, id *int, in domain.SeasonInput) (int, error) {
	if id == nil {
		var newID int
		err := tx.QueryRow(ctx, `
			INSERT INTO crop_season(code, name, crop_year, starts_on, ends_on,
			                        planting_window_start, planting_window_end,
			                        harvest_window_start, harvest_window_end, status, remark, active)
			VALUES ($1, $2, $3, $4::date, $5::date, $6::date, $7::date, $8::date, $9::date, $10, $11, COALESCE($12, true))
			RETURNING id`,
			in.Code, in.Name, in.CropYear, in.StartsOn, in.EndsOn,
			in.PlantingWindowStart, in.PlantingWindowEnd, in.HarvestWindowStart, in.HarvestWindowEnd,
			in.Status, in.Remark, in.Active).Scan(&newID)
		return newID, mapActivityError(err, in.Code)
	}

	tag, err := tx.Exec(ctx, `
		UPDATE crop_season SET code = $2, name = $3, crop_year = $4, starts_on = $5::date, ends_on = $6::date,
		       planting_window_start = $7::date, planting_window_end = $8::date,
		       harvest_window_start = $9::date, harvest_window_end = $10::date,
		       status = $11, remark = $12, active = COALESCE($13, active), version = version + 1
		 WHERE id = $1 AND version = $14`,
		*id, in.Code, in.Name, in.CropYear, in.StartsOn, in.EndsOn,
		in.PlantingWindowStart, in.PlantingWindowEnd, in.HarvestWindowStart, in.HarvestWindowEnd,
		in.Status, in.Remark, in.Active, in.Version)
	if err != nil {
		return 0, mapActivityError(err, in.Code)
	}
	if tag.RowsAffected() == 0 {
		return 0, explainMiss(ctx, r.q(tx), "crop_season", "Growing season", *id)
	}
	return *id, nil
}

// ---------------------------------------------------------------- varieties

const varietySelect = `
	SELECT id, code, name, growing_period_months, seed_rate_per_ha, expected_yield_per_ha,
	       expected_loss_percent, remark, active, version
	  FROM cane_variety`

func (r *ActivityRepository) ListVarieties(ctx context.Context, f domain.Filter, page domain.Page) ([]domain.Variety, int, error) {
	b := &builder{}
	if f.VarietyID != nil {
		b.eq("id", *f.VarietyID)
	}
	if !f.IncludeInactive {
		b.raw("active")
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		p := b.add("%" + strings.ToLower(s) + "%")
		b.raw(fmt.Sprintf("(lower(code) LIKE %s OR lower(name) LIKE %s)", p, p))
	}

	var total int
	if err := r.db.Pool().QueryRow(ctx, "SELECT count(*) FROM cane_variety"+b.whereSQL(), b.args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count varieties: %w", err)
	}

	query := varietySelect + b.whereSQL() +
		fmt.Sprintf(" ORDER BY code LIMIT %s OFFSET %s", b.add(page.Size), b.add(page.Offset()))
	rows, err := r.db.Pool().Query(ctx, query, b.args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list varieties: %w", err)
	}
	defer rows.Close()

	out := []domain.Variety{}
	for rows.Next() {
		var v domain.Variety
		if err := rows.Scan(&v.ID, &v.Code, &v.Name, &v.GrowingPeriodMonths, &v.SeedRatePerHa,
			&v.ExpectedYieldPerHa, &v.ExpectedLossPercent, &v.Remark, &v.Active, &v.Version); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, rows.Err()
}

func (r *ActivityRepository) GetVariety(ctx context.Context, tx database.Querier, id int) (domain.Variety, error) {
	var v domain.Variety
	err := r.q(tx).QueryRow(ctx, varietySelect+" WHERE id = $1", id).Scan(
		&v.ID, &v.Code, &v.Name, &v.GrowingPeriodMonths, &v.SeedRatePerHa,
		&v.ExpectedYieldPerHa, &v.ExpectedLossPercent, &v.Remark, &v.Active, &v.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Variety{}, domain.NotFound("Cane variety", id)
	}
	return v, err
}

func (r *ActivityRepository) SaveVariety(ctx context.Context, tx database.Querier, id *int, in domain.VarietyInput) (int, error) {
	if id == nil {
		var newID int
		err := tx.QueryRow(ctx, `
			INSERT INTO cane_variety(code, name, growing_period_months, seed_rate_per_ha,
			                         expected_yield_per_ha, expected_loss_percent, remark, active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE($8, true)) RETURNING id`,
			in.Code, in.Name, in.GrowingPeriodMonths, in.SeedRatePerHa,
			in.ExpectedYieldPerHa, in.ExpectedLossPercent, in.Remark, in.Active).Scan(&newID)
		return newID, mapActivityError(err, in.Code)
	}

	tag, err := tx.Exec(ctx, `
		UPDATE cane_variety SET code = $2, name = $3, growing_period_months = $4, seed_rate_per_ha = $5,
		       expected_yield_per_ha = $6, expected_loss_percent = $7, remark = $8,
		       active = COALESCE($9, active), version = version + 1
		 WHERE id = $1 AND version = $10`,
		*id, in.Code, in.Name, in.GrowingPeriodMonths, in.SeedRatePerHa,
		in.ExpectedYieldPerHa, in.ExpectedLossPercent, in.Remark, in.Active, in.Version)
	if err != nil {
		return 0, mapActivityError(err, in.Code)
	}
	if tag.RowsAffected() == 0 {
		return 0, explainMiss(ctx, r.q(tx), "cane_variety", "Cane variety", *id)
	}
	return *id, nil
}

// ---------------------------------------------------------------- activities

const activitySelect = `
	SELECT a.id, a.company_id, a.code, a.name, a.category::text, a.applicable_crop_type::text,
	       a.sequence_no, a.standard_start_day_offset,
	       a.standard_capacity_per_hour, a.standard_capacity_per_day,
	       a.standard_duration_per_ha, a.standard_labour_days_per_ha,
	       a.required_tractor_type, a.required_equipment_category,
	       a.is_mandatory, a.requires_tractor, a.requires_equipment, a.requires_material,
	       a.requires_labour, a.allow_overlap, a.remark, a.active, a.version,
	       a.updated_at, a.updated_by
	  FROM planting_activity a`

func scanActivity(row pgx.Row) (domain.Activity, error) {
	var a domain.Activity
	err := row.Scan(&a.ID, &a.CompanyID, &a.Code, &a.Name, &a.Category, &a.ApplicableCropType,
		&a.SequenceNo, &a.StandardStartDayOffset,
		&a.StandardCapacityPerHour, &a.StandardCapacityPerDay,
		&a.StandardDurationPerHa, &a.StandardLabourDaysPerHa,
		&a.RequiredTractorType, &a.RequiredEquipmentCategory,
		&a.IsMandatory, &a.RequiresTractor, &a.RequiresEquipment, &a.RequiresMaterial,
		&a.RequiresLabour, &a.AllowOverlap, &a.Remark, &a.Active, &a.Version,
		&a.UpdatedAt, &a.UpdatedBy)
	return a, err
}

func (r *ActivityRepository) ListActivities(ctx context.Context, f domain.Filter, page domain.Page) ([]domain.Activity, int, error) {
	b := &builder{}
	if f.CompanyID != nil {
		b.eq("a.company_id", *f.CompanyID)
	}
	if f.ActivityID != nil {
		b.eq("a.id", *f.ActivityID)
	}
	if f.ActivityCategory != nil {
		b.where = append(b.where, "a.category = "+b.add(*f.ActivityCategory)+"::activity_category")
	}
	if f.PlantingType != nil {
		// An activity applies to a crop type when it names it, or when it applies to both.
		b.where = append(b.where,
			"(a.applicable_crop_type = 'Both' OR a.applicable_crop_type = "+b.add(*f.PlantingType)+"::applicable_crop_type)")
	}
	if !f.IncludeInactive {
		b.raw("a.active")
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		p := b.add("%" + strings.ToLower(s) + "%")
		b.raw(fmt.Sprintf("(lower(a.code) LIKE %s OR lower(a.name) LIKE %s)", p, p))
	}

	var total int
	if err := r.db.Pool().QueryRow(ctx,
		"SELECT count(*) FROM planting_activity a"+b.whereSQL(), b.args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count activities: %w", err)
	}

	query := activitySelect + b.whereSQL() +
		fmt.Sprintf(" ORDER BY a.sequence_no LIMIT %s OFFSET %s", b.add(page.Size), b.add(page.Offset()))
	rows, err := r.db.Pool().Query(ctx, query, b.args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list activities: %w", err)
	}
	defer rows.Close()

	out := []domain.Activity{}
	for rows.Next() {
		a, err := scanActivity(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// The dependencies come back with the activities: a caller listing the master data almost
	// always wants the chain too, and nineteen extra round trips to build one screen is worse than
	// one query that fetches them all.
	if err := r.attachDependencies(ctx, out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *ActivityRepository) attachDependencies(ctx context.Context, activities []domain.Activity) error {
	if len(activities) == 0 {
		return nil
	}
	ids := make([]int, 0, len(activities))
	for _, a := range activities {
		ids = append(ids, a.ID)
	}

	rows, err := r.db.Pool().Query(ctx, `
		SELECT d.id, d.activity_id, d.depends_on_id, p.code, p.name, p.sequence_no,
		       d.lag_days, d.is_blocking, d.remark
		  FROM activity_dependency d
		  JOIN planting_activity p ON p.id = d.depends_on_id
		 WHERE d.activity_id = ANY($1)
		 ORDER BY p.sequence_no`, ids)
	if err != nil {
		return fmt.Errorf("dependencies: %w", err)
	}
	defer rows.Close()

	byActivity := map[int][]domain.Dependency{}
	for rows.Next() {
		var d domain.Dependency
		if err := rows.Scan(&d.ID, &d.ActivityID, &d.DependsOnID, &d.DependsOnCode, &d.DependsOnName,
			&d.DependsOnSequenceNo, &d.LagDays, &d.IsBlocking, &d.Remark); err != nil {
			return err
		}
		byActivity[d.ActivityID] = append(byActivity[d.ActivityID], d)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for i := range activities {
		if deps, ok := byActivity[activities[i].ID]; ok {
			activities[i].Dependencies = deps
		} else {
			activities[i].Dependencies = []domain.Dependency{}
		}
	}
	return nil
}

func (r *ActivityRepository) GetActivity(ctx context.Context, tx database.Querier, id int) (domain.Activity, error) {
	a, err := scanActivity(r.q(tx).QueryRow(ctx, activitySelect+" WHERE a.id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Activity{}, domain.NotFound("Planting activity", id)
	}
	if err != nil {
		return domain.Activity{}, err
	}
	list := []domain.Activity{a}
	if err := r.attachDependencies(ctx, list); err != nil {
		return domain.Activity{}, err
	}
	return list[0], nil
}

func (r *ActivityRepository) SaveActivity(ctx context.Context, tx database.Querier, id *int, in domain.ActivityInput, actor string) (int, error) {
	if id == nil {
		var newID int
		err := tx.QueryRow(ctx, `
			INSERT INTO planting_activity(company_id, code, name, category, applicable_crop_type, sequence_no,
			    standard_start_day_offset, standard_capacity_per_hour, standard_capacity_per_day,
			    standard_duration_per_ha, standard_labour_days_per_ha,
			    required_tractor_type, required_equipment_category,
			    is_mandatory, requires_tractor, requires_equipment, requires_material, requires_labour,
			    allow_overlap, remark, active, created_by, updated_by)
			VALUES ($1, $2, $3, $4::activity_category, $5::applicable_crop_type, $6, $7, $8, $9, $10, $11,
			        $12, $13, $14, $15, $16, $17, $18, $19, $20, COALESCE($21, true), $22, $22)
			RETURNING id`,
			in.CompanyID, in.Code, in.Name, in.Category, in.ApplicableCropType, in.SequenceNo,
			in.StandardStartDayOffset, in.StandardCapacityPerHour, in.StandardCapacityPerDay,
			in.StandardDurationPerHa, in.StandardLabourDaysPerHa,
			in.RequiredTractorType, in.RequiredEquipmentCategory,
			in.IsMandatory, in.RequiresTractor, in.RequiresEquipment, in.RequiresMaterial, in.RequiresLabour,
			in.AllowOverlap, in.Remark, in.Active, actor).Scan(&newID)
		return newID, mapActivityError(err, in.Code)
	}

	tag, err := tx.Exec(ctx, `
		UPDATE planting_activity SET code = $2, name = $3, category = $4::activity_category,
		    applicable_crop_type = $5::applicable_crop_type, sequence_no = $6,
		    standard_start_day_offset = $7, standard_capacity_per_hour = $8, standard_capacity_per_day = $9,
		    standard_duration_per_ha = $10, standard_labour_days_per_ha = $11,
		    required_tractor_type = $12, required_equipment_category = $13,
		    is_mandatory = $14, requires_tractor = $15, requires_equipment = $16, requires_material = $17,
		    requires_labour = $18, allow_overlap = $19, remark = $20, active = COALESCE($21, active),
		    version = version + 1, updated_at = now(), updated_by = $22
		 WHERE id = $1 AND version = $23`,
		*id, in.Code, in.Name, in.Category, in.ApplicableCropType, in.SequenceNo,
		in.StandardStartDayOffset, in.StandardCapacityPerHour, in.StandardCapacityPerDay,
		in.StandardDurationPerHa, in.StandardLabourDaysPerHa,
		in.RequiredTractorType, in.RequiredEquipmentCategory,
		in.IsMandatory, in.RequiresTractor, in.RequiresEquipment, in.RequiresMaterial, in.RequiresLabour,
		in.AllowOverlap, in.Remark, in.Active, actor, in.Version)
	if err != nil {
		return 0, mapActivityError(err, in.Code)
	}
	if tag.RowsAffected() == 0 {
		return 0, explainMiss(ctx, r.q(tx), "planting_activity", "Planting activity", *id)
	}
	return *id, nil
}

// ---------------------------------------------------------------- dependencies

func (r *ActivityRepository) AddDependency(ctx context.Context, tx database.Querier, in domain.DependencyInput) (int, error) {
	var id int
	err := tx.QueryRow(ctx, `
		INSERT INTO activity_dependency(activity_id, depends_on_id, lag_days, is_blocking, remark)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		in.ActivityID, in.DependsOnID, in.LagDays, in.IsBlocking, in.Remark).Scan(&id)
	return id, mapActivityError(err, fmt.Sprintf("%d → %d", in.DependsOnID, in.ActivityID))
}

func (r *ActivityRepository) DeleteDependency(ctx context.Context, tx database.Querier, id int) error {
	tag, err := tx.Exec(ctx, `DELETE FROM activity_dependency WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.NotFound("Activity dependency", id)
	}
	return nil
}

// Chain returns every activity in sequence with its blocking predecessors resolved — what the
// activity-plan engine walks when it lays a projection out over the calendar.
func (r *ActivityRepository) Chain(ctx context.Context, companyID int, cropType string) ([]domain.Activity, error) {
	f := domain.Filter{CompanyID: &companyID}
	if cropType != "" {
		f.PlantingType = &cropType
	}
	activities, _, err := r.ListActivities(ctx, f, domain.Page{Number: 1, Size: 500})
	return activities, err
}

// mapActivityError turns this module's constraint names and RAISE messages into errors a caller
// can act on, falling back to the shared mapping for the ones every table shares.
func mapActivityError(err error, key string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "DEPENDENCY_CYCLE"):
		return domain.Invalid([]domain.FieldError{{
			Field: "dependsOnId", Message: afterMarker(msg, "DEPENDENCY_CYCLE: ")}})
	case strings.Contains(msg, "dependency_not_self"):
		return domain.Invalid([]domain.FieldError{{
			Field: "dependsOnId", Message: "An activity cannot wait for itself."}})
	case strings.Contains(msg, "dependency_lag_not_negative"):
		return domain.Invalid([]domain.FieldError{{
			Field: "lagDays", Message: "A lag cannot be negative; an activity cannot start before the one it waits for."}})
	case strings.Contains(msg, "activity_needs_a_daily_capacity"):
		return domain.Invalid([]domain.FieldError{{
			Field: "standardCapacityPerDay",
			Message: "An activity that needs a tractor or an implement must have a daily capacity, " +
				"or the engine cannot work out how long it takes."}})
	case strings.Contains(msg, "activity_sequence_positive"):
		return domain.Invalid([]domain.FieldError{{
			Field: "sequenceNo", Message: "The sequence number must be greater than zero."}})
	case strings.Contains(msg, "activity_capacity_not_negative"),
		strings.Contains(msg, "activity_duration_not_negative"):
		return domain.Invalid([]domain.FieldError{{
			Field: "standardCapacityPerDay", Message: "Capacities and durations cannot be negative."}})
	case strings.Contains(msg, "crop_season_planting_window"),
		strings.Contains(msg, "crop_season_harvest_window"):
		return domain.Invalid([]domain.FieldError{{
			Field: "plantingWindowEnd", Message: "A window cannot end before it starts."}})
	case strings.Contains(msg, "crop_season_dates"):
		return domain.Invalid([]domain.FieldError{{
			Field: "endsOn", Message: "The season must end after it starts."}})
	case strings.Contains(msg, "cane_variety_loss"):
		return domain.Invalid([]domain.FieldError{{
			Field: "expectedLossPercent", Message: "The expected loss must be between 0 and 100 per cent."}})
	case strings.Contains(msg, "cane_variety_seed_rate"):
		return domain.Invalid([]domain.FieldError{{
			Field: "seedRatePerHa", Message: "The seed rate must be greater than zero."}})
	}
	return mapWriteError(err, "Activity", key)
}
