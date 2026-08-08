package service_test

import (
	"testing"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
)

// Comparing more than two versions.
//
// The thing worth protecting here is that the matrix and the pairwise
// comparison agree. They read the same aggregation on purpose, and a review
// screen that disagreed with the scenario screen about the same two versions
// would be worse than having only one of them.

// scenario copies the seeded budget and regenerates it with an override, which
// is how a planner makes V2.
func scenario(t *testing.T, h *harness, code string, overrides map[string]domain.Dec) domain.PlanVersion {
	t.Helper()
	ctx := h.as(auth.RoleProductionPlanner)
	v, err := h.planning.CopyVersion(ctx, service.CopyRequest{
		SourceVersionID: h.seeded.BudgetID, Code: code,
		Description: "scenario " + code, PlanType: domain.PlanTypeForecast,
		AssumptionOverrides: overrides,
	})
	if err != nil {
		t.Fatalf("copy %s: %v", code, err)
	}
	if _, err := h.planning.Generate(ctx, v.ID, service.GenerateRequest{Replace: true}); err != nil {
		t.Fatalf("generate %s: %v", code, err)
	}
	return v
}

func TestTheMatrixAgreesWithThePairwiseComparison(t *testing.T) {
	h := newHarness(t, 0)
	ctx := h.as(auth.RoleProductionPlanner)
	v2 := scenario(t, h, "V2", map[string]domain.Dec{
		string(domain.AsmRecoveryPct): domain.D("9.5"),
	})

	pair, err := h.planning.Compare(ctx, service.CompareRequest{
		BaseVersionID: h.seeded.BudgetID, OtherVersionID: v2.ID, Dimension: "PRODUCT",
	})
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	matrix, err := h.planning.CompareMatrix(ctx, service.MatrixRequest{
		VersionIDs: []string{h.seeded.BudgetID, v2.ID}, Dimension: "PRODUCT",
	})
	if err != nil {
		t.Fatalf("matrix: %v", err)
	}

	if len(matrix.Columns) != 2 {
		t.Fatalf("%d columns, want 2", len(matrix.Columns))
	}
	if !matrix.Columns[0].IsBaseline || matrix.Columns[1].IsBaseline {
		t.Error("with no baseline named, the first column is the baseline")
	}

	// Every row of the pair has to appear in the matrix with the same figures.
	type cell struct{ key, measure string }
	byKey := map[cell]service.MatrixRow{}
	for _, r := range matrix.Rows {
		byKey[cell{r.Key, r.Measure}] = r
	}
	if len(byKey) != len(pair.Rows) {
		t.Errorf("matrix has %d rows, the pair has %d", len(byKey), len(pair.Rows))
	}
	for _, p := range pair.Rows {
		m, ok := byKey[cell{p.Key, p.Measure}]
		if !ok {
			t.Errorf("%s/%s is in the pair and not the matrix", p.Key, p.Measure)
			continue
		}
		if !m.Values[0].Equal(p.BaseValue) || !m.Values[1].Equal(p.OtherValue) {
			t.Errorf("%s/%s: matrix %s|%s, pair %s|%s",
				p.Key, p.Measure, m.Values[0], m.Values[1], p.BaseValue, p.OtherValue)
		}
		if !m.Deltas[1].Equal(p.Delta) {
			t.Errorf("%s/%s: matrix delta %s, pair delta %s", p.Key, p.Measure, m.Deltas[1], p.Delta)
		}
	}
}

func TestAThirdVersionIsJustAnotherColumn(t *testing.T) {
	h := newHarness(t, 0)
	ctx := h.as(auth.RoleProductionPlanner)
	v2 := scenario(t, h, "V2", map[string]domain.Dec{
		string(domain.AsmRecoveryPct): domain.D("9.5"),
	})
	v3 := scenario(t, h, "V3", map[string]domain.Dec{
		string(domain.AsmCaneTarget): domain.D("2000000"),
	})

	m, err := h.planning.CompareMatrix(ctx, service.MatrixRequest{
		VersionIDs: []string{h.seeded.BudgetID, v2.ID, v3.ID}, Dimension: "PROCESS",
	})
	if err != nil {
		t.Fatalf("matrix: %v", err)
	}
	if len(m.Columns) != 3 {
		t.Fatalf("%d columns, want 3", len(m.Columns))
	}
	for i, r := range m.Rows {
		if len(r.Values) != 3 || len(r.Deltas) != 3 {
			t.Fatalf("row %d has %d values and %d deltas across 3 columns",
				i, len(r.Values), len(r.Deltas))
		}
		// The baseline's own delta is zero, by construction.
		if !r.Deltas[0].Equal(domain.Zero) {
			t.Errorf("%s: the baseline differs from itself by %s", r.Key, r.Deltas[0])
		}
	}

	// V3 crushes 300,000 t less cane, and the cane row must say so.
	var cane service.MatrixRow
	for _, r := range m.Rows {
		if r.Measure == "caneCrushed" {
			cane = r
		}
	}
	if len(cane.Values) == 0 {
		t.Fatal("no cane row in the comparison")
	}
	if !cane.Deltas[2].Equal(domain.D("-300000.000")) {
		t.Errorf("V3 cane delta = %s, want -300,000", cane.Deltas[2])
	}
}

func TestTheActualsAreJustAnotherColumn(t *testing.T) {
	// The point of the whole exercise: "actual against V1 and V2" has to be the
	// same request as "V1 against V2", or a planner needs two screens to ask
	// one question.
	h := newHarness(t, 14)
	ctx := h.as(auth.RoleProductionPlanner)
	v2 := scenario(t, h, "V2", map[string]domain.Dec{
		string(domain.AsmRecoveryPct): domain.D("10.5"),
	})

	m, err := h.planning.CompareMatrix(ctx, service.MatrixRequest{
		VersionIDs:    []string{h.seeded.BudgetID, v2.ID},
		IncludeActual: true, Dimension: "PROCESS",
		From: "2026-12-01", To: "2026-12-14",
	})
	if err != nil {
		t.Fatalf("matrix: %v", err)
	}
	if len(m.Columns) != 3 {
		t.Fatalf("%d columns, want the two plans and the actuals", len(m.Columns))
	}
	last := m.Columns[2]
	if last.Version.PlanType != domain.PlanTypeActual {
		t.Errorf("the appended column is %s, want the actuals container", last.Version.PlanType)
	}
	// And it must be read as ACTUAL. Read as PLAN the column would be empty and
	// look like a plan nobody generated.
	if last.Series != domain.SeriesActual {
		t.Errorf("the actuals column was read as %s", last.Series)
	}
	var cane service.MatrixRow
	for _, r := range m.Rows {
		if r.Measure == "caneCrushed" {
			cane = r
		}
	}
	if len(cane.Values) != 3 || cane.Values[2].IsZero() {
		t.Errorf("the actuals column has no cane in it: %v", cane.Values)
	}
}

func TestAskingForTheActualsTwiceGivesOneColumn(t *testing.T) {
	h := newHarness(t, 14)
	ctx := h.as(auth.RoleProductionPlanner)
	m, err := h.planning.CompareMatrix(ctx, service.MatrixRequest{
		VersionIDs:    []string{h.seeded.BudgetID, h.seeded.ActualID},
		IncludeActual: true,
	})
	if err != nil {
		t.Fatalf("matrix: %v", err)
	}
	if len(m.Columns) != 2 {
		t.Errorf("%d columns; naming the actuals and also asking for them is one column", len(m.Columns))
	}
}

func TestTheBaselineIsChosenNotAssumed(t *testing.T) {
	h := newHarness(t, 0)
	ctx := h.as(auth.RoleProductionPlanner)
	v2 := scenario(t, h, "V2", map[string]domain.Dec{
		string(domain.AsmCaneTarget): domain.D("2000000"),
	})

	m, err := h.planning.CompareMatrix(ctx, service.MatrixRequest{
		VersionIDs:        []string{h.seeded.BudgetID, v2.ID},
		BaselineVersionID: v2.ID, Dimension: "PROCESS",
	})
	if err != nil {
		t.Fatalf("matrix: %v", err)
	}
	if m.Columns[0].IsBaseline || !m.Columns[1].IsBaseline {
		t.Fatal("V2 was named as the baseline")
	}
	// Measured from V2, the budget is 300,000 t of cane *more*.
	for _, r := range m.Rows {
		if r.Measure == "caneCrushed" {
			if !r.Deltas[0].Equal(domain.D("300000.000")) {
				t.Errorf("budget against V2 = %s, want +300,000", r.Deltas[0])
			}
			if !r.Deltas[1].Equal(domain.Zero) {
				t.Errorf("V2 differs from itself by %s", r.Deltas[1])
			}
		}
	}
}

func TestOnlyTheAssumptionsThatChangedAreReported(t *testing.T) {
	// A review does not need to be told that the twelve settings nobody touched
	// are still equal. It needs the one that was.
	h := newHarness(t, 0)
	ctx := h.as(auth.RoleProductionPlanner)
	v2 := scenario(t, h, "V2", map[string]domain.Dec{
		string(domain.AsmRecoveryPct): domain.D("9.5"),
	})

	m, err := h.planning.CompareMatrix(ctx, service.MatrixRequest{
		VersionIDs: []string{h.seeded.BudgetID, v2.ID},
	})
	if err != nil {
		t.Fatalf("matrix: %v", err)
	}
	if len(m.Assumptions) != 1 {
		t.Fatalf("%d assumptions reported, want just the recovery: %v", len(m.Assumptions), m.Assumptions)
	}
	a := m.Assumptions[0]
	if a.Key != string(domain.AsmRecoveryPct) {
		t.Errorf("the changed assumption is %s", a.Key)
	}
	if !a.Values[0].Equal(domain.D("11")) || !a.Values[1].Equal(domain.D("9.5")) {
		t.Errorf("recovery reads %s then %s, want 11 then 9.5", a.Values[0], a.Values[1])
	}
}

func TestTheActualsDoNotFloodTheAssumptionComparison(t *testing.T) {
	// The actuals container carries no assumptions - it records what happened,
	// not what was assumed. Counting its absent settings as zeros made every
	// one of the seventeen look changed, which buries the one a planner
	// actually moved. Found by running the comparison, not by a unit test.
	h := newHarness(t, 14)
	ctx := h.as(auth.RoleProductionPlanner)
	v2 := scenario(t, h, "V2", map[string]domain.Dec{
		string(domain.AsmRecoveryPct): domain.D("9.5"),
	})

	m, err := h.planning.CompareMatrix(ctx, service.MatrixRequest{
		VersionIDs: []string{h.seeded.BudgetID, v2.ID}, IncludeActual: true,
	})
	if err != nil {
		t.Fatalf("matrix: %v", err)
	}
	if len(m.Columns) != 3 {
		t.Fatalf("%d columns, want two plans and the actuals", len(m.Columns))
	}
	if len(m.Assumptions) != 1 {
		var codes []string
		for _, a := range m.Assumptions {
			codes = append(codes, a.Key)
		}
		t.Fatalf("%d assumptions reported with the actuals in the comparison, want just "+
			"the recovery: %v", len(m.Assumptions), codes)
	}
	if m.Assumptions[0].Key != string(domain.AsmRecoveryPct) {
		t.Errorf("the reported assumption is %s", m.Assumptions[0].Key)
	}
}

func TestAComparisonNeedsTwoVersionsFromOneSeason(t *testing.T) {
	h := newHarness(t, 0)
	ctx := h.as(auth.RoleProductionPlanner)

	if _, err := h.planning.CompareMatrix(ctx, service.MatrixRequest{
		VersionIDs: []string{h.seeded.BudgetID},
	}); err == nil {
		t.Error("one version is not a comparison")
	}
	// The same version twice is still one version.
	if _, err := h.planning.CompareMatrix(ctx, service.MatrixRequest{
		VersionIDs: []string{h.seeded.BudgetID, h.seeded.BudgetID},
	}); err == nil {
		t.Error("the same version named twice is not a comparison")
	}
	if _, err := h.planning.CompareMatrix(ctx, service.MatrixRequest{
		VersionIDs:        []string{h.seeded.BudgetID, h.seeded.ActualID},
		BaselineVersionID: "00000000-0000-0000-0000-000000000000",
	}); err == nil {
		t.Error("a baseline that is not one of the columns must be refused")
	}
}
