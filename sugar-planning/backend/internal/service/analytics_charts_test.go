package service_test

import (
	"testing"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
)

// The series behind the charts. A chart drawn from wrong numbers is worse than
// no chart, because it is believed.

func TestTheRecoveryTrendFollowsTheActualsNotTheAssumption(t *testing.T) {
	h := newHarness(t, 14)
	dash, err := h.analytics.Dashboard(h.as(auth.RoleExecutiveViewer),
		service.DashboardRequest{SeasonID: h.seeded.SeasonID})
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}

	if len(dash.RecoveryTrend) != len(dash.CaneTrend) {
		t.Fatalf("the recovery trend must cover the same days as the cane trend: %d against %d",
			len(dash.RecoveryTrend), len(dash.CaneTrend))
	}

	// The target line is the assumption, flat across the season. Plotting the
	// plan's own recovery would draw it twice.
	for _, p := range dash.RecoveryTrend {
		if !p.Target.Equal(domain.D("11.000")) {
			t.Fatalf("%s: target recovery = %s, want the 11 %% assumption", p.Date, p.Target)
		}
	}

	// The seeded actuals drift 10.7, 10.8, 10.9, 11.0, 11.1 and repeat.
	want := []string{"10.700", "10.800", "10.900", "11.000", "11.100"}
	for i, w := range want {
		p := dash.RecoveryTrend[i]
		if !p.HasActual {
			t.Fatalf("day %d has cane and raw sugar recorded but no recovery", i)
		}
		if !p.Actual.Equal(domain.D(w)) {
			t.Errorf("day %d recovery = %s, want %s", i, p.Actual, w)
		}
	}

	// Beyond the fortnight of actuals there is nothing to plot, and a zero
	// would read as a factory producing no sugar at all.
	last := dash.RecoveryTrend[len(dash.RecoveryTrend)-1]
	if last.HasActual {
		t.Errorf("%s is past the recorded actuals and must carry none", last.Date)
	}
}

func TestTheProductMixIsOrderedByHowMuchItMatters(t *testing.T) {
	h := newHarness(t, 14)
	dash, err := h.analytics.Dashboard(h.as(auth.RoleExecutiveViewer),
		service.DashboardRequest{SeasonID: h.seeded.SeasonID})
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}

	if len(dash.ProductTrend) == 0 {
		t.Fatal("no product mix series")
	}
	for i := 1; i < len(dash.ProductTrend); i++ {
		if dash.ProductTrend[i-1].Total.LessThan(dash.ProductTrend[i].Total) {
			t.Errorf("the mix bands must be ordered largest first: %s (%s) before %s (%s)",
				dash.ProductTrend[i-1].Code, dash.ProductTrend[i-1].Total,
				dash.ProductTrend[i].Code, dash.ProductTrend[i].Total)
		}
	}

	// Raw sugar is not a finished good and has no place in a finished goods
	// mix; its band would be twice the height of everything else.
	for _, s := range dash.ProductTrend {
		if s.Code == "RAW" {
			t.Error("raw sugar must not be a band of the finished goods mix")
		}
	}

	// The plan targets in the series must add back to the seeded mix, which is
	// what makes the chart reconcile with the KPI table beside it.
	total := domain.Zero
	for _, s := range dash.ProductTrend {
		if n := len(s.Points); n > 0 {
			total = total.Add(s.Points[n-1].CumTarget)
		}
	}
	if !total.Equal(domain.D("242100.000")) {
		t.Errorf("the mix series sum to %s t of plan, want 242,100", total)
	}
}

func TestTheShipmentTrendIsDailyAndPerChannel(t *testing.T) {
	h := newHarness(t, 14)
	dash, err := h.analytics.Dashboard(h.as(auth.RoleExecutiveViewer),
		service.DashboardRequest{SeasonID: h.seeded.SeasonID})
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}

	if len(dash.ShipmentTrend) == 0 {
		t.Fatal("no shipment series")
	}
	// Every channel with a figure in the KPI table needs a curve, or the chart
	// and the table below it disagree about who shipped what.
	inTrend := map[string]bool{}
	for _, s := range dash.ShipmentTrend {
		inTrend[s.Code] = true
		if len(s.Points) != len(dash.CaneTrend) {
			t.Errorf("channel %s has %d points against %d days",
				s.Code, len(s.Points), len(dash.CaneTrend))
		}
	}
	for _, c := range dash.Shipments {
		if !inTrend[c.ChannelCode] {
			t.Errorf("channel %s is in the KPI table but has no curve", c.ChannelCode)
		}
	}

	// The curve must add back to the KPI it sits beside.
	for _, s := range dash.ShipmentTrend {
		for _, c := range dash.Shipments {
			if c.ChannelID != s.ID {
				continue
			}
			last := s.Points[len(s.Points)-1]
			if !last.CumTarget.Equal(c.PlannedTons) {
				t.Errorf("channel %s: the curve ends at %s t planned, the KPI says %s",
					s.Code, last.CumTarget, c.PlannedTons)
			}
			if !last.CumActual.Equal(c.ActualTons) {
				t.Errorf("channel %s: the curve ends at %s t actual, the KPI says %s",
					s.Code, last.CumActual, c.ActualTons)
			}
		}
	}
}

func TestTheDowntimeParetoRanksTheReasons(t *testing.T) {
	h := newHarness(t, 0)
	ctx := h.as(auth.RoleShiftSupervisor)

	// Three stoppages: a long boiler outage, a shorter one, and a power cut.
	// The mill is rated at 700 t/h, so the tonnage follows the hours.
	for _, e := range []struct {
		date           domain.BusinessDate
		reason         string
		startHr, endHr int
	}{
		{date: "2026-12-01", reason: "DT-BOILER", startHr: 2, endHr: 8},
		{date: "2026-12-02", reason: "DT-BOILER", startHr: 1, endHr: 5},
		{date: "2026-12-03", reason: "DT-POWER", startHr: 3, endHr: 5},
	} {
		day, err := time.Parse("2006-01-02", string(e.date))
		if err != nil {
			t.Fatalf("parse %s: %v", e.date, err)
		}
		_, err = h.store.Planning().SaveDowntime(ctx, domain.DowntimeEvent{
			FactoryID: h.seeded.FactoryID, LineID: h.lineID(t, "MILL-1"),
			BusinessDate: e.date,
			StartAt:      day.Add(time.Duration(e.startHr) * time.Hour),
			EndAt:        day.Add(time.Duration(e.endHr) * time.Hour),
			DurationHrs:  domain.DI(int64(e.endHr - e.startHr)),
			ReasonCode:   e.reason,
		}, "test")
		if err != nil {
			t.Fatalf("save downtime: %v", err)
		}
	}

	dash, err := h.analytics.Dashboard(h.as(auth.RoleExecutiveViewer),
		service.DashboardRequest{SeasonID: h.seeded.SeasonID})
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}

	if len(dash.Downtime.ByReason) != 2 {
		t.Fatalf("two distinct reasons, got %v", dash.Downtime.ByReason)
	}

	// Worst first. A Pareto that is not sorted is a bar chart.
	boiler, power := dash.Downtime.ByReason[0], dash.Downtime.ByReason[1]
	if boiler.ReasonCode != "DT-BOILER" || power.ReasonCode != "DT-POWER" {
		t.Fatalf("the reasons must be ranked by hours lost, got %s then %s",
			boiler.ReasonCode, power.ReasonCode)
	}
	if boiler.ReasonName != "Boiler problem" {
		t.Errorf("the reason name must come from master data, got %q", boiler.ReasonName)
	}
	if boiler.EventCount != 2 {
		t.Errorf("boiler events = %d, want 2", boiler.EventCount)
	}
	// 6 h + 4 h = 10 h of 12 h total, so 83.333 %.
	if !boiler.Hours.Equal(domain.D("10")) || !boiler.SharePct.Equal(domain.D("83.333")) {
		t.Errorf("boiler = %s h, %s %% of the total", boiler.Hours, boiler.SharePct)
	}
	// 10 h at 700 t/h.
	if !boiler.LostTons.Equal(domain.D("7000")) {
		t.Errorf("boiler lost tons = %s, want 7000.000", boiler.LostTons)
	}
	// The cumulative line is what makes it a Pareto: the last bar reaches 100 %.
	if !power.CumSharePct.Equal(domain.DI(100)) {
		t.Errorf("the cumulative share must end at 100, got %s", power.CumSharePct)
	}

	// The ranking has to add back to the headline figure, or the chart and the
	// KPI above it are describing different factories.
	hours := domain.Zero
	for _, r := range dash.Downtime.ByReason {
		hours = hours.Add(r.Hours)
	}
	if !hours.Equal(dash.Downtime.Hours) {
		t.Errorf("the reasons sum to %s h, the KPI says %s", hours, dash.Downtime.Hours)
	}
}

// lineID resolves a production line by its code.
func (h *harness) lineID(t *testing.T, code string) string {
	t.Helper()
	page, err := h.store.MasterData().Lines().List(h.as(auth.RoleProductionPlanner),
		store.ListOptions{Top: 100, ParentID: h.seeded.FactoryID})
	if err != nil {
		t.Fatalf("lines: %v", err)
	}
	for _, l := range page.Items {
		if l.Code == code {
			return l.ID
		}
	}
	t.Fatalf("no line %q is seeded", code)
	return ""
}

// TestTheCumulativeShareDoesNotOvershoot is the demonstration scenario's own
// downtime, which is what found this: 18, 7, 2 and 2 hours of 29 round to
// shares of 62.069, 24.138, 6.897 and 6.897, and those add up to 100.001.
//
// The cumulative share has to come from the running hours rather than from a
// sum of rounded percentages. A Pareto whose line finishes past 100 % is
// visibly wrong to the one person in the room who checks.
func TestTheCumulativeShareDoesNotOvershoot(t *testing.T) {
	h := newHarness(t, 0)
	ctx := h.as(auth.RoleShiftSupervisor)
	mill := h.lineID(t, "MILL-1")

	for i, e := range []struct {
		reason string
		hours  int
	}{
		{"DT-BOILER", 18}, {"DT-RAIN", 7}, {"DT-MILL", 2}, {"DT-POWER", 2},
	} {
		date := domain.BusinessDate("2026-12-0" + string(rune('1'+i)))
		day, err := time.Parse("2006-01-02", string(date))
		if err != nil {
			t.Fatalf("parse %s: %v", date, err)
		}
		if _, err := h.store.Planning().SaveDowntime(ctx, domain.DowntimeEvent{
			FactoryID: h.seeded.FactoryID, LineID: mill, BusinessDate: date,
			StartAt: day, EndAt: day.Add(time.Duration(e.hours) * time.Hour),
			DurationHrs: domain.DI(int64(e.hours)), ReasonCode: e.reason,
		}, "test"); err != nil {
			t.Fatalf("save downtime: %v", err)
		}
	}

	dash, err := h.analytics.Dashboard(h.as(auth.RoleExecutiveViewer),
		service.DashboardRequest{SeasonID: h.seeded.SeasonID})
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}

	ranked := dash.Downtime.ByReason
	if len(ranked) != 4 {
		t.Fatalf("four reasons, got %d", len(ranked))
	}

	// The rounded shares really do overshoot, which is the point of the test:
	// if they ever stop doing so this case is no longer exercising anything.
	naive := domain.Zero
	for _, r := range ranked {
		naive = naive.Add(r.SharePct)
	}
	if naive.Equal(domain.DI(100)) {
		t.Fatal("this case no longer exercises the rounding overshoot")
	}

	if last := ranked[len(ranked)-1].CumSharePct; !last.Equal(domain.DI(100)) {
		t.Errorf("the cumulative share ends at %s, want exactly 100 (the rounded shares sum to %s)",
			last, naive)
	}
	// And it never runs backwards or past 100 on the way there.
	previous := domain.Zero
	for _, r := range ranked {
		if r.CumSharePct.LessThan(previous) {
			t.Errorf("the cumulative share fell at %s", r.ReasonCode)
		}
		if r.CumSharePct.GreaterThan(domain.DI(100)) {
			t.Errorf("%s: cumulative share %s is past 100", r.ReasonCode, r.CumSharePct)
		}
		previous = r.CumSharePct
	}
}
