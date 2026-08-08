package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/seed"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
	"github.com/kss/sugarplan/internal/store/memory"
)

// fixedClock keeps audit timestamps and forecasts deterministic.
func fixedClock() func() time.Time {
	t := time.Date(2026, 12, 15, 6, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

type harness struct {
	store     store.Store
	planning  *service.Planning
	analytics *service.Analytics
	materials *service.Materials
	seeded    seed.Result
}

// newHarness seeds the reference scenario into an in-memory store.
func newHarness(t *testing.T, actualDays int) *harness {
	t.Helper()
	s := memory.New()
	planning := service.NewPlanning(s, fixedClock())
	res, err := seed.LoadWithActuals(context.Background(), s, planning, actualDays)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return &harness{
		store: s, planning: planning,
		analytics: service.NewAnalytics(s, planning),
		materials: service.NewMaterials(s, planning),
		seeded:    res,
	}
}

// as returns a context carrying a principal with the given roles.
func (h *harness) as(roles ...string) context.Context {
	p := auth.NewPrincipal("test|"+strings.Join(roles, "+"), "test-"+roles[0], "Test user", "",
		roles, []string{h.seeded.CompanyID}, []string{h.seeded.FactoryID})
	return auth.WithPrincipal(context.Background(), p)
}

// ---------------------------------------------------------------------------
// Seeded scenario
// ---------------------------------------------------------------------------

func TestSeedProducesTheReferenceScenario(t *testing.T) {
	h := newHarness(t, 0)
	sum := h.seeded.Generated.Summary

	if !sum.CaneAllocated.Equal(domain.D("2300000")) {
		t.Errorf("cane allocated = %s, want 2,300,000", sum.CaneAllocated)
	}
	if !sum.RawSugarExpected.Equal(domain.D("253000.000")) {
		t.Errorf("raw sugar = %s, want 253,000", sum.RawSugarExpected)
	}
	if !sum.FinishedGoods.Equal(domain.D("242100")) {
		t.Errorf("finished goods = %s, want 242,100", sum.FinishedGoods)
	}
	if sum.WorkingDays != 137 {
		t.Errorf("working days = %d, want 137", sum.WorkingDays)
	}
	// The campaign is the whole planning horizon - crushing plus the remelt
	// season that lives off the silo - and it is nine months, not four and a
	// half. The cane stops on 16 April; the refinery and the quota do not.
	if h.seeded.Generated.FirstDate != "2026-12-01" || h.seeded.Generated.LastCrushingDate != "2027-04-16" {
		t.Errorf("crushing runs %s to %s", h.seeded.Generated.FirstDate,
			h.seeded.Generated.LastCrushingDate)
	}
	if h.seeded.Generated.LastDate != "2027-09-02" {
		t.Errorf("the campaign ends %s, the workbook runs to 2027-09-02",
			h.seeded.Generated.LastDate)
	}
	if got := h.seeded.Generated.RowCounts["cane"]; got != 137 {
		t.Errorf("cane rows = %d, want 137", got)
	}
}

func TestSeedIsIdempotent(t *testing.T) {
	h := newHarness(t, 0)
	// Re-running the seed must not create a second company or season.
	if _, err := seed.Load(context.Background(), h.store, h.planning); err != nil {
		t.Fatalf("second seed run: %v", err)
	}
	ctx := h.as(auth.RoleProductionPlanner)
	companies, err := h.store.MasterData().Companies().List(ctx, store.ListOptions{})
	if err != nil {
		t.Fatalf("list companies: %v", err)
	}
	if companies.Count != 1 {
		t.Errorf("companies = %d, want 1 after two seed runs", companies.Count)
	}
	seasons, err := h.planning.ListSeasons(ctx, store.ListOptions{})
	if err != nil {
		t.Fatalf("list seasons: %v", err)
	}
	if seasons.Count != 1 {
		t.Errorf("seasons = %d, want 1 after two seed runs", seasons.Count)
	}
}

func TestGeneratedLedgerBalancesAreStoredAndContinuous(t *testing.T) {
	h := newHarness(t, 0)
	ctx := h.as(auth.RoleProductionPlanner)

	for code, id := range h.seeded.Warehouses {
		rows, err := h.store.Planning().ListStorage(ctx, store.PlanFilter{
			VersionIDs: []string{h.seeded.BudgetID}, WarehouseIDs: []string{id},
		})
		if err != nil {
			t.Fatalf("list storage: %v", err)
		}
		if len(rows) == 0 {
			continue
		}
		byProduct := map[string][]domain.DailyStoragePlan{}
		for _, r := range rows {
			byProduct[r.ProductID] = append(byProduct[r.ProductID], r)
		}
		for product, group := range byProduct {
			for i := 1; i < len(group); i++ {
				if !group[i].BeginningBalance.Equal(group[i-1].EndingBalance) {
					t.Fatalf("%s/%s: beginning balance on %s (%s) does not match the previous ending balance (%s)",
						code, product, group[i].BusinessDate,
						group[i].BeginningBalance, group[i-1].EndingBalance)
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Workflow and locking
// ---------------------------------------------------------------------------

func TestPlanWorkflowEndToEnd(t *testing.T) {
	h := newHarness(t, 0)
	plannerCtx := h.as(auth.RoleProductionPlanner)
	approverCtx := h.as(auth.RoleApprover)

	// A planner submits.
	v, err := h.planning.Transition(plannerCtx, h.seeded.BudgetID,
		service.TransitionRequest{Action: domain.ActionSubmit})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if v.Status != domain.StatusInReview {
		t.Fatalf("status after submit = %s", v.Status)
	}

	// The planner cannot approve their own plan.
	if _, err := h.planning.Transition(plannerCtx, h.seeded.BudgetID,
		service.TransitionRequest{Action: domain.ActionApprove}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("a planner approving their own plan = %v, want ErrForbidden", err)
	}

	// The approver approves and releases.
	v, err = h.planning.Transition(approverCtx, h.seeded.BudgetID,
		service.TransitionRequest{Action: domain.ActionApprove})
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if v.ApprovedBy == "" || v.ApprovedAt == nil {
		t.Error("approval must be stamped with who and when")
	}

	v, err = h.planning.Transition(approverCtx, h.seeded.BudgetID, service.TransitionRequest{
		Action: domain.ActionRelease, LockThrough: "2026-12-31",
	})
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	if v.Status != domain.StatusReleased || v.LockedThrough != "2026-12-31" {
		t.Fatalf("release did not lock the period: %+v", v)
	}

	// A locked date can no longer be edited.
	res, err := h.planning.UpsertCane(plannerCtx, h.seeded.BudgetID, []domain.DailyCanePlan{{
		BusinessDate: "2026-12-15", Series: domain.SeriesPlan,
		CaneCrushed: domain.D("20000"), AvailableHrs: domain.D("24"),
	}}, service.UpsertOptions{})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("editing a locked date = %v, want a validation error", err)
	}
	if len(res.Issues) != 1 || res.Issues[0].Code != "LOCKED" {
		t.Errorf("expected a LOCKED issue, got %+v", res.Issues)
	}

	// A date beyond the lock is still editable.
	if _, err := h.planning.UpsertCane(plannerCtx, h.seeded.BudgetID, []domain.DailyCanePlan{{
		BusinessDate: "2027-02-15", Series: domain.SeriesPlan,
		CaneCrushed: domain.D("17000"), AvailableHrs: domain.D("24"),
	}}, service.UpsertOptions{}); err != nil {
		t.Errorf("editing beyond the lock date: %v", err)
	}

	// Reopening needs a reason and the reopen permission.
	if _, err := h.planning.Transition(approverCtx, h.seeded.BudgetID,
		service.TransitionRequest{Action: domain.ActionReopen}); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("reopen without a reason = %v, want a validation error", err)
	}
	if _, err := h.planning.Transition(plannerCtx, h.seeded.BudgetID, service.TransitionRequest{
		Action: domain.ActionReopen, Reason: "revised cane forecast",
	}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("a planner reopening = %v, want ErrForbidden", err)
	}
	reopened, err := h.planning.Transition(approverCtx, h.seeded.BudgetID, service.TransitionRequest{
		Action: domain.ActionReopen, Reason: "revised cane forecast",
	})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if reopened.Status != domain.StatusDraft || reopened.ApprovedBy != "" || reopened.ReleasedAt != nil {
		t.Errorf("reopening must clear the approval trail: %+v", reopened)
	}

	// The whole sequence is in the audit log.
	events, err := h.store.Audit().List(approverCtx, store.AuditFilter{
		Entity: "plan_version", EntityID: h.seeded.BudgetID,
	})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	actions := map[string]bool{}
	for _, e := range events.Items {
		actions[e.Action] = true
	}
	for _, want := range []string{"SUBMIT", "APPROVE", "RELEASE", "REOPEN"} {
		if !actions[want] {
			t.Errorf("the audit trail is missing %s (has %v)", want, actions)
		}
	}
}

func TestReleaseSupersedesThePreviousBaseline(t *testing.T) {
	h := newHarness(t, 0)
	plannerCtx := h.as(auth.RoleProductionPlanner)
	approverCtx := h.as(auth.RoleApprover)

	release := func(id string) {
		t.Helper()
		if _, err := h.planning.Transition(plannerCtx, id, service.TransitionRequest{Action: domain.ActionSubmit}); err != nil {
			t.Fatalf("submit: %v", err)
		}
		if _, err := h.planning.Transition(approverCtx, id, service.TransitionRequest{Action: domain.ActionApprove}); err != nil {
			t.Fatalf("approve: %v", err)
		}
		if _, err := h.planning.Transition(approverCtx, id, service.TransitionRequest{Action: domain.ActionRelease}); err != nil {
			t.Fatalf("release: %v", err)
		}
	}
	release(h.seeded.BudgetID)

	revised, err := h.planning.CopyVersion(plannerCtx, service.CopyRequest{
		SourceVersionID: h.seeded.BudgetID, Code: "V2", Description: "Revised plan",
		PlanType: domain.PlanTypeRevised, CopyDailyRows: true,
	})
	if err != nil {
		t.Fatalf("copy: %v", err)
	}
	release(revised.ID)

	old, err := h.planning.GetVersion(plannerCtx, h.seeded.BudgetID)
	if err != nil {
		t.Fatalf("re-read the first version: %v", err)
	}
	if old.Version.Status != domain.StatusSuperseded {
		t.Errorf("the previous baseline is %s, want SUPERSEDED", old.Version.Status)
	}
}

// ---------------------------------------------------------------------------
// Scenarios
// ---------------------------------------------------------------------------

func TestWhatIfScenarioChangesTheOutcome(t *testing.T) {
	h := newHarness(t, 0)
	ctx := h.as(auth.RoleProductionPlanner)

	// A lower recovery assumption: what does 10.2 % do to the season?
	scenario, err := h.planning.CopyVersion(ctx, service.CopyRequest{
		SourceVersionID: h.seeded.BudgetID, Code: "WHATIF-LOW-RECOVERY",
		Description: "Recovery at 10.2 %", PlanType: domain.PlanTypeWhatIf,
		AssumptionOverrides: map[string]domain.Dec{domain.AsmRecoveryPct: domain.D("10.2")},
	})
	if err != nil {
		t.Fatalf("copy scenario: %v", err)
	}

	gen, err := h.planning.Generate(ctx, scenario.ID, service.GenerateRequest{Replace: true})
	if err != nil {
		t.Fatalf("generate scenario: %v", err)
	}
	// 2,300,000 x 10.20 % = 234,600 t
	if !gen.Summary.RawSugarExpected.Equal(domain.D("234600.000")) {
		t.Errorf("raw sugar at 10.2 %% = %s, want 234,600", gen.Summary.RawSugarExpected)
	}

	cmp, err := h.planning.Compare(ctx, service.CompareRequest{
		BaseVersionID: h.seeded.BudgetID, OtherVersionID: scenario.ID, Dimension: "PROCESS",
	})
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if len(cmp.AssumptionDeltas) != 1 || cmp.AssumptionDeltas[0].Key != domain.AsmRecoveryPct {
		t.Errorf("the comparison should show exactly the changed assumption, got %+v", cmp.AssumptionDeltas)
	}
	if !cmp.AssumptionDeltas[0].Delta.Equal(domain.D("-0.8")) {
		t.Errorf("recovery delta = %s, want -0.8", cmp.AssumptionDeltas[0].Delta)
	}

	// A simulation can never be released.
	if _, err := h.planning.Transition(ctx, scenario.ID,
		service.TransitionRequest{Action: domain.ActionSubmit}); !errors.Is(err, domain.ErrStateTransition) {
		t.Errorf("submitting a simulation = %v, want a state transition error", err)
	}
}

// ---------------------------------------------------------------------------
// Actuals and dashboard
// ---------------------------------------------------------------------------

func TestActualsCannotBePostedToAPlanVersion(t *testing.T) {
	h := newHarness(t, 0)
	ctx := h.as(auth.RoleShiftSupervisor)

	res, err := h.planning.UpsertCane(ctx, h.seeded.BudgetID, []domain.DailyCanePlan{{
		BusinessDate: "2026-12-01", Series: domain.SeriesActual,
		CaneCrushed: domain.D("15000"), AvailableHrs: domain.D("24"),
	}}, service.UpsertOptions{})
	if err == nil {
		t.Fatal("posting an actual to the budget version must fail")
	}
	if len(res.Issues) == 0 || res.Issues[0].Code != "INVALID_SERIES" {
		t.Errorf("expected an INVALID_SERIES issue, got %+v", res.Issues)
	}
}

func TestOperatorPermissionsAreEnforcedPerSeries(t *testing.T) {
	h := newHarness(t, 0)

	// A cane operator may post cane actuals.
	caneCtx := h.as(auth.RoleCaneOperator)
	if _, err := h.planning.UpsertCane(caneCtx, h.seeded.ActualID, []domain.DailyCanePlan{{
		BusinessDate: "2026-12-01", Series: domain.SeriesActual,
		CaneCrushed: domain.D("15000"), AvailableHrs: domain.D("24"),
	}}, service.UpsertOptions{}); err != nil {
		t.Fatalf("cane operator posting cane: %v", err)
	}

	// but not production actuals.
	res, err := h.planning.UpsertProduction(caneCtx, h.seeded.ActualID, []domain.DailyProductPlan{{
		BusinessDate: "2026-12-01", ProductID: h.seeded.Products["RAW"],
		Series: domain.SeriesActual, Quantity: domain.D("1650"),
	}}, service.UpsertOptions{})
	if err == nil {
		t.Fatal("a cane operator must not post production actuals")
	}
	if len(res.Issues) == 0 || res.Issues[0].Code != "FORBIDDEN" {
		t.Errorf("expected a FORBIDDEN issue, got %+v", res.Issues)
	}

	// An executive viewer may post nothing at all.
	viewerCtx := h.as(auth.RoleExecutiveViewer)
	if _, err := h.planning.UpsertCane(viewerCtx, h.seeded.ActualID, []domain.DailyCanePlan{{
		BusinessDate: "2026-12-02", Series: domain.SeriesActual,
		CaneCrushed: domain.D("15000"), AvailableHrs: domain.D("24"),
	}}, service.UpsertOptions{}); err == nil {
		t.Error("an executive viewer must not post actuals")
	}
}

func TestBulkUpsertIsAllOrNothingByDefault(t *testing.T) {
	h := newHarness(t, 0)
	ctx := h.as(auth.RoleShiftSupervisor)

	rows := []domain.DailyCanePlan{
		{BusinessDate: "2026-12-01", Series: domain.SeriesActual,
			CaneCrushed: domain.D("15000"), AvailableHrs: domain.D("24")},
		{BusinessDate: "2026-12-02", Series: domain.SeriesActual,
			CaneCrushed: domain.D("-5"), AvailableHrs: domain.D("24")}, // invalid
	}
	res, err := h.planning.UpsertCane(ctx, h.seeded.ActualID, rows, service.UpsertOptions{})
	if err == nil {
		t.Fatal("a batch with an invalid row must be rejected")
	}
	if res.Accepted != 0 {
		t.Errorf("accepted = %d, want 0 for an all-or-nothing batch", res.Accepted)
	}
	stored, err := h.store.Planning().ListCane(ctx, store.PlanFilter{
		VersionIDs: []string{h.seeded.ActualID}, Series: domain.SeriesActual,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(stored) != 0 {
		t.Errorf("nothing should have been written, found %d rows", len(stored))
	}

	// With partial mode the good row is kept and the bad one reported.
	res, err = h.planning.UpsertCane(ctx, h.seeded.ActualID, rows, service.UpsertOptions{Partial: true})
	if err != nil {
		t.Fatalf("partial upsert: %v", err)
	}
	if res.Accepted != 1 || res.Rejected != 1 {
		t.Errorf("partial result = %+v, want 1 accepted and 1 rejected", res)
	}
	if len(res.Issues) != 1 || res.Issues[0].Row != 1 || res.Issues[0].Field != "caneCrushed" {
		t.Errorf("issue should point at row 1 field caneCrushed, got %+v", res.Issues)
	}
}

func TestStorageEditRebuildsTheRunningBalances(t *testing.T) {
	h := newHarness(t, 0)
	ctx := h.as(auth.RoleProductionPlanner)

	warehouse := h.seeded.Warehouses["FG-WH1"]
	rows, err := h.store.Planning().ListStorage(ctx, store.PlanFilter{
		VersionIDs: []string{h.seeded.BudgetID}, WarehouseIDs: []string{warehouse},
	})
	if err != nil || len(rows) < 3 {
		t.Fatalf("list storage: %v (%d rows)", err, len(rows))
	}

	// Add 1,000 t of shipment on the first day; every later balance must drop.
	target := rows[0]
	before := rows[2].EndingBalance
	target.ShipmentQty = target.ShipmentQty.Add(domain.D("1000"))
	if _, err := h.planning.UpsertStorage(ctx, h.seeded.BudgetID,
		[]domain.DailyStoragePlan{target}, service.UpsertOptions{}); err != nil {
		t.Fatalf("upsert storage: %v", err)
	}

	after, err := h.store.Planning().ListStorage(ctx, store.PlanFilter{
		VersionIDs: []string{h.seeded.BudgetID}, WarehouseIDs: []string{warehouse},
		ProductIDs: []string{target.ProductID},
	})
	if err != nil {
		t.Fatalf("re-read storage: %v", err)
	}
	if !after[2].EndingBalance.Equal(before.Sub(domain.D("1000"))) {
		t.Errorf("day 3 ending balance = %s, want %s (1,000 t less than before)",
			after[2].EndingBalance, before.Sub(domain.D("1000")))
	}
	for i := 1; i < len(after); i++ {
		if !after[i].BeginningBalance.Equal(after[i-1].EndingBalance) {
			t.Fatalf("continuity broken on %s after the edit", after[i].BusinessDate)
		}
	}
}

func TestDashboardReconcilesWithTheSeededPlan(t *testing.T) {
	h := newHarness(t, 14)
	ctx := h.as(auth.RoleExecutiveViewer)

	dash, err := h.analytics.Dashboard(ctx, service.DashboardRequest{SeasonID: h.seeded.SeasonID})
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}

	if !dash.Cane.SeasonTarget.Equal(domain.D("2300000.000")) {
		t.Errorf("season target = %s, want 2,300,000", dash.Cane.SeasonTarget)
	}
	if dash.AsOf != "2026-12-14" {
		t.Errorf("as-of date = %s, want the last day with an actual (2026-12-14)", dash.AsOf)
	}
	if dash.Cane.CumulativeActual.IsZero() {
		t.Error("the dashboard should show the recorded actuals")
	}
	if dash.Cane.CumulativeActual.GreaterThan(dash.Cane.CumulativeTarget) {
		t.Errorf("the seeded actuals run below target, got actual %s against target %s",
			dash.Cane.CumulativeActual, dash.Cane.CumulativeTarget)
	}
	// Cumulative actual plus remaining must equal the season target.
	if got := dash.Cane.CumulativeActual.Add(dash.Cane.Remaining); !got.Equal(dash.Cane.SeasonTarget) {
		t.Errorf("cumulative actual + remaining = %s, want the season target %s",
			got, dash.Cane.SeasonTarget)
	}
	if !dash.Cane.ForecastReliable {
		t.Error("with two weeks of actuals the completion forecast should be reliable")
	}
	if dash.Cane.ForecastEndDate <= dash.AsOf {
		t.Errorf("the forecast end date %s must be in the future", dash.Cane.ForecastEndDate)
	}

	// The seeded fortnight starts badly and recovers to slightly above target,
	// so the campaign is behind on tonnage but the forecast, which follows the
	// recent rate, is not alarming. No schedule alert is expected here; the
	// alert itself is exercised in TestDashboardWarnsWhenCrushingFallsBehind.
	for _, a := range dash.Alerts {
		if a.Code == "CRUSHING_BEHIND_SCHEDULE" {
			t.Errorf("unexpected schedule alert while the rolling rate is at plan: %s", a.Detail)
		}
	}

	// Finished goods capacity is exceeded by the reference plan; the dashboard
	// must say so and say what shipment rate would fix it.
	var capacityAlert bool
	for _, a := range dash.Alerts {
		if a.Code == "CAPACITY_EXCEEDED" {
			capacityAlert = true
		}
	}
	if !capacityAlert {
		t.Errorf("expected a capacity alert; alerts were %v", alertCodes(dash.Alerts))
	}
	for _, s := range dash.Storage {
		if s.StorageClass == domain.StorageFinished && s.RequiredShipTPD.IsZero() {
			t.Errorf("%s overflows but reports no required shipment rate", s.WarehouseCode)
		}
	}

	// Product targets must add back to the seeded mix.
	total := domain.Zero
	for _, p := range dash.Products {
		total = total.Add(p.TargetTons)
	}
	if !total.Equal(domain.D("242100.000")) {
		t.Errorf("product targets sum to %s, want 242,100", total)
	}
}

func TestDashboardWarnsWhenCrushingFallsBehind(t *testing.T) {
	h := newHarness(t, 0)
	opCtx := h.as(auth.RoleShiftSupervisor)

	// Two weeks of crushing at 60 % of target: the mill cannot finish 2.3
	// million tons at that rate and the dashboard has to say so.
	plan, err := h.store.Planning().ListCane(opCtx, store.PlanFilter{
		VersionIDs: []string{h.seeded.BudgetID}, Series: domain.SeriesPlan, Top: 14,
	})
	if err != nil {
		t.Fatalf("read plan: %v", err)
	}
	var rows []domain.DailyCanePlan
	for _, p := range plan {
		rows = append(rows, domain.DailyCanePlan{
			FactoryID: h.seeded.FactoryID, BusinessDate: p.BusinessDate, Series: domain.SeriesActual,
			CaneCrushed:  domain.RoundQty(p.CaneCrushed.Mul(domain.D("0.6"))),
			AvailableHrs: domain.D("24"), StoppageHrs: domain.D("9"),
		})
	}
	if _, err := h.planning.UpsertCane(opCtx, h.seeded.ActualID, rows, service.UpsertOptions{}); err != nil {
		t.Fatalf("post actuals: %v", err)
	}

	dash, err := h.analytics.Dashboard(h.as(auth.RoleExecutiveViewer),
		service.DashboardRequest{SeasonID: h.seeded.SeasonID})
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if dash.Cane.DaysBehindSchedule <= 0 {
		t.Fatalf("crushing at 60 %% of plan must forecast late, got %d days", dash.Cane.DaysBehindSchedule)
	}
	var alert *domain.Alert
	for i := range dash.Alerts {
		if dash.Alerts[i].Code == "CRUSHING_BEHIND_SCHEDULE" {
			alert = &dash.Alerts[i]
		}
	}
	if alert == nil {
		t.Fatalf("expected a behind-schedule alert; alerts were %v", alertCodes(dash.Alerts))
	}
	if alert.Severity != domain.SeverityError {
		t.Errorf("being weeks late should be an error, got %s", alert.Severity)
	}
	// Errors sort first so the worst news is at the top of the dashboard.
	if dash.Alerts[0].Severity != domain.SeverityError {
		t.Errorf("alerts must be ordered worst first, got %s", dash.Alerts[0].Severity)
	}
}

func alertCodes(alerts []domain.Alert) []string {
	out := make([]string, len(alerts))
	for i, a := range alerts {
		out[i] = a.Code
	}
	return out
}

// ---------------------------------------------------------------------------
// Materials
// ---------------------------------------------------------------------------

func TestPackagingRequirements(t *testing.T) {
	h := newHarness(t, 0)
	ctx := h.as(auth.RoleProductionPlanner)

	reqs, err := h.materials.Requirements(ctx, service.RequirementsRequest{VersionID: h.seeded.BudgetID})
	if err != nil {
		t.Fatalf("requirements: %v", err)
	}
	if len(reqs) == 0 {
		t.Fatal("the plan packs sugar, so there must be packaging requirements")
	}

	byCode := map[string]service.MaterialRequirement{}
	for _, r := range reqs {
		byCode[r.MaterialCode] = r
	}

	// 240,100 t of refined and white sugar in 50 kg bags, plus 0.5 % scrap.
	bags, ok := byCode["BAG-50"]
	if !ok {
		t.Fatal("no requirement calculated for the 50 kg bag")
	}
	expected := domain.DI(domain.RequiredPackages(domain.D("240100"), domain.D("50"), domain.D("0.5")))
	if !bags.GrossRequired.Equal(expected) {
		t.Errorf("50 kg bags required = %s, want %s", bags.GrossRequired, expected)
	}
	if bags.Severity != domain.SeverityError {
		t.Errorf("4.8 million bags against 650,000 in stock and on order should be an error, got %s",
			bags.Severity)
	}
	if bags.SuggestedOrder == "" || bags.SuggestedOrder >= bags.RequiredBy {
		t.Errorf("the suggested order date %s must precede the required-by date %s",
			bags.SuggestedOrder, bags.RequiredBy)
	}
	if len(bags.Sources) == 0 {
		t.Error("the requirement should show which packaging drove it")
	}

	// A viewer without the materials permission gets nothing.
	if _, err := h.materials.Requirements(h.as(auth.RoleExecutiveViewer),
		service.RequirementsRequest{VersionID: h.seeded.BudgetID}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("materials access without permission = %v, want ErrForbidden", err)
	}
}

// ---------------------------------------------------------------------------
// Data scope
// ---------------------------------------------------------------------------

func TestDataScopeHidesOtherFactories(t *testing.T) {
	h := newHarness(t, 0)

	// A principal scoped to a different factory sees no seasons and cannot
	// open one by id.
	outsider := auth.NewPrincipal("test|outsider", "outsider", "Other factory planner", "",
		[]string{auth.RoleProductionPlanner}, []string{"another-company"}, []string{"another-factory"})
	ctx := auth.WithPrincipal(context.Background(), outsider)

	seasons, err := h.planning.ListSeasons(ctx, store.ListOptions{})
	if err != nil {
		t.Fatalf("list seasons: %v", err)
	}
	if seasons.Count != 0 {
		t.Errorf("an out-of-scope user sees %d seasons, want 0", seasons.Count)
	}
	if _, err := h.planning.GetSeason(ctx, h.seeded.SeasonID); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("reading an out-of-scope season = %v, want ErrForbidden", err)
	}
	if _, err := h.planning.GetVersion(ctx, h.seeded.BudgetID); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("reading an out-of-scope version = %v, want ErrForbidden", err)
	}
}

func TestAnonymousCallerIsRejected(t *testing.T) {
	h := newHarness(t, 0)
	ctx := context.Background() // no principal at all

	if _, err := h.planning.ListSeasons(ctx, store.ListOptions{}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("an anonymous caller = %v, want ErrForbidden", err)
	}
}

// ---------------------------------------------------------------------------
// Transactions
// ---------------------------------------------------------------------------

func TestGenerateRefusesToOverwriteWithoutReplace(t *testing.T) {
	h := newHarness(t, 0)
	ctx := h.as(auth.RoleProductionPlanner)

	// The seed already generated a plan; a second run without replace must fail
	// and must leave the existing plan untouched.
	before, err := h.store.Planning().ListCane(ctx, store.PlanFilter{VersionIDs: []string{h.seeded.BudgetID}})
	if err != nil {
		t.Fatalf("list cane: %v", err)
	}
	if _, err := h.planning.Generate(ctx, h.seeded.BudgetID,
		service.GenerateRequest{}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("generate without replace = %v, want a validation error", err)
	}
	after, err := h.store.Planning().ListCane(ctx, store.PlanFilter{VersionIDs: []string{h.seeded.BudgetID}})
	if err != nil {
		t.Fatalf("list cane after: %v", err)
	}
	if len(after) != len(before) {
		t.Errorf("the failed generate changed the plan: %d rows before, %d after", len(before), len(after))
	}
}
