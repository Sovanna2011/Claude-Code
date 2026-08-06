package seed_test

import (
	"context"
	"testing"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/seed"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
	"github.com/kss/sugarplan/internal/store/memory"
)

// The demonstration scenario. Two things have to hold, and the second was found
// by running it: the figures have to be the ones the scenario claims, and a
// second boot must not double them.

func loadDemo(t *testing.T, s store.Store) (seed.DemoResult, *service.Analytics) {
	t.Helper()
	planning := service.NewPlanning(s, func() time.Time { return time.Now().UTC() })
	analytics := service.NewAnalytics(s, planning)
	res, err := seed.LoadDemo(context.Background(), s, planning, analytics, 14)
	if err != nil {
		t.Fatalf("load the demonstration: %v", err)
	}
	return res, analytics
}

func TestTheDemonstrationFillsEveryExecutionScreen(t *testing.T) {
	s := memory.New()
	res, _ := loadDemo(t, s)

	for _, c := range []struct {
		what string
		got  int
		want int
	}{
		{"stoppages", res.Downtime, 7},
		{"orders", res.Orders, 5},
		{"confirmations", res.Confirmations, 5},
		// Ten, not the eight the scenario posts by hand: placing a quality hold
		// and releasing it are each their own document. The count is read back
		// from the ledger rather than tallied along the way, which is how the
		// difference was found.
		{"stock documents", res.Documents, 10},
		{"quality samples", res.Samples, 3},
		{"quality holds", res.Holds, 1},
		{"cost runs", res.CostRuns, 1},
		{"saved views", res.Views, 3},
	} {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.what, c.got, c.want)
		}
	}
	// Two capacity problems are in the reference plan itself; the alert job
	// finds them, and each goes to two roles.
	if res.Alerts == 0 {
		t.Error("the plan has two real capacity problems; the inboxes must not be empty")
	}
}

func TestASecondBootDoesNotDoubleTheDemonstration(t *testing.T) {
	s := memory.New()
	first, _ := loadDemo(t, s)
	if first.AlreadyPlayed {
		t.Fatal("the first load cannot have found execution data")
	}

	planning := service.NewPlanning(s, func() time.Time { return time.Now().UTC() })
	analytics := service.NewAnalytics(s, planning)
	second, err := seed.LoadDemo(context.Background(), s, planning, analytics, 14)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if !second.AlreadyPlayed {
		t.Fatal("the second load must find the execution data and leave it alone")
	}

	// The real assertion is on the data, not the flag: a container that
	// restarts must not turn seven stoppages into fourteen, or a demonstration
	// grows on its own and nobody can quote a figure from it.
	ctx := auth.WithPrincipal(context.Background(),
		auth.NewPrincipal("t", "t", "t", "", []string{auth.RoleSystemAdmin,
			auth.RoleShiftSupervisor, auth.RoleQualityUser},
			[]string{first.CompanyID}, []string{first.FactoryID}))

	events, err := s.Planning().ListDowntime(ctx, store.PlanFilter{FactoryID: first.FactoryID})
	if err != nil {
		t.Fatalf("read the stoppages: %v", err)
	}
	if len(events) != first.Downtime {
		t.Errorf("stoppages = %d after two boots, want the original %d",
			len(events), first.Downtime)
	}

	orders, err := s.Execution().ListOrders(ctx, store.ExecutionFilter{
		FactoryID: first.FactoryID, Top: 100,
	})
	if err != nil {
		t.Fatalf("read the orders: %v", err)
	}
	if orders.Count != first.Orders {
		t.Errorf("orders = %d after two boots, want the original %d",
			orders.Count, first.Orders)
	}
}

func TestTheDemonstrationDataReconcilesWithTheDashboard(t *testing.T) {
	s := memory.New()
	res, analytics := loadDemo(t, s)

	ctx := auth.WithPrincipal(context.Background(),
		auth.NewPrincipal("t", "t", "t", "", []string{auth.RoleExecutiveViewer},
			[]string{res.CompanyID}, []string{res.FactoryID}))
	dash, err := analytics.Dashboard(ctx, service.DashboardRequest{SeasonID: res.SeasonID})
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}

	// The stoppages the scenario recorded are the ones the Pareto ranks.
	if dash.Downtime.EventCount != res.Downtime {
		t.Errorf("the dashboard counts %d stoppages, the scenario recorded %d",
			dash.Downtime.EventCount, res.Downtime)
	}
	if len(dash.Downtime.ByReason) < 3 {
		t.Errorf("a Pareto of fewer than three reasons teaches nobody anything: %v",
			dash.Downtime.ByReason)
	}
	// The boiler dominates, which is what makes the chart worth looking at.
	worst := dash.Downtime.ByReason[0]
	if worst.ReasonCode != "DT-BOILER" {
		t.Errorf("the worst reason is %s; the scenario is built so the boiler leads",
			worst.ReasonCode)
	}
	if worst.SharePct.LessThan(domain.D("50")) {
		t.Errorf("the leading reason is only %s %% of the lost time; the scenario "+
			"is meant to have an obvious answer", worst.SharePct)
	}

	// The cumulative share reaches exactly 100. This is the case that found the
	// rounding overshoot: 18, 7, 2 and 2 hours of 29.
	last := dash.Downtime.ByReason[len(dash.Downtime.ByReason)-1]
	if !last.CumSharePct.Equal(domain.DI(100)) {
		t.Errorf("the cumulative share ends at %s, want exactly 100", last.CumSharePct)
	}

	// And the reference figures still hold with a fortnight of factory life on
	// top of them: the demonstration must not have moved the plan.
	if !dash.Cane.SeasonTarget.Equal(domain.D("2300000.000")) {
		t.Errorf("season target = %s, want 2,300,000", dash.Cane.SeasonTarget)
	}
	total := domain.Zero
	for _, p := range dash.Products {
		total = total.Add(p.TargetTons)
	}
	if !total.Equal(domain.D("242100.000")) {
		t.Errorf("the finished goods plan totals %s, want 242,100", total)
	}
}

func TestTheDemonstrationClosesAnOrderShortWithAReason(t *testing.T) {
	s := memory.New()
	res, _ := loadDemo(t, s)

	ctx := auth.WithPrincipal(context.Background(),
		auth.NewPrincipal("t", "t", "t", "", []string{auth.RoleShiftSupervisor},
			[]string{res.CompanyID}, []string{res.FactoryID}))
	orders, err := s.Execution().ListOrders(ctx, store.ExecutionFilter{
		FactoryID: res.FactoryID, Top: 100,
	})
	if err != nil {
		t.Fatalf("read the orders: %v", err)
	}

	// One order finishes short. That is the case worth demonstrating: the
	// system refuses a silent close, so the reason on it is one somebody had to
	// give, and it is what the variance report reads back.
	short := 0
	for _, o := range orders.Items {
		if o.ConfirmedQty.LessThan(o.PlannedQty) {
			short++
			if o.VarianceReason == "" {
				t.Errorf("order %s is %s t short with no reason recorded",
					o.OrderNo, o.PlannedQty.Sub(o.ConfirmedQty))
			}
			if o.Status != domain.OrderTechnicallyClosed {
				t.Errorf("order %s is short but %s, not closed", o.OrderNo, o.Status)
			}
		}
	}
	if short != 1 {
		t.Errorf("%d orders finished short; the scenario is built with exactly one", short)
	}
}

func TestTheDemonstrationBlocksAndThenReleasesFailingSugar(t *testing.T) {
	s := memory.New()
	res, _ := loadDemo(t, s)

	ctx := auth.WithPrincipal(context.Background(),
		auth.NewPrincipal("t", "t", "t", "", []string{auth.RoleQualityUser},
			[]string{res.CompanyID}, []string{res.FactoryID}))

	holds, err := s.Execution().ListHolds(ctx, store.ExecutionFilter{Top: 100})
	if err != nil {
		t.Fatalf("read the holds: %v", err)
	}
	if len(holds) != 1 {
		t.Fatalf("one hold expected, got %d", len(holds))
	}
	// It was placed by a failing sample and then released after a rework, so
	// the demonstration shows both halves rather than leaving sugar blocked
	// with nothing to explain how it gets unblocked.
	if holds[0].ReleasedOn == "" {
		t.Error("the hold must be released, so the demonstration shows the way out")
	}
	if holds[0].SampleID == "" {
		t.Error("the hold must point at the sample that caused it")
	}

	samples, err := s.Execution().ListSamples(ctx, store.ExecutionFilter{
		FactoryID: res.FactoryID, Top: 100,
	})
	if err != nil {
		t.Fatalf("read the samples: %v", err)
	}
	failed := 0
	for _, sample := range samples.Items {
		for _, r := range sample.Results {
			if r.Status == domain.QualityFail {
				failed++
			}
		}
	}
	if failed == 0 {
		t.Error("a quality module that only ever passes has demonstrated nothing")
	}
}
