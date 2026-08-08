package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// Cane supply planning: where the season's cane comes from.
//
// The crushing plan says how much cane the mill will put through each day. This
// says which farms it arrives from, when each block is cut, and whether the
// trucks exist to move it - and then reconciles the two, which is the point.
// A cane target with no sources behind it is a number somebody typed.

// SupplyPlan is the whole cane-supply picture for one plan version.
type SupplyPlan struct {
	VersionID string                      `json:"versionId"`
	Entries   []SupplyLine                `json:"entries"`
	Reconcile domain.SupplyReconciliation `json:"reconciliation"`
	Warnings  []domain.Alert              `json:"warnings"`
}

// SupplyLine is one commitment with its source resolved, so a screen does not
// have to join two lists to show a row.
type SupplyLine struct {
	domain.CaneSupplyEntry
	SourceCode string            `json:"sourceCode"`
	SourceName string            `json:"sourceName"`
	SourceType domain.SourceType `json:"sourceType"`
	Zone       string            `json:"zone,omitempty"`
	Hectares   domain.Dec        `json:"hectares"`
	// ExpectedTons is what the land should yield, against which the commitment
	// is a promise that may or may not be keepable.
	ExpectedTons domain.Dec `json:"expectedTons"`
	// RequiredDailyRate and TransportCapacity are the pair that says whether
	// the haulage exists for this commitment.
	RequiredDailyRate domain.Dec `json:"requiredDailyRateTons"`
	TransportCapacity domain.Dec `json:"transportCapacityTons"`
	HarvestDays       int        `json:"harvestDays"`
}

// SupplyPlan reads the commitments for a version and reconciles them against
// the season's cane target.
func (p *Planning) SupplyPlan(ctx context.Context, versionID string) (SupplyPlan, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return SupplyPlan{}, err
	}
	version, err := p.versionInScope(ctx, versionID)
	if err != nil {
		return SupplyPlan{}, err
	}

	entries, err := p.store.Planning().ListSupply(ctx, versionID)
	if err != nil {
		return SupplyPlan{}, err
	}
	sources, err := p.caneSourceIndex(ctx)
	if err != nil {
		return SupplyPlan{}, err
	}

	out := SupplyPlan{VersionID: versionID}
	for _, e := range entries {
		line := SupplyLine{
			CaneSupplyEntry:   e,
			RequiredDailyRate: e.RequiredDailyRate(),
			HarvestDays:       e.HarvestDays(),
		}
		if s, ok := sources[e.SourceID]; ok {
			line.SourceCode, line.SourceName, line.SourceType = s.Code, s.Name, s.Type
			line.Zone, line.Hectares = s.Zone, s.Hectares
			line.ExpectedTons = s.ExpectedTons()
			line.TransportCapacity = s.DailyTransportCapacity()
		}
		out.Entries = append(out.Entries, line)
	}

	stored, err := p.store.Planning().ListAssumptions(ctx, version.ID)
	if err != nil {
		return SupplyPlan{}, err
	}
	assumptions := map[string]domain.Dec{}
	for _, a := range stored {
		assumptions[a.Code] = a.Value
	}
	out.Reconcile = domain.ReconcileSupply(assumptions[domain.AsmCaneTarget], entries, sources)
	out.Warnings = domain.SupplyWarnings(out.Reconcile, entries, sources,
		assumptions[domain.AsmSupplyTolerancePct])
	return out, nil
}

// SaveSupplyEntry records what one source is committed to.
func (p *Planning) SaveSupplyEntry(ctx context.Context, e domain.CaneSupplyEntry) (domain.CaneSupplyEntry, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanWrite); err != nil {
		return domain.CaneSupplyEntry{}, err
	}
	v, err := p.versionInScope(ctx, e.VersionID)
	if err != nil {
		return domain.CaneSupplyEntry{}, err
	}
	if !v.IsEditable() {
		return domain.CaneSupplyEntry{}, fmt.Errorf("%w: version %s is %s",
			domain.ErrLocked, v.Code, v.Status)
	}
	if err := e.Validate(); err != nil {
		return domain.CaneSupplyEntry{}, err
	}
	// The source has to exist and belong to a factory this caller can see,
	// otherwise a commitment is a foreign key to somebody else's farm.
	source, err := p.store.MasterData().CaneSources().Get(ctx, e.SourceID)
	if err != nil {
		return domain.CaneSupplyEntry{}, err
	}
	if err := caller.RequireFactory(source.FactoryID); err != nil {
		return domain.CaneSupplyEntry{}, err
	}

	var saved domain.CaneSupplyEntry
	err = p.store.InTx(ctx, func(tx store.Store) error {
		var err error
		saved, err = tx.Planning().SaveSupply(ctx, e, caller.Username)
		if err != nil {
			return err
		}
		return writeAudit(ctx, tx, p.now, auditEntry{
			action: "SAVE_SUPPLY", entity: "cane_supply_entry", entityID: saved.ID, after: saved,
		})
	})
	return saved, err
}

// DeleteSupplyEntry removes a commitment from a version.
func (p *Planning) DeleteSupplyEntry(ctx context.Context, versionID, id string) error {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanWrite); err != nil {
		return err
	}
	v, err := p.versionInScope(ctx, versionID)
	if err != nil {
		return err
	}
	if !v.IsEditable() {
		return fmt.Errorf("%w: version %s is %s", domain.ErrLocked, v.Code, v.Status)
	}
	return p.store.InTx(ctx, func(tx store.Store) error {
		if err := tx.Planning().DeleteSupply(ctx, id); err != nil {
			return err
		}
		return writeAudit(ctx, tx, p.now, auditEntry{
			action: "DELETE_SUPPLY", entity: "cane_supply_entry", entityID: id,
		})
	})
}

// GenerateSupplySchedule turns the commitments into a delivery schedule: one
// row per source per day of its harvest window.
//
// The tonnage is spread evenly across the window and the remainder lands on the
// last day, so the rows sum to exactly the commitment rather than to the
// commitment minus a rounding crumb. That is the same rule the crushing plan
// follows, and it is what lets the two be compared at all.
func (p *Planning) GenerateSupplySchedule(ctx context.Context, versionID string) (SupplyScheduleResult, error) {
	caller := auth.FromContext(ctx)
	// Rebuilding the delivery schedule replaces every row of it, so it is the
	// generating right rather than the row-editing one, for the same reason
	// Generate is.
	if err := caller.Require(domain.PermPlanGenerate); err != nil {
		return SupplyScheduleResult{}, err
	}
	version, err := p.versionInScope(ctx, versionID)
	if err != nil {
		return SupplyScheduleResult{}, err
	}
	if !version.IsEditable() {
		return SupplyScheduleResult{}, fmt.Errorf("%w: version %s is %s",
			domain.ErrLocked, version.Code, version.Status)
	}
	season, err := p.store.Planning().GetSeason(ctx, version.SeasonID)
	if err != nil {
		return SupplyScheduleResult{}, err
	}
	entries, err := p.store.Planning().ListSupply(ctx, versionID)
	if err != nil {
		return SupplyScheduleResult{}, err
	}
	sources, err := p.caneSourceIndex(ctx)
	if err != nil {
		return SupplyScheduleResult{}, err
	}

	var rows []domain.DailyCaneSupply
	result := SupplyScheduleResult{VersionID: versionID}
	for _, e := range entries {
		source, ok := sources[e.SourceID]
		if !ok {
			continue
		}
		days := e.HarvestDays()
		if days == 0 || e.CommittedTons.LessThanOrEqual(domain.Zero) {
			continue
		}
		per := domain.RoundQty(e.CommittedTons.Div(domain.DI(int64(days))))
		running := domain.Zero
		for d := 0; d < days; d++ {
			tons := per
			if d == days-1 {
				// The last day absorbs the rounding, so the schedule sums to
				// the commitment exactly.
				tons = domain.RoundQty(e.CommittedTons.Sub(running))
			}
			running = running.Add(tons)
			date := e.HarvestFrom.AddDays(d)
			rows = append(rows, domain.DailyCaneSupply{
				VersionID: versionID, SourceID: e.SourceID, FactoryID: season.FactoryID,
				BusinessDate: date, Series: domain.SeriesPlan, Tons: tons,
				Trips:  domain.TripsFor(tons, source.TruckCapacityTons),
				PolPct: source.ExpectedPolPct,
			})
		}
		result.Sources++
	}
	result.Rows = len(rows)

	if len(rows) == 0 {
		return result, nil
	}
	err = p.store.InTx(ctx, func(tx store.Store) error {
		n, err := tx.Planning().UpsertCaneSupply(ctx, rows, caller.Username)
		if err != nil {
			return err
		}
		result.Rows = n
		return writeAudit(ctx, tx, p.now, auditEntry{
			action: "GENERATE_SUPPLY", entity: "plan_version", entityID: versionID,
			after: result,
		})
	})
	return result, err
}

// SupplyScheduleResult reports what the schedule run wrote.
type SupplyScheduleResult struct {
	VersionID string `json:"versionId"`
	Sources   int    `json:"sources"`
	Rows      int    `json:"rows"`
}

// ListCaneSupply reads the delivery schedule or the deliveries recorded
// against it.
func (p *Planning) ListCaneSupply(ctx context.Context, f store.PlanFilter) ([]domain.DailyCaneSupply, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return nil, err
	}
	if len(f.VersionIDs) > 0 {
		for _, id := range f.VersionIDs {
			if _, err := p.versionInScope(ctx, id); err != nil {
				return nil, err
			}
		}
	}
	return p.store.Planning().ListCaneSupply(ctx, f)
}

// UpsertCaneSupply records deliveries at the gate.
//
// An ACTUAL row needs the cane permission rather than the planning one: a
// weighbridge clerk records what arrived, and a planner does not.
func (p *Planning) UpsertCaneSupply(ctx context.Context, versionID string,
	rows []domain.DailyCaneSupply) (UpsertResult, error) {

	caller := auth.FromContext(ctx)
	version, err := p.versionInScope(ctx, versionID)
	if err != nil {
		return UpsertResult{}, err
	}

	series := domain.SeriesPlan
	for _, r := range rows {
		if r.Series == domain.SeriesActual {
			series = domain.SeriesActual
		}
	}
	if series == domain.SeriesActual {
		if err := caller.Require(domain.PermActualCane); err != nil {
			return UpsertResult{}, err
		}
	} else if err := caller.Require(domain.PermPlanWrite); err != nil {
		return UpsertResult{}, err
	}

	season, err := p.store.Planning().GetSeason(ctx, version.SeasonID)
	if err != nil {
		return UpsertResult{}, err
	}

	out := UpsertResult{}
	sound := make([]domain.DailyCaneSupply, 0, len(rows))
	for i, r := range rows {
		r.VersionID, r.FactoryID = versionID, season.FactoryID
		if r.Series == "" {
			r.Series = domain.SeriesPlan
		}
		if err := r.Validate(); err != nil {
			out.Rejected++
			out.Issues = append(out.Issues, supplyIssues(i, err)...)
			continue
		}
		if err := domain.CheckWritable(version, r.BusinessDate, r.Series); err != nil {
			out.Rejected++
			out.Issues = append(out.Issues, RowIssue{Row: i, Field: "businessDate",
				Code: "LOCKED", Message: err.Error()})
			continue
		}
		sound = append(sound, r)
	}
	if out.Rejected > 0 {
		return out, &domain.ValidationError{Errors: toFieldErrors(out.Issues)}
	}
	if len(sound) == 0 {
		return out, nil
	}

	err = p.store.InTx(ctx, func(tx store.Store) error {
		n, err := tx.Planning().UpsertCaneSupply(ctx, sound, caller.Username)
		if err != nil {
			return err
		}
		out.Accepted = n
		return writeAudit(ctx, tx, p.now, auditEntry{
			action: "UPSERT_CANE_SUPPLY", entity: "plan_version", entityID: versionID,
			after: map[string]any{"rows": n, "series": string(series)},
		})
	})
	return out, err
}

// supplyIssues turns a row's validation error into the per-row, per-field form
// the planning grid highlights cells with.
func supplyIssues(row int, err error) []RowIssue {
	var v *domain.ValidationError
	if !errors.As(err, &v) {
		return []RowIssue{{Row: row, Field: "", Code: "INVALID", Message: err.Error()}}
	}
	out := make([]RowIssue, 0, len(v.Errors))
	for _, e := range v.Errors {
		out = append(out, RowIssue{Row: row, Field: e.Field, Code: e.Code, Message: e.Message})
	}
	return out
}

// caneSourceIndex reads every cane source the caller can see, keyed by id.
func (p *Planning) caneSourceIndex(ctx context.Context) (map[string]domain.CaneSource, error) {
	page, err := p.store.MasterData().CaneSources().List(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return nil, err
	}
	out := make(map[string]domain.CaneSource, len(page.Items))
	for _, s := range page.Items {
		out[s.ID] = s
	}
	return out, nil
}
