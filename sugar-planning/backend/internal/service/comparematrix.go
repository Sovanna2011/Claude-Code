package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// Comparing more than two versions at once.
//
// Compare answers "what did changing the recovery assumption do?" - one
// scenario against one baseline. That is not the question a planner asks at a
// review. There they have a budget, two or three what-ifs and the actuals so
// far, and they want them in columns beside each other, because a scenario only
// means anything against the others on the table.
//
// Building this as a second pass over the same aggregation as Compare is
// deliberate: both go through compareValues and compareLabels, so the pairwise
// view and the matrix cannot come to different answers about the same two
// versions. A comparison screen that disagreed with itself would be worse than
// having only one of them.

// MatrixRequest asks for two or more versions side by side.
type MatrixRequest struct {
	// VersionIDs are the columns, in the order they should be shown. A planner
	// decides the order; it is not sorted for them.
	VersionIDs []string `json:"versionIds"`
	// BaselineVersionID is the column the differences are measured from.
	// Empty means the first column, which is the usual reading: budget first,
	// scenarios after it.
	BaselineVersionID string `json:"baselineVersionId,omitempty"`
	// IncludeActual appends the season's actuals container as a further column,
	// so "actual against V1 and V2" is the same request as "V1 against V2".
	// The actuals are read as the ACTUAL series; every plan version is read as
	// PLAN. That is handled by compareValues, not here.
	IncludeActual bool                `json:"includeActual,omitempty"`
	From          domain.BusinessDate `json:"from,omitempty"`
	To            domain.BusinessDate `json:"to,omitempty"`
	// Dimension is DATE, PRODUCT, WAREHOUSE, CHANNEL or PROCESS, as for Compare.
	Dimension string `json:"dimension"`
}

// MatrixColumn is one version in the comparison.
type MatrixColumn struct {
	Version domain.PlanVersion `json:"version"`
	// Series says which series was read for this column: a plan version
	// contributes its PLAN rows and the actuals container its ACTUAL ones.
	// Without it a column of actuals looks like a plan that was never generated.
	Series     domain.Series `json:"series"`
	IsBaseline bool          `json:"isBaseline"`
}

// MatrixRow is one key across every column.
type MatrixRow struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Measure string `json:"measure"`
	// Values, Deltas and DeltaPcts are aligned with Columns, so a cell is
	// Values[i] for Columns[i] and nothing has to be looked up by id.
	Values    []domain.Dec `json:"values"`
	Deltas    []domain.Dec `json:"deltas"`
	DeltaPcts []domain.Dec `json:"deltaPcts"`
}

// MatrixResult is the whole side-by-side comparison.
type MatrixResult struct {
	Dimension string         `json:"dimension"`
	Columns   []MatrixColumn `json:"columns"`
	Rows      []MatrixRow    `json:"rows"`
	Totals    []MatrixRow    `json:"totals"`
	// Assumptions carries only the assumptions that are not the same in every
	// column. A review does not need to be told that the twelve settings
	// nobody touched are still equal; it needs the three that were.
	Assumptions []MatrixRow `json:"assumptions"`
}

// CompareMatrix lays two or more versions of a season side by side.
func (p *Planning) CompareMatrix(ctx context.Context, req MatrixRequest) (MatrixResult, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return MatrixResult{}, err
	}
	if req.Dimension == "" {
		req.Dimension = "DATE"
	}

	ids := dedupeStrings(req.VersionIDs)
	if len(ids) == 0 {
		return MatrixResult{}, fmt.Errorf(
			"%w: name the versions to compare", domain.ErrValidation)
	}

	versions := make([]domain.PlanVersion, 0, len(ids)+1)
	seasonID := ""
	for _, id := range ids {
		v, err := p.versionInScope(ctx, id)
		if err != nil {
			return MatrixResult{}, err
		}
		if seasonID == "" {
			seasonID = v.SeasonID
		} else if v.SeasonID != seasonID {
			return MatrixResult{}, fmt.Errorf(
				"%w: versions from different seasons cannot be compared", domain.ErrValidation)
		}
		versions = append(versions, v)
	}

	if req.IncludeActual {
		actual, err := p.actualsFor(ctx, seasonID)
		if err != nil {
			return MatrixResult{}, err
		}
		// Only if it is not already a column: asking for the actuals twice is a
		// mistake in the request, not two columns of the same thing.
		if actual.ID != "" && !hasVersion(versions, actual.ID) {
			versions = append(versions, actual)
		}
	}
	if len(versions) < 2 {
		return MatrixResult{}, fmt.Errorf(
			"%w: a comparison needs at least two versions", domain.ErrValidation)
	}

	baseline := 0
	if req.BaselineVersionID != "" {
		baseline = -1
		for i, v := range versions {
			if v.ID == req.BaselineVersionID {
				baseline = i
				break
			}
		}
		if baseline < 0 {
			return MatrixResult{}, fmt.Errorf(
				"%w: the baseline must be one of the versions being compared", domain.ErrValidation)
		}
	}

	out := MatrixResult{Dimension: req.Dimension}
	values := make([]map[compareKey]domain.Dec, len(versions))
	for i, v := range versions {
		series := domain.SeriesPlan
		if v.PlanType == domain.PlanTypeActual {
			series = domain.SeriesActual
		}
		out.Columns = append(out.Columns, MatrixColumn{
			Version: v, Series: series, IsBaseline: i == baseline,
		})
		vals, err := p.compareValues(ctx, v, CompareRequest{
			From: req.From, To: req.To, Dimension: req.Dimension,
		})
		if err != nil {
			return MatrixResult{}, err
		}
		values[i] = vals
	}

	labels, err := p.compareLabels(ctx, req.Dimension)
	if err != nil {
		return MatrixResult{}, err
	}

	// Every key any column has a figure for. A key missing from one version is
	// a zero in that column, not a missing row: "V2 plans nothing on this day"
	// is the answer somebody is looking for.
	keys := map[compareKey]bool{}
	for _, vals := range values {
		for k := range vals {
			keys[k] = true
		}
	}
	for k := range keys {
		row := MatrixRow{Key: k.key, Measure: k.measure, Label: labels[k.key]}
		if row.Label == "" {
			row.Label = k.key
		}
		base := domain.RoundQty(values[baseline][k])
		for i := range versions {
			v := domain.RoundQty(values[i][k])
			row.Values = append(row.Values, v)
			row.Deltas = append(row.Deltas, domain.RoundQty(v.Sub(base)))
			row.DeltaPcts = append(row.DeltaPcts, domain.RoundPct(domain.SafePct(v.Sub(base), base)))
		}
		out.Rows = append(out.Rows, row)
	}
	sortMatrixRows(out.Rows)

	out.Totals = matrixTotals(out.Rows, len(versions), baseline)
	out.Assumptions, err = p.matrixAssumptions(ctx, versions, baseline)
	if err != nil {
		return MatrixResult{}, err
	}
	return out, nil
}

// matrixTotals sums each measure down its column.
func matrixTotals(rows []MatrixRow, columns, baseline int) []MatrixRow {
	byMeasure := map[string][]domain.Dec{}
	for _, r := range rows {
		acc, ok := byMeasure[r.Measure]
		if !ok {
			acc = make([]domain.Dec, columns)
		}
		for i := range r.Values {
			acc[i] = acc[i].Add(r.Values[i])
		}
		byMeasure[r.Measure] = acc
	}
	out := make([]MatrixRow, 0, len(byMeasure))
	for measure, acc := range byMeasure {
		row := MatrixRow{Key: "TOTAL", Label: "Total", Measure: measure}
		base := domain.RoundQty(acc[baseline])
		for _, v := range acc {
			v = domain.RoundQty(v)
			row.Values = append(row.Values, v)
			row.Deltas = append(row.Deltas, domain.RoundQty(v.Sub(base)))
			row.DeltaPcts = append(row.DeltaPcts, domain.RoundPct(domain.SafePct(v.Sub(base), base)))
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Measure < out[j].Measure })
	return out
}

// matrixAssumptions reports the assumptions that differ somewhere across the
// set, which is the "what was changed" half of a scenario review.
//
// Only versions that carry assumptions at all take part in deciding whether one
// differs. The actuals container carries none by design - it records what
// happened rather than what was assumed - and counting its absent settings as
// zeros made every assumption look changed. Seventeen rows of "differs" with
// the one a planner actually moved buried among them is the same as reporting
// nothing.
//
// A version that has assumptions but is missing one of them is a different
// case, and still counts: a scenario that dropped a setting has changed it.
func (p *Planning) matrixAssumptions(ctx context.Context, versions []domain.PlanVersion,
	baseline int) ([]MatrixRow, error) {

	perVersion := make([]map[string]domain.Dec, len(versions))
	codes := map[string]string{} // code -> description
	for i, v := range versions {
		rows, err := p.store.Planning().ListAssumptions(ctx, v.ID)
		if err != nil {
			return nil, err
		}
		m := map[string]domain.Dec{}
		for _, a := range rows {
			m[a.Code] = a.Value
			if codes[a.Code] == "" {
				codes[a.Code] = a.Description
			}
		}
		perVersion[i] = m
	}

	var out []MatrixRow
	for code, description := range codes {
		row := MatrixRow{Key: code, Measure: "assumption", Label: description}
		if row.Label == "" {
			row.Label = code
		}
		same := true
		base := perVersion[baseline][code]
		for i := range versions {
			v := perVersion[i][code]
			if len(perVersion[i]) > 0 && !v.Equal(base) {
				same = false
			}
			row.Values = append(row.Values, v)
			row.Deltas = append(row.Deltas, v.Sub(base))
			row.DeltaPcts = append(row.DeltaPcts, domain.RoundPct(domain.SafePct(v.Sub(base), base)))
		}
		if same {
			continue
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

// actualsFor finds a season's actuals container, if it has one.
func (p *Planning) actualsFor(ctx context.Context, seasonID string) (domain.PlanVersion, error) {
	page, err := p.store.Planning().ListVersions(ctx, seasonID, store.ListOptions{Top: 1000})
	if err != nil {
		return domain.PlanVersion{}, err
	}
	for _, v := range page.Items {
		if v.PlanType == domain.PlanTypeActual {
			return v, nil
		}
	}
	return domain.PlanVersion{}, nil
}

func hasVersion(list []domain.PlanVersion, id string) bool {
	for _, v := range list {
		if v.ID == id {
			return true
		}
	}
	return false
}

func dedupeStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func sortMatrixRows(rows []MatrixRow) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Measure != rows[j].Measure {
			return rows[i].Measure < rows[j].Measure
		}
		return rows[i].Key < rows[j].Key
	})
}
