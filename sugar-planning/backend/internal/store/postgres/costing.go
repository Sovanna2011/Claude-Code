package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

type costing struct{ s *Store }

// Costing returns the costing repository.
func (s *Store) Costing() store.Costing { return costing{s} }

// ---------------------------------------------------------------------------
// Elements
// ---------------------------------------------------------------------------

const elementCols = `id, code, name, category, driver, variable, note,
	valid_from, valid_to, active, created_at, created_by, updated_at, updated_by, row_version`

func scanElement(r scanner) (domain.CostElement, error) {
	var e domain.CostElement
	var category, driver string
	var from, to *time.Time
	err := r.Scan(&e.ID, &e.Code, &e.Name, &category, &driver, &e.Variable, &e.Note,
		&from, &to, &e.Active, &e.CreatedAt, &e.CreatedBy, &e.UpdatedAt, &e.UpdatedBy, &e.RowVersion)
	e.Category, e.Driver = domain.CostCategory(category), domain.CostDriver(driver)
	e.ValidFrom, e.ValidTo = bd(from), bd(to)
	return e, err
}

func (c costing) ListElements(ctx context.Context, opts store.ListOptions) (store.Page[domain.CostElement], error) {
	opts = opts.Normalise()
	w := &execWhere{}
	if opts.Active != nil {
		w.eq("active", *opts.Active)
	}
	if opts.Search != "" {
		w.args = append(w.args, "%"+strings.ToLower(opts.Search)+"%")
		w.raw(fmt.Sprintf("lower(code || ' ' || name) LIKE $%d", len(w.args)))
	}
	clause := w.sql()

	var total int
	if err := c.s.q.QueryRow(ctx, "SELECT count(*) FROM cost_elements"+clause, w.args...).
		Scan(&total); err != nil {
		return store.Page[domain.CostElement]{}, mapError("cost element", err)
	}
	w.args = append(w.args, opts.Top, opts.Skip)
	limit := fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(w.args)-1, len(w.args))

	rows, err := c.s.q.Query(ctx, "SELECT "+elementCols+" FROM cost_elements"+clause+
		" ORDER BY category, code"+limit, w.args...)
	if err != nil {
		return store.Page[domain.CostElement]{}, mapError("cost element", err)
	}
	defer rows.Close()

	items := []domain.CostElement{}
	for rows.Next() {
		e, err := scanElement(rows)
		if err != nil {
			return store.Page[domain.CostElement]{}, mapError("cost element", err)
		}
		items = append(items, e)
	}
	return store.Page[domain.CostElement]{Items: items, Count: total},
		mapError("cost element", rows.Err())
}

func (c costing) GetElement(ctx context.Context, id string) (domain.CostElement, error) {
	if _, err := uuid.Parse(id); err != nil {
		return domain.CostElement{}, fmt.Errorf("%w: cost element %s", domain.ErrNotFound, id)
	}
	e, err := scanElement(c.s.q.QueryRow(ctx,
		"SELECT "+elementCols+" FROM cost_elements WHERE id = $1", id))
	return e, mapError("cost element "+id, err)
}

func (c costing) SaveElement(ctx context.Context, e domain.CostElement, actor string) (domain.CostElement, error) {
	now := nowUTC()
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	// The code is the business key, so a repeat updates in place: the cost
	// structure is configuration and re-running a seed must be idempotent.
	var version int64
	err := c.s.q.QueryRow(ctx, `INSERT INTO cost_elements
		(id, code, name, category, driver, variable, note, valid_from, valid_to, active,
		 created_at, created_by, updated_at, updated_by, row_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$11,$12,1)
		ON CONFLICT (code) DO UPDATE SET
			name = EXCLUDED.name, category = EXCLUDED.category, driver = EXCLUDED.driver,
			variable = EXCLUDED.variable, note = EXCLUDED.note,
			valid_from = EXCLUDED.valid_from, valid_to = EXCLUDED.valid_to,
			active = EXCLUDED.active, updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by, row_version = cost_elements.row_version + 1
		RETURNING id, created_at, created_by, row_version`,
		e.ID, e.Code, e.Name, string(e.Category), string(e.Driver), e.Variable, e.Note,
		nd(e.ValidFrom), nd(e.ValidTo), e.Active, now, actor).
		Scan(&e.ID, &e.CreatedAt, &e.CreatedBy, &version)
	if err != nil {
		return domain.CostElement{}, mapError("cost element "+e.Code, err)
	}
	e.UpdatedAt, e.UpdatedBy, e.RowVersion = now, actor, version
	return e, nil
}

// ---------------------------------------------------------------------------
// Rates
// ---------------------------------------------------------------------------

const rateCols = `id, element_id, factory_id, season_id, rate_type, rate, currency,
	valid_from, valid_to, note, created_at, created_by, updated_at, updated_by, row_version`

func scanRate(r scanner) (domain.CostRate, error) {
	var c domain.CostRate
	var season *string
	var rateType string
	var from time.Time
	var to *time.Time
	err := r.Scan(&c.ID, &c.ElementID, &c.FactoryID, &season, &rateType, &c.Rate, &c.Currency,
		&from, &to, &c.Note, &c.CreatedAt, &c.CreatedBy, &c.UpdatedAt, &c.UpdatedBy, &c.RowVersion)
	c.SeasonID, c.RateType = ds(season), domain.RateType(rateType)
	c.ValidFrom, c.ValidTo = mustDate(from), bd(to)
	return c, err
}

// ListRates hands over every rate matching the filter and leaves the effective
// dating to the domain, so the two store implementations cannot disagree about
// which rate applies on a date.
func (c costing) ListRates(ctx context.Context, f store.CostFilter) ([]domain.CostRate, error) {
	w := &execWhere{}
	if f.FactoryID != "" {
		w.eq("factory_id", f.FactoryID)
	}
	if f.SeasonID != "" {
		w.args = append(w.args, f.SeasonID)
		w.raw(fmt.Sprintf("(season_id IS NULL OR season_id = $%d)", len(w.args)))
	}
	if f.ElementID != "" {
		w.eq("element_id", f.ElementID)
	}
	if f.RateType != "" {
		w.eq("rate_type", f.RateType)
	}
	if f.On != "" {
		w.args = append(w.args, nd(f.On))
		w.raw(fmt.Sprintf("valid_from <= $%d AND (valid_to IS NULL OR valid_to >= $%d)",
			len(w.args), len(w.args)))
	}

	rows, err := c.s.q.Query(ctx, "SELECT "+rateCols+" FROM cost_rates"+w.sql()+
		" ORDER BY element_id, rate_type, valid_from", w.args...)
	if err != nil {
		return nil, mapError("cost rate", err)
	}
	defer rows.Close()

	var out []domain.CostRate
	for rows.Next() {
		r, err := scanRate(rows)
		if err != nil {
			return nil, mapError("cost rate", err)
		}
		out = append(out, r)
	}
	return out, mapError("cost rate", rows.Err())
}

func (c costing) SaveRate(ctx context.Context, r domain.CostRate, actor string) (domain.CostRate, error) {
	now := nowUTC()
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	// Element, factory, type and start date are the business key: re-entering a
	// rate for a date that already has one corrects it rather than leaving two
	// rates silently competing.
	var version int64
	err := c.s.q.QueryRow(ctx, `INSERT INTO cost_rates
		(id, element_id, factory_id, season_id, rate_type, rate, currency,
		 valid_from, valid_to, note, created_at, created_by, updated_at, updated_by, row_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$11,$12,1)
		ON CONFLICT (element_id, factory_id, rate_type, valid_from) DO UPDATE SET
			season_id = EXCLUDED.season_id, rate = EXCLUDED.rate,
			currency = EXCLUDED.currency, valid_to = EXCLUDED.valid_to,
			note = EXCLUDED.note, updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by, row_version = cost_rates.row_version + 1
		RETURNING id, created_at, created_by, row_version`,
		r.ID, r.ElementID, r.FactoryID, nu(r.SeasonID), string(r.RateType), r.Rate, r.Currency,
		nd(r.ValidFrom), nd(r.ValidTo), r.Note, now, actor).
		Scan(&r.ID, &r.CreatedAt, &r.CreatedBy, &version)
	if err != nil {
		return domain.CostRate{}, mapError("cost rate", err)
	}
	r.UpdatedAt, r.UpdatedBy, r.RowVersion = now, actor, version
	return r, nil
}

func (c costing) DeleteRate(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: cost rate %s", domain.ErrNotFound, id)
	}
	tag, err := c.s.q.Exec(ctx, "DELETE FROM cost_rates WHERE id = $1", id)
	if err != nil {
		return mapError("cost rate", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: cost rate %s", domain.ErrNotFound, id)
	}
	return nil
}

func (c costing) ListExchangeRates(ctx context.Context) ([]domain.ExchangeRate, error) {
	rows, err := c.s.q.Query(ctx, `SELECT id, from_currency, to_currency, rate, valid_from,
		created_at, created_by, updated_at, updated_by, row_version
		FROM exchange_rates ORDER BY from_currency, to_currency, valid_from`)
	if err != nil {
		return nil, mapError("exchange rate", err)
	}
	defer rows.Close()

	var out []domain.ExchangeRate
	for rows.Next() {
		var r domain.ExchangeRate
		var from time.Time
		if err := rows.Scan(&r.ID, &r.FromCurrency, &r.ToCurrency, &r.Rate, &from,
			&r.CreatedAt, &r.CreatedBy, &r.UpdatedAt, &r.UpdatedBy, &r.RowVersion); err != nil {
			return nil, mapError("exchange rate", err)
		}
		r.ValidFrom = mustDate(from)
		out = append(out, r)
	}
	return out, mapError("exchange rate", rows.Err())
}

func (c costing) SaveExchangeRate(ctx context.Context, r domain.ExchangeRate, actor string) (domain.ExchangeRate, error) {
	now := nowUTC()
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	var version int64
	err := c.s.q.QueryRow(ctx, `INSERT INTO exchange_rates
		(id, from_currency, to_currency, rate, valid_from,
		 created_at, created_by, updated_at, updated_by, row_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$6,$7,1)
		ON CONFLICT (from_currency, to_currency, valid_from) DO UPDATE SET
			rate = EXCLUDED.rate, updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by, row_version = exchange_rates.row_version + 1
		RETURNING id, created_at, created_by, row_version`,
		r.ID, r.FromCurrency, r.ToCurrency, r.Rate, nd(r.ValidFrom), now, actor).
		Scan(&r.ID, &r.CreatedAt, &r.CreatedBy, &version)
	if err != nil {
		return domain.ExchangeRate{}, mapError("exchange rate", err)
	}
	r.UpdatedAt, r.UpdatedBy, r.RowVersion = now, actor, version
	return r, nil
}

// ---------------------------------------------------------------------------
// Runs
// ---------------------------------------------------------------------------

const runCols = `id, season_id, version_id, factory_id, code, from_date, to_date, currency,
	totals, note, created_at, created_by, updated_at, updated_by, row_version`

func scanRun(r scanner) (domain.CostRun, error) {
	var run domain.CostRun
	var version *string
	var from, to time.Time
	var totals []byte
	err := r.Scan(&run.ID, &run.SeasonID, &version, &run.FactoryID, &run.Code,
		&from, &to, &run.Currency, &totals, &run.Note,
		&run.CreatedAt, &run.CreatedBy, &run.UpdatedAt, &run.UpdatedBy, &run.RowVersion)
	run.VersionID = ds(version)
	run.From, run.To = mustDate(from), mustDate(to)
	if err == nil && len(totals) > 0 {
		// A malformed totals document is not worth failing the read: the lines
		// are the authority and the totals can be recomputed from them.
		_ = json.Unmarshal(totals, &run.Totals)
	}
	return run, err
}

func (c costing) ListRuns(ctx context.Context, f store.CostFilter) (store.Page[domain.CostRun], error) {
	w := &execWhere{}
	if f.SeasonID != "" {
		w.eq("season_id", f.SeasonID)
	}
	if f.VersionID != "" {
		w.eq("version_id", f.VersionID)
	}
	if f.FactoryID != "" {
		w.eq("factory_id", f.FactoryID)
	}
	clause := w.sql()

	var total int
	if err := c.s.q.QueryRow(ctx, "SELECT count(*) FROM cost_runs"+clause, w.args...).
		Scan(&total); err != nil {
		return store.Page[domain.CostRun]{}, mapError("cost run", err)
	}
	limit := w.limit(store.ExecutionFilter{Skip: f.Skip, Top: f.Top})

	rows, err := c.s.q.Query(ctx, "SELECT "+runCols+" FROM cost_runs"+clause+
		" ORDER BY from_date DESC, code"+limit, w.args...)
	if err != nil {
		return store.Page[domain.CostRun]{}, mapError("cost run", err)
	}
	defer rows.Close()

	// A list carries the totals only; the lines belong to the detail view.
	items := []domain.CostRun{}
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return store.Page[domain.CostRun]{}, mapError("cost run", err)
		}
		items = append(items, run)
	}
	return store.Page[domain.CostRun]{Items: items, Count: total}, mapError("cost run", rows.Err())
}

func (c costing) GetRun(ctx context.Context, id string) (domain.CostRun, error) {
	if _, err := uuid.Parse(id); err != nil {
		return domain.CostRun{}, fmt.Errorf("%w: cost run %s", domain.ErrNotFound, id)
	}
	run, err := scanRun(c.s.q.QueryRow(ctx, "SELECT "+runCols+" FROM cost_runs WHERE id = $1", id))
	if err != nil {
		return domain.CostRun{}, mapError("cost run "+id, err)
	}
	run.Lines, err = c.linesFor(ctx, run.ID)
	return run, err
}

// linesFor reads a run's lines, joining the element so that a stored run still
// names its elements after somebody renames one.
func (c costing) linesFor(ctx context.Context, runID string) ([]domain.CostLine, error) {
	rows, err := c.s.q.Query(ctx, `SELECT l.element_id, e.code, e.name, e.category, e.driver,
		e.variable, l.standard_rate, l.actual_rate, l.planned_qty, l.actual_qty,
		l.planned_cost, l.actual_cost, l.rate_variance, l.usage_variance, l.total_variance, l.missing
		FROM cost_run_lines l JOIN cost_elements e ON e.id = l.element_id
		WHERE l.run_id = $1 ORDER BY e.category, e.code`, runID)
	if err != nil {
		return nil, mapError("cost run line", err)
	}
	defer rows.Close()

	var out []domain.CostLine
	for rows.Next() {
		var line domain.CostLine
		var category, driver, missing string
		if err := rows.Scan(&line.ElementID, &line.ElementCode, &line.ElementName,
			&category, &driver, &line.Variable,
			&line.StandardRate, &line.ActualRate, &line.PlannedQty, &line.ActualQty,
			&line.PlannedCost, &line.ActualCost, &line.RateVariance, &line.UsageVariance,
			&line.TotalVariance, &missing); err != nil {
			return nil, mapError("cost run line", err)
		}
		line.Category, line.Driver = domain.CostCategory(category), domain.CostDriver(driver)
		line.DriverUnit = domain.DriverUnit(line.Driver)
		if missing != "" {
			line.Missing = strings.Split(missing, "; ")
		}
		out = append(out, line)
	}
	return out, mapError("cost run line", rows.Err())
}

// SaveRun writes the run and replaces its lines. Callers wrap it in InTx, so a
// run whose lines failed half way leaves neither behind.
func (c costing) SaveRun(ctx context.Context, r domain.CostRun, actor string) (domain.CostRun, error) {
	now := nowUTC()
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	totals, err := json.Marshal(r.Totals)
	if err != nil {
		return domain.CostRun{}, fmt.Errorf("%w: cost run totals: %v", domain.ErrValidation, err)
	}

	var version int64
	err = c.s.q.QueryRow(ctx, `INSERT INTO cost_runs
		(id, season_id, version_id, factory_id, code, from_date, to_date, currency,
		 totals, note, created_at, created_by, updated_at, updated_by, row_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$11,$12,1)
		ON CONFLICT (season_id, code) DO UPDATE SET
			version_id = EXCLUDED.version_id, factory_id = EXCLUDED.factory_id,
			from_date = EXCLUDED.from_date, to_date = EXCLUDED.to_date,
			currency = EXCLUDED.currency, totals = EXCLUDED.totals, note = EXCLUDED.note,
			updated_at = EXCLUDED.updated_at, updated_by = EXCLUDED.updated_by,
			row_version = cost_runs.row_version + 1
		RETURNING id, created_at, created_by, row_version`,
		r.ID, r.SeasonID, nu(r.VersionID), r.FactoryID, r.Code,
		nd(r.From), nd(r.To), r.Currency, totals, r.Note, now, actor).
		Scan(&r.ID, &r.CreatedAt, &r.CreatedBy, &version)
	if err != nil {
		return domain.CostRun{}, mapError("cost run "+r.Code, err)
	}
	r.UpdatedAt, r.UpdatedBy, r.RowVersion = now, actor, version

	// A re-run replaces its lines wholesale: an element dropped from the cost
	// structure must disappear from the run rather than linger at its old cost.
	if _, err := c.s.q.Exec(ctx, "DELETE FROM cost_run_lines WHERE run_id = $1", r.ID); err != nil {
		return domain.CostRun{}, mapError("cost run line", err)
	}
	for _, line := range r.Lines {
		if _, err := c.s.q.Exec(ctx, `INSERT INTO cost_run_lines
			(id, run_id, element_id, standard_rate, actual_rate, planned_qty, actual_qty,
			 planned_cost, actual_cost, rate_variance, usage_variance, total_variance, missing)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
			uuid.NewString(), r.ID, line.ElementID, line.StandardRate, line.ActualRate,
			line.PlannedQty, line.ActualQty, line.PlannedCost, line.ActualCost,
			line.RateVariance, line.UsageVariance, line.TotalVariance,
			strings.Join(line.Missing, "; ")); err != nil {
			return domain.CostRun{}, mapError("cost run line "+line.ElementCode, err)
		}
	}
	return r, nil
}
