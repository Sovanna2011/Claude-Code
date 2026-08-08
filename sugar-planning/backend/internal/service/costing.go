package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// Costing answers what a ton of sugar costs and why the actual differs from the
// plan.
//
// It reads driver quantities from the same daily rows the dashboard reports on,
// so a cost figure and a tonnage figure can never disagree: if the cost run
// says 800 hours were run, the operations report says the same, because both
// added up the same rows.
type Costing struct {
	store    store.Store
	planning *Planning
	now      func() time.Time
}

// NewCosting builds the service.
func NewCosting(s store.Store, planning *Planning, now func() time.Time) *Costing {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Costing{store: s, planning: planning, now: now}
}

func (c *Costing) audit(ctx context.Context, tx store.Store, entry auditEntry) error {
	return writeAudit(ctx, tx, c.now, entry)
}

// ---------------------------------------------------------------------------
// Cost structure
// ---------------------------------------------------------------------------

// ListElements returns the cost structure.
func (c *Costing) ListElements(ctx context.Context, opts store.ListOptions) (store.Page[domain.CostElement], error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermMasterDataRead); err != nil {
		return store.Page[domain.CostElement]{}, err
	}
	return c.store.Costing().ListElements(ctx, opts)
}

// SaveElement creates or updates a cost element.
func (c *Costing) SaveElement(ctx context.Context, e domain.CostElement) (domain.CostElement, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermMasterDataWrite); err != nil {
		return domain.CostElement{}, err
	}

	verr := &domain.ValidationError{}
	if e.Code == "" {
		verr.Add("code", "REQUIRED", "the element needs a code, for example FUEL")
	}
	if e.Name == "" {
		verr.Add("name", "REQUIRED", "the element needs a name")
	}
	if !domain.ValidCostCategory(e.Category) {
		verr.Add("category", "INVALID", fmt.Sprintf("%q is not a cost category", e.Category))
	}
	if !domain.ValidDriver(e.Driver) {
		verr.Add("driver", "INVALID", fmt.Sprintf(
			"%q is not a cost driver; a cost scales with cane tons, sugar tons, run hours, "+
				"calendar days, or is a fixed seasonal sum", e.Driver))
	}
	if err := verr.OrNil(); err != nil {
		return domain.CostElement{}, err
	}

	var saved domain.CostElement
	err := c.store.InTx(ctx, func(tx store.Store) error {
		var err error
		saved, err = tx.Costing().SaveElement(ctx, e, caller.Username)
		if err != nil {
			return err
		}
		return c.audit(ctx, tx, auditEntry{
			action: actionFor(e.ID == ""), entity: "cost_element", entityID: saved.ID, after: saved,
		})
	})
	return saved, err
}

// ListRates returns the rates the caller may see.
func (c *Costing) ListRates(ctx context.Context, f store.CostFilter) ([]domain.CostRate, error) {
	caller := auth.FromContext(ctx)
	// Rates are commercially sensitive: what a factory pays for cane is not
	// something every plan reader should see. They sit behind the reporting
	// permission the cost controller holds rather than behind plan:read.
	if err := caller.Require(domain.PermCostRead); err != nil {
		return nil, err
	}
	if f.FactoryID != "" {
		if err := caller.RequireFactory(f.FactoryID); err != nil {
			return nil, err
		}
	}
	return c.store.Costing().ListRates(ctx, f)
}

// SaveRate creates or corrects a cost rate.
func (c *Costing) SaveRate(ctx context.Context, r domain.CostRate) (domain.CostRate, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermCostWrite); err != nil {
		return domain.CostRate{}, err
	}
	if err := caller.RequireFactory(r.FactoryID); err != nil {
		return domain.CostRate{}, err
	}

	verr := &domain.ValidationError{}
	if r.ElementID == "" {
		verr.Add("elementId", "REQUIRED", "the rate needs a cost element")
	}
	if !domain.ValidRateType(r.RateType) {
		verr.Add("rateType", "INVALID", "a rate is either STANDARD or ACTUAL")
	}
	if r.Rate.IsNegative() {
		verr.Add("rate", "NEGATIVE", "a rate cannot be negative")
	}
	if len(r.Currency) != 3 {
		verr.Add("currency", "INVALID", "the currency is a three-letter ISO code, for example USD")
	}
	if !r.ValidFrom.Valid() {
		verr.Add("validFrom", "INVALID_DATE", "the rate needs a start date")
	}
	if r.ValidTo != "" && r.ValidTo < r.ValidFrom {
		verr.Add("validTo", "OUT_OF_RANGE", "the end date cannot be before the start date")
	}
	if err := verr.OrNil(); err != nil {
		return domain.CostRate{}, err
	}
	if _, err := c.store.Costing().GetElement(ctx, r.ElementID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.CostRate{}, fmt.Errorf(
				"%w: no such cost element", domain.ErrValidation)
		}
		return domain.CostRate{}, err
	}

	var saved domain.CostRate
	err := c.store.InTx(ctx, func(tx store.Store) error {
		var err error
		saved, err = tx.Costing().SaveRate(ctx, r, caller.Username)
		if err != nil {
			return err
		}
		return c.audit(ctx, tx, auditEntry{
			action: actionFor(r.ID == ""), entity: "cost_rate", entityID: saved.ID, after: saved,
			reason: fmt.Sprintf("%s rate from %s", r.RateType, r.ValidFrom),
		})
	})
	return saved, err
}

// DeleteRate removes a rate entered in error.
func (c *Costing) DeleteRate(ctx context.Context, id string) error {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermCostWrite); err != nil {
		return err
	}
	return c.store.InTx(ctx, func(tx store.Store) error {
		if err := tx.Costing().DeleteRate(ctx, id); err != nil {
			return err
		}
		return c.audit(ctx, tx, auditEntry{
			action: "DELETE", entity: "cost_rate", entityID: id,
		})
	})
}

// ListExchangeRates returns the currency quotations.
func (c *Costing) ListExchangeRates(ctx context.Context) ([]domain.ExchangeRate, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermCostRead); err != nil {
		return nil, err
	}
	return c.store.Costing().ListExchangeRates(ctx)
}

// SaveExchangeRate records a currency quotation.
func (c *Costing) SaveExchangeRate(ctx context.Context, r domain.ExchangeRate) (domain.ExchangeRate, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermCostWrite); err != nil {
		return domain.ExchangeRate{}, err
	}

	verr := &domain.ValidationError{}
	if len(r.FromCurrency) != 3 || len(r.ToCurrency) != 3 {
		verr.Add("fromCurrency", "INVALID", "both currencies are three-letter ISO codes")
	}
	if r.FromCurrency == r.ToCurrency {
		verr.Add("toCurrency", "SAME_CURRENCY",
			"a quotation converts between two different currencies")
	}
	if r.Rate.LessThanOrEqual(domain.Zero) {
		verr.Add("rate", "NOT_POSITIVE", "an exchange rate must be more than zero")
	}
	if !r.ValidFrom.Valid() {
		verr.Add("validFrom", "INVALID_DATE", "the quotation needs a start date")
	}
	if err := verr.OrNil(); err != nil {
		return domain.ExchangeRate{}, err
	}

	var saved domain.ExchangeRate
	err := c.store.InTx(ctx, func(tx store.Store) error {
		var err error
		saved, err = tx.Costing().SaveExchangeRate(ctx, r, caller.Username)
		if err != nil {
			return err
		}
		return c.audit(ctx, tx, auditEntry{
			action: actionFor(r.ID == ""), entity: "exchange_rate", entityID: saved.ID, after: saved,
		})
	})
	return saved, err
}

// ---------------------------------------------------------------------------
// The cost run
// ---------------------------------------------------------------------------

// CostRunRequest asks for a costing over a range of days.
type CostRunRequest struct {
	SeasonID  string              `json:"seasonId"`
	VersionID string              `json:"versionId,omitempty"`
	From      domain.BusinessDate `json:"from,omitempty"`
	To        domain.BusinessDate `json:"to,omitempty"`
	// Currency is what the run reports in. Empty uses the company's currency.
	Currency string `json:"currency,omitempty"`
	// Save stores the run under Code so the figure can be reproduced later.
	Save bool   `json:"save,omitempty"`
	Code string `json:"code,omitempty"`
	Note string `json:"note,omitempty"`
}

// CostRunResult is a costing, with the drivers it was built from so that a
// reader can check the arithmetic without opening another screen.
type CostRunResult struct {
	Run      domain.CostRun      `json:"run"`
	Lines    []domain.CostLine   `json:"lines"`
	Totals   domain.CostTotals   `json:"totals"`
	Planned  Drivers             `json:"plannedDrivers"`
	Actual   Drivers             `json:"actualDrivers"`
	Warnings []domain.Alert      `json:"warnings,omitempty"`
	Currency string              `json:"currency"`
	From     domain.BusinessDate `json:"from"`
	To       domain.BusinessDate `json:"to"`
	Saved    bool                `json:"saved"`
}

// Drivers is the measured volumes a run multiplied its rates by, reported so
// the arithmetic is checkable.
type Drivers struct {
	CaneTons     domain.Dec `json:"caneTons"`
	SugarTons    domain.Dec `json:"sugarTons"`
	RunHours     domain.Dec `json:"runHours"`
	CalendarDays domain.Dec `json:"calendarDays"`
}

func (d Drivers) toDomain() domain.DriverQuantities {
	return domain.DriverQuantities{
		CaneTons: d.CaneTons, SugarTons: d.SugarTons,
		RunHours: d.RunHours, CalendarDays: d.CalendarDays,
	}
}

// Run costs a period.
func (c *Costing) Run(ctx context.Context, req CostRunRequest) (CostRunResult, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermCostRead); err != nil {
		return CostRunResult{}, err
	}

	season, err := c.planning.GetSeason(ctx, req.SeasonID)
	if err != nil {
		return CostRunResult{}, err
	}

	plan, actual, err := c.resolveVersions(ctx, season.ID, req.VersionID)
	if err != nil {
		return CostRunResult{}, err
	}

	from, to := req.From, req.To
	if from == "" {
		from = season.StartDate
	}
	if to == "" {
		// The default range ends on the last day anybody has recorded, not on
		// the last day of the season.
		//
		// Costing a whole season's plan against a fortnight of actuals produces
		// a variance of minus ninety per cent and a cost per ton of zero, which
		// is arithmetically right and completely useless: it says only that the
		// season has not finished. Ending the range where the actuals end makes
		// the two sides cover the same days, which is the comparison somebody
		// opening this screen is asking for. A season with nothing recorded
		// yet falls back to its own end date and reports the plan alone.
		to = c.lastRecordedDay(ctx, actual.ID, season.FactoryID, from, season.EndDate)
	}
	if to == "" {
		to = season.EndDate
	}
	if to == "" {
		to = domain.NewBusinessDate(c.now())
	}
	if !from.Valid() || !to.Valid() {
		return CostRunResult{}, fmt.Errorf(
			"%w: the run needs a valid from and to date", domain.ErrValidation)
	}
	if to < from {
		return CostRunResult{}, fmt.Errorf(
			"%w: the end of the range is before its start", domain.ErrValidation)
	}

	currency := req.Currency
	if currency == "" {
		company, err := c.store.MasterData().Companies().Get(ctx, season.CompanyID)
		if err != nil {
			return CostRunResult{}, err
		}
		currency = company.Currency
	}

	planned, err := c.drivers(ctx, plan.ID, season.FactoryID, from, to, domain.SeriesPlan)
	if err != nil {
		return CostRunResult{}, err
	}
	measured, err := c.drivers(ctx, actual.ID, season.FactoryID, from, to, domain.SeriesActual)
	if err != nil {
		return CostRunResult{}, err
	}

	elements, err := c.store.Costing().ListElements(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return CostRunResult{}, err
	}
	rates, err := c.store.Costing().ListRates(ctx, store.CostFilter{
		FactoryID: season.FactoryID, SeasonID: season.ID,
	})
	if err != nil {
		return CostRunResult{}, err
	}
	exchange, err := c.store.Costing().ListExchangeRates(ctx)
	if err != nil {
		return CostRunResult{}, err
	}

	// The rates are read as at the last day of the range, so a price change
	// inside the range applies to the whole of it. Splitting the range is how a
	// site prices either side of a change separately, and saying so here is
	// better than pretending a single run can do both.
	output := domain.CalculateCost(domain.CostInput{
		Elements: elements.Items, Rates: rates, Exchange: exchange,
		Planned: planned.toDomain(), Actual: measured.toDomain(),
		On: to, Currency: currency,
	})

	result := CostRunResult{
		Lines: output.Lines, Totals: output.Totals, Warnings: output.Warnings,
		Planned: planned, Actual: measured,
		Currency: currency, From: from, To: to,
	}

	if !req.Save {
		return result, nil
	}
	if err := caller.Require(domain.PermCostWrite); err != nil {
		return CostRunResult{}, err
	}
	code := req.Code
	if code == "" {
		code = string(from) + "_" + string(to)
	}

	run := domain.CostRun{
		SeasonID: season.ID, VersionID: plan.ID, FactoryID: season.FactoryID,
		Code: code, From: from, To: to, Currency: currency,
		Totals: output.Totals, Lines: output.Lines, Note: req.Note,
	}
	err = c.store.InTx(ctx, func(tx store.Store) error {
		saved, err := tx.Costing().SaveRun(ctx, run, caller.Username)
		if err != nil {
			return err
		}
		result.Run, result.Saved = saved, true
		if err := c.audit(ctx, tx, auditEntry{
			action: "COST_RUN", entity: "cost_run", entityID: saved.ID, after: output.Totals,
			reason: fmt.Sprintf("costed %s to %s in %s", from, to, currency),
		}); err != nil {
			return err
		}
		// Only a saved run is published. A planner trying figures on screen is
		// not making a statement the finance system should record.
		return emit(ctx, tx, domain.TopicCostRunCompleted, CostRunEvent{
			RunID: saved.ID, SeasonID: season.ID, VersionID: plan.ID, FactoryID: season.FactoryID,
			From: from, To: to, Currency: currency,
			PlannedCost: output.Totals.PlannedCost, ActualCost: output.Totals.ActualCost,
			Variance: output.Totals.TotalVariance, RunBy: caller.Username,
		})
	})
	if err != nil {
		return CostRunResult{}, err
	}
	return result, nil
}

// drivers adds up the measured volumes for a version over a range.
//
// They come from the same daily rows the dashboard reports on, so a cost figure
// and a tonnage figure can never disagree: both added up the same rows.
func (c *Costing) drivers(ctx context.Context, versionID, factoryID string,
	from, to domain.BusinessDate, series domain.Series,
) (Drivers, error) {

	out := Drivers{
		CaneTons: domain.Zero, SugarTons: domain.Zero,
		RunHours: domain.Zero, CalendarDays: domain.Zero,
	}
	if versionID == "" {
		return out, nil
	}

	filter := store.PlanFilter{
		VersionIDs: []string{versionID}, FactoryID: factoryID,
		From: from, To: to, Series: series,
	}

	cane, err := c.store.Planning().ListCane(ctx, filter)
	if err != nil {
		return out, err
	}
	days := map[domain.BusinessDate]bool{}
	for _, row := range cane {
		out.CaneTons = out.CaneTons.Add(row.CaneCrushed)
		out.RunHours = out.RunHours.Add(row.RunHours())
		days[row.BusinessDate] = true
	}

	products, err := c.store.Planning().ListProducts(ctx, filter)
	if err != nil {
		return out, err
	}
	finished, err := c.finishedProducts(ctx)
	if err != nil {
		return out, err
	}
	for _, row := range products {
		// Only finished goods count towards the tonnage a cost is spread over.
		// Counting raw sugar as well would divide the cost by roughly twice the
		// output and halve the cost per ton.
		if finished[row.ProductID] {
			out.SugarTons = out.SugarTons.Add(row.Quantity)
		}
	}

	// The calendar day count is the days the range covers, not the days that
	// have rows: a salaried crew is paid on a day the mill stood idle, which is
	// exactly the day with no production row.
	out.CalendarDays = domain.DI(int64(from.DaysBetween(to) + 1))
	if series == domain.SeriesActual && len(days) > 0 {
		// For actuals the elapsed days are what has been recorded, so a run in
		// the middle of a season is not charged for days that have not happened.
		out.CalendarDays = domain.DI(int64(len(days)))
	}

	out.CaneTons = domain.RoundQty(out.CaneTons)
	out.SugarTons = domain.RoundQty(out.SugarTons)
	out.RunHours = domain.RoundRate(out.RunHours)
	return out, nil
}

// lastRecordedDay finds the latest day the actuals container has a cane row
// for, so a run's default range ends where the recording does.
func (c *Costing) lastRecordedDay(ctx context.Context, actualID, factoryID string,
	from, seasonEnd domain.BusinessDate,
) domain.BusinessDate {

	if actualID == "" {
		return seasonEnd
	}
	rows, err := c.store.Planning().ListCane(ctx, store.PlanFilter{
		VersionIDs: []string{actualID}, FactoryID: factoryID,
		From: from, To: seasonEnd, Series: domain.SeriesActual,
	})
	if err != nil || len(rows) == 0 {
		// A season nobody has posted to yet reports the plan alone. Failing the
		// run because the actuals could not be read would be worse than
		// reporting the plan, which is all there is to report.
		return seasonEnd
	}
	last := rows[0].BusinessDate
	for _, row := range rows {
		if row.BusinessDate > last {
			last = row.BusinessDate
		}
	}
	return last
}

func (c *Costing) finishedProducts(ctx context.Context) (map[string]bool, error) {
	page, err := c.store.MasterData().Products().List(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, p := range page.Items {
		if p.IsFinished {
			out[p.ID] = true
		}
	}
	return out, nil
}

// resolveVersions finds the plan to cost against and the actuals container to
// measure. It mirrors the dashboard's rule so the two screens never report on
// different versions.
func (c *Costing) resolveVersions(ctx context.Context, seasonID, requested string) (plan, actual domain.PlanVersion, err error) {
	page, err := c.store.Planning().ListVersions(ctx, seasonID, store.ListOptions{Top: 1000})
	if err != nil {
		return plan, actual, err
	}
	for _, v := range page.Items {
		if v.PlanType == domain.PlanTypeActual {
			actual = v
			continue
		}
		if requested != "" && v.ID == requested {
			plan = v
		}
		if requested == "" && v.Status == domain.StatusReleased {
			plan = v
		}
	}
	if plan.ID == "" && requested == "" {
		for _, v := range page.Items {
			if v.PlanType != domain.PlanTypeActual && v.VersionNo >= plan.VersionNo {
				plan = v
			}
		}
	}
	if plan.ID == "" {
		return plan, actual, fmt.Errorf(
			"%w: the season has no plan version to cost against", domain.ErrNotFound)
	}
	return plan, actual, nil
}

// ListRuns returns the saved runs of a season.
func (c *Costing) ListRuns(ctx context.Context, f store.CostFilter) (store.Page[domain.CostRun], error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermCostRead); err != nil {
		return store.Page[domain.CostRun]{}, err
	}
	page, err := c.store.Costing().ListRuns(ctx, f)
	if err != nil {
		return page, err
	}
	filtered := page.Items[:0]
	for _, r := range page.Items {
		if caller.CanSeeFactory(r.FactoryID) {
			filtered = append(filtered, r)
		}
	}
	page.Items = filtered
	page.Count = len(filtered)
	return page, nil
}

// GetRun reads one saved run with its lines.
func (c *Costing) GetRun(ctx context.Context, id string) (domain.CostRun, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermCostRead); err != nil {
		return domain.CostRun{}, err
	}
	run, err := c.store.Costing().GetRun(ctx, id)
	if err != nil {
		return domain.CostRun{}, err
	}
	if err := caller.RequireFactory(run.FactoryID); err != nil {
		return domain.CostRun{}, err
	}
	return run, nil
}
