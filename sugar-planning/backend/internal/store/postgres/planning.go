package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

type planning struct{ s *Store }

// Planning returns the planning repository.
func (s *Store) Planning() store.Planning { return planning{s} }

// ---------------------------------------------------------------------------
// Seasons
// ---------------------------------------------------------------------------

const seasonCols = `id, company_id, factory_id, code, name, start_date, end_date,
	planned_days, status, created_at, created_by, updated_at, updated_by, row_version`

func scanSeason(r scanner) (domain.Season, error) {
	var s domain.Season
	var start time.Time
	var end *time.Time
	err := r.Scan(&s.ID, &s.CompanyID, &s.FactoryID, &s.Code, &s.Name, &start, &end,
		&s.PlannedDays, &s.Status, &s.CreatedAt, &s.CreatedBy, &s.UpdatedAt, &s.UpdatedBy, &s.RowVersion)
	s.StartDate, s.EndDate = mustDate(start), bd(end)
	return s, err
}

func (p planning) ListSeasons(ctx context.Context, opts store.ListOptions) (store.Page[domain.Season], error) {
	opts = opts.Normalise()
	var where []string
	var args []any
	if opts.ParentID != "" {
		args = append(args, opts.ParentID)
		where = append(where, fmt.Sprintf("(company_id = $%d OR factory_id = $%d)", len(args), len(args)))
	}
	if opts.Search != "" {
		args = append(args, "%"+strings.ToLower(opts.Search)+"%")
		where = append(where, fmt.Sprintf("(lower(code) LIKE $%d OR lower(name) LIKE $%d)", len(args), len(args)))
	}
	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := p.s.q.QueryRow(ctx, "SELECT count(*) FROM seasons"+clause, args...).Scan(&total); err != nil {
		return store.Page[domain.Season]{}, mapError("season", err)
	}
	args = append(args, opts.Top, opts.Skip)
	rows, err := p.s.q.Query(ctx, fmt.Sprintf(
		"SELECT %s FROM seasons%s ORDER BY code LIMIT $%d OFFSET $%d",
		seasonCols, clause, len(args)-1, len(args)), args...)
	if err != nil {
		return store.Page[domain.Season]{}, mapError("season", err)
	}
	defer rows.Close()

	items := []domain.Season{}
	for rows.Next() {
		s, err := scanSeason(rows)
		if err != nil {
			return store.Page[domain.Season]{}, mapError("season", err)
		}
		items = append(items, s)
	}
	return store.Page[domain.Season]{Items: items, Count: total}, mapError("season", rows.Err())
}

func (p planning) GetSeason(ctx context.Context, id string) (domain.Season, error) {
	if _, err := uuid.Parse(id); err != nil {
		return domain.Season{}, fmt.Errorf("%w: season %s", domain.ErrNotFound, id)
	}
	s, err := scanSeason(p.s.q.QueryRow(ctx, "SELECT "+seasonCols+" FROM seasons WHERE id = $1", id))
	return s, mapError("season "+id, err)
}

func (p planning) SaveSeason(ctx context.Context, s domain.Season, actor string) (domain.Season, error) {
	now := time.Now().UTC()
	if s.ID == "" {
		s.ID = uuid.NewString()
		s.CreatedAt, s.CreatedBy, s.UpdatedAt, s.UpdatedBy, s.RowVersion = now, actor, now, actor, 1
		_, err := p.s.q.Exec(ctx, `INSERT INTO seasons
			(id, company_id, factory_id, code, name, start_date, end_date, planned_days, status,
			 created_at, created_by, updated_at, updated_by, row_version)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
			s.ID, s.CompanyID, s.FactoryID, s.Code, s.Name, nd(s.StartDate), nd(s.EndDate),
			s.PlannedDays, s.Status, now, actor, now, actor, int64(1))
		if err != nil {
			return domain.Season{}, mapError("season "+s.Code, err)
		}
		return s, nil
	}
	var created time.Time
	var createdBy string
	var version int64
	err := p.s.q.QueryRow(ctx, `UPDATE seasons SET
			company_id=$1, factory_id=$2, code=$3, name=$4, start_date=$5, end_date=$6,
			planned_days=$7, status=$8, updated_at=$9, updated_by=$10, row_version = row_version + 1
		WHERE id=$11 AND row_version=$12
		RETURNING created_at, created_by, row_version`,
		s.CompanyID, s.FactoryID, s.Code, s.Name, nd(s.StartDate), nd(s.EndDate),
		s.PlannedDays, s.Status, now, actor, s.ID, s.RowVersion).
		Scan(&created, &createdBy, &version)
	if err != nil {
		if isNoRows(err) {
			return domain.Season{}, versionConflict(ctx, p.s, "seasons", "season", s.ID, s.RowVersion)
		}
		return domain.Season{}, mapError("season "+s.Code, err)
	}
	s.CreatedAt, s.CreatedBy, s.UpdatedAt, s.UpdatedBy, s.RowVersion = created, createdBy, now, actor, version
	return s, nil
}

// ---------------------------------------------------------------------------
// Plan versions
// ---------------------------------------------------------------------------

const versionCols = `id, season_id, version_no, code, description, plan_type, status, owner,
	source_version_id, effective_from, effective_to, submitted_at, approved_at, approved_by,
	released_at, locked_through, comment, created_at, created_by, updated_at, updated_by, row_version`

func scanVersion(r scanner) (domain.PlanVersion, error) {
	var v domain.PlanVersion
	var planType, status string
	var source, approvedBy *string
	var effFrom, effTo, locked *time.Time
	err := r.Scan(&v.ID, &v.SeasonID, &v.VersionNo, &v.Code, &v.Description, &planType, &status,
		&v.Owner, &source, &effFrom, &effTo, &v.SubmittedAt, &v.ApprovedAt, &approvedBy,
		&v.ReleasedAt, &locked, &v.Comment,
		&v.CreatedAt, &v.CreatedBy, &v.UpdatedAt, &v.UpdatedBy, &v.RowVersion)
	v.PlanType, v.Status = domain.PlanType(planType), domain.PlanStatus(status)
	v.SourceVersion, v.ApprovedBy = ds(source), ds(approvedBy)
	v.EffectiveFrom, v.EffectiveTo, v.LockedThrough = bd(effFrom), bd(effTo), bd(locked)
	return v, err
}

func (p planning) ListVersions(ctx context.Context, seasonID string, opts store.ListOptions) (store.Page[domain.PlanVersion], error) {
	opts = opts.Normalise()
	var where []string
	var args []any
	if seasonID != "" {
		args = append(args, seasonID)
		where = append(where, fmt.Sprintf("season_id = $%d", len(args)))
	}
	if opts.Search != "" {
		args = append(args, "%"+strings.ToLower(opts.Search)+"%")
		where = append(where, fmt.Sprintf("(lower(code) LIKE $%d OR lower(description) LIKE $%d)",
			len(args), len(args)))
	}
	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := p.s.q.QueryRow(ctx, "SELECT count(*) FROM plan_versions"+clause, args...).Scan(&total); err != nil {
		return store.Page[domain.PlanVersion]{}, mapError("plan version", err)
	}
	args = append(args, opts.Top, opts.Skip)
	rows, err := p.s.q.Query(ctx, fmt.Sprintf(
		"SELECT %s FROM plan_versions%s ORDER BY version_no LIMIT $%d OFFSET $%d",
		versionCols, clause, len(args)-1, len(args)), args...)
	if err != nil {
		return store.Page[domain.PlanVersion]{}, mapError("plan version", err)
	}
	defer rows.Close()

	items := []domain.PlanVersion{}
	for rows.Next() {
		v, err := scanVersion(rows)
		if err != nil {
			return store.Page[domain.PlanVersion]{}, mapError("plan version", err)
		}
		items = append(items, v)
	}
	return store.Page[domain.PlanVersion]{Items: items, Count: total}, mapError("plan version", rows.Err())
}

func (p planning) GetVersion(ctx context.Context, id string) (domain.PlanVersion, error) {
	if _, err := uuid.Parse(id); err != nil {
		return domain.PlanVersion{}, fmt.Errorf("%w: plan version %s", domain.ErrNotFound, id)
	}
	v, err := scanVersion(p.s.q.QueryRow(ctx, "SELECT "+versionCols+" FROM plan_versions WHERE id = $1", id))
	return v, mapError("plan version "+id, err)
}

func (p planning) SaveVersion(ctx context.Context, v domain.PlanVersion, actor string) (domain.PlanVersion, error) {
	now := time.Now().UTC()
	if v.ID == "" {
		if v.VersionNo == 0 {
			// Next number within the season. Inside a transaction this is
			// serialised by the unique index, which turns a race into a
			// duplicate-key error rather than two versions with one number.
			var max *int
			if err := p.s.q.QueryRow(ctx,
				"SELECT max(version_no) FROM plan_versions WHERE season_id = $1", v.SeasonID).Scan(&max); err != nil {
				return domain.PlanVersion{}, mapError("plan version", err)
			}
			v.VersionNo = 1
			if max != nil {
				v.VersionNo = *max + 1
			}
		}
		v.ID = uuid.NewString()
		v.CreatedAt, v.CreatedBy, v.UpdatedAt, v.UpdatedBy, v.RowVersion = now, actor, now, actor, 1
		_, err := p.s.q.Exec(ctx, `INSERT INTO plan_versions
			(id, season_id, version_no, code, description, plan_type, status, owner, source_version_id,
			 effective_from, effective_to, submitted_at, approved_at, approved_by, released_at,
			 locked_through, comment, created_at, created_by, updated_at, updated_by, row_version)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)`,
			v.ID, v.SeasonID, v.VersionNo, v.Code, v.Description, string(v.PlanType), string(v.Status),
			v.Owner, nu(v.SourceVersion), nd(v.EffectiveFrom), nd(v.EffectiveTo), v.SubmittedAt,
			v.ApprovedAt, nu(v.ApprovedBy), v.ReleasedAt, nd(v.LockedThrough), v.Comment,
			now, actor, now, actor, int64(1))
		if err != nil {
			return domain.PlanVersion{}, mapError("plan version "+v.Code, err)
		}
		return v, nil
	}
	var created time.Time
	var createdBy string
	var version int64
	err := p.s.q.QueryRow(ctx, `UPDATE plan_versions SET
			code=$1, description=$2, plan_type=$3, status=$4, owner=$5, source_version_id=$6,
			effective_from=$7, effective_to=$8, submitted_at=$9, approved_at=$10, approved_by=$11,
			released_at=$12, locked_through=$13, comment=$14,
			updated_at=$15, updated_by=$16, row_version = row_version + 1
		WHERE id=$17 AND row_version=$18
		RETURNING created_at, created_by, row_version`,
		v.Code, v.Description, string(v.PlanType), string(v.Status), v.Owner, nu(v.SourceVersion),
		nd(v.EffectiveFrom), nd(v.EffectiveTo), v.SubmittedAt, v.ApprovedAt, nu(v.ApprovedBy),
		v.ReleasedAt, nd(v.LockedThrough), v.Comment, now, actor, v.ID, v.RowVersion).
		Scan(&created, &createdBy, &version)
	if err != nil {
		if isNoRows(err) {
			return domain.PlanVersion{}, versionConflict(ctx, p.s, "plan_versions", "plan version", v.ID, v.RowVersion)
		}
		return domain.PlanVersion{}, mapError("plan version "+v.Code, err)
	}
	v.CreatedAt, v.CreatedBy, v.UpdatedAt, v.UpdatedBy, v.RowVersion = created, createdBy, now, actor, version
	return v, nil
}

// ---------------------------------------------------------------------------
// Assumptions and product mix
// ---------------------------------------------------------------------------

func (p planning) ListAssumptions(ctx context.Context, versionID string) ([]domain.PlanAssumption, error) {
	rows, err := p.s.q.Query(ctx, `SELECT id, version_id, code, description, value, uom,
		valid_from, valid_to, created_at, created_by, updated_at, updated_by, row_version
		FROM plan_assumptions WHERE version_id = $1 ORDER BY code`, versionID)
	if err != nil {
		return nil, mapError("assumption", err)
	}
	defer rows.Close()

	var out []domain.PlanAssumption
	for rows.Next() {
		var a domain.PlanAssumption
		var from, to *time.Time
		if err := rows.Scan(&a.ID, &a.VersionID, &a.Code, &a.Description, &a.Value, &a.UOM,
			&from, &to, &a.CreatedAt, &a.CreatedBy, &a.UpdatedAt, &a.UpdatedBy, &a.RowVersion); err != nil {
			return nil, mapError("assumption", err)
		}
		a.ValidFrom, a.ValidTo = bd(from), bd(to)
		out = append(out, a)
	}
	return out, mapError("assumption", rows.Err())
}

func (p planning) SaveAssumption(ctx context.Context, a domain.PlanAssumption, actor string) (domain.PlanAssumption, error) {
	now := time.Now().UTC()
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	// The business key is (version, code, valid_from); an upsert on it means an
	// import can be re-run without creating a second row for the same date.
	err := p.s.q.QueryRow(ctx, `INSERT INTO plan_assumptions
		(id, version_id, code, description, value, uom, valid_from, valid_to,
		 created_at, created_by, updated_at, updated_by, row_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$9,$10,1)
		ON CONFLICT (version_id, code, COALESCE(valid_from, DATE '0001-01-01')) DO UPDATE SET
			description = EXCLUDED.description, value = EXCLUDED.value, uom = EXCLUDED.uom,
			valid_to = EXCLUDED.valid_to, updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by, row_version = plan_assumptions.row_version + 1
		RETURNING id, created_at, created_by, row_version`,
		a.ID, a.VersionID, a.Code, a.Description, a.Value, a.UOM, nd(a.ValidFrom), nd(a.ValidTo),
		now, actor).
		Scan(&a.ID, &a.CreatedAt, &a.CreatedBy, &a.RowVersion)
	if err != nil {
		return domain.PlanAssumption{}, mapError("assumption "+a.Code, err)
	}
	a.UpdatedAt, a.UpdatedBy = now, actor
	return a, nil
}

func (p planning) DeleteAssumption(ctx context.Context, id string) error {
	tag, err := p.s.q.Exec(ctx, "DELETE FROM plan_assumptions WHERE id = $1", id)
	if err != nil {
		return mapError("assumption", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: assumption %s", domain.ErrNotFound, id)
	}
	return nil
}

func (p planning) ListMix(ctx context.Context, versionID string) ([]domain.ProductMixEntry, error) {
	rows, err := p.s.q.Query(ctx, `SELECT id, version_id, product_id, packaging_id, warehouse_id,
		line_id, season_tons, daily_rate_tons, created_at, created_by, updated_at, updated_by, row_version
		FROM product_mix_entries WHERE version_id = $1 ORDER BY product_id`, versionID)
	if err != nil {
		return nil, mapError("product mix", err)
	}
	defer rows.Close()

	var out []domain.ProductMixEntry
	for rows.Next() {
		var m domain.ProductMixEntry
		var pack, wh, line *string
		if err := rows.Scan(&m.ID, &m.VersionID, &m.ProductID, &pack, &wh, &line,
			&m.SeasonTons, &m.DailyRateTons,
			&m.CreatedAt, &m.CreatedBy, &m.UpdatedAt, &m.UpdatedBy, &m.RowVersion); err != nil {
			return nil, mapError("product mix", err)
		}
		m.PackagingID, m.WarehouseID, m.LineID = ds(pack), ds(wh), ds(line)
		out = append(out, m)
	}
	return out, mapError("product mix", rows.Err())
}

func (p planning) SaveMix(ctx context.Context, m domain.ProductMixEntry, actor string) (domain.ProductMixEntry, error) {
	now := time.Now().UTC()
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	err := p.s.q.QueryRow(ctx, `INSERT INTO product_mix_entries
		(id, version_id, product_id, packaging_id, warehouse_id, line_id, season_tons, daily_rate_tons,
		 created_at, created_by, updated_at, updated_by, row_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$9,$10,1)
		ON CONFLICT (version_id, product_id, COALESCE(packaging_id, '00000000-0000-0000-0000-000000000000'::uuid))
		DO UPDATE SET warehouse_id = EXCLUDED.warehouse_id, line_id = EXCLUDED.line_id,
			season_tons = EXCLUDED.season_tons, daily_rate_tons = EXCLUDED.daily_rate_tons,
			updated_at = EXCLUDED.updated_at, updated_by = EXCLUDED.updated_by,
			row_version = product_mix_entries.row_version + 1
		RETURNING id, created_at, created_by, row_version`,
		m.ID, m.VersionID, m.ProductID, nu(m.PackagingID), nu(m.WarehouseID), nu(m.LineID),
		m.SeasonTons, m.DailyRateTons, now, actor).
		Scan(&m.ID, &m.CreatedAt, &m.CreatedBy, &m.RowVersion)
	if err != nil {
		return domain.ProductMixEntry{}, mapError("product mix", err)
	}
	m.UpdatedAt, m.UpdatedBy = now, actor
	return m, nil
}

func (p planning) DeleteMix(ctx context.Context, id string) error {
	tag, err := p.s.q.Exec(ctx, "DELETE FROM product_mix_entries WHERE id = $1", id)
	if err != nil {
		return mapError("product mix", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: product mix entry %s", domain.ErrNotFound, id)
	}
	return nil
}

// versionConflict reports whether a failed guarded update was a missing row or
// a concurrent change.
func versionConflict(ctx context.Context, s *Store, table, label, id string, expected int64) error {
	var current int64
	var updatedBy string
	err := s.q.QueryRow(ctx,
		fmt.Sprintf("SELECT row_version, updated_by FROM %s WHERE id = $1", table), id).
		Scan(&current, &updatedBy)
	if err != nil {
		if isNoRows(err) {
			return fmt.Errorf("%w: %s %s", domain.ErrNotFound, label, id)
		}
		return mapError(label, err)
	}
	return fmt.Errorf("%w: %s %s was changed by %s (version %d, you have %d)",
		domain.ErrConflict, label, id, updatedBy, current, expected)
}
