package domain

import "testing"

func eq(t *testing.T, got, want Dec, what string) {
	t.Helper()
	if !got.Equal(want) {
		t.Errorf("%s = %s, want %s", what, got, want)
	}
}

// ---------------------------------------------------------------------------
// Reference reconciliation (section 24 of the master prompt)
// ---------------------------------------------------------------------------

func TestReferenceScenarioReconciliation(t *testing.T) {
	t.Run("expected raw sugar", func(t *testing.T) {
		eq(t, ExpectedRawSugar(D("2300000"), D("11.00")), D("253000"), "2,300,000 x 11.00%")
	})
	t.Run("raw sugar storage capacity", func(t *testing.T) {
		eq(t, SumDec(D("45000"), D("65000")), D("110000"), "WH1 + WH2")
	})
	t.Run("finished goods capacity", func(t *testing.T) {
		eq(t, SumDec(D("22000"), D("47000")), D("69000"), "WH1 + WH3")
	})
	t.Run("finished goods total", func(t *testing.T) {
		eq(t, SumDec(D("106700"), D("133400"), D("2000")), D("242100"), "refined + white + super refined")
	})
	t.Run("raw sugar split", func(t *testing.T) {
		eq(t, SumDec(D("124950"), D("128050")), D("253000"), "direct to refining + to storage")
	})
	t.Run("jumbo bags for the planned tonnage", func(t *testing.T) {
		// 20,700 t at 1.10 t per bag, no scrap allowance.
		if got := RequiredPackages(D("20700"), D("1100"), Zero); got != 18819 {
			t.Errorf("jumbo bags = %d, want 18819", got)
		}
	})
	t.Run("jumbo packing days at 300 t/day", func(t *testing.T) {
		_, days, ok := ForecastCompletion("2026-12-01", D("20700"), D("300"))
		if !ok || days != 69 {
			t.Errorf("packing days = %d (ok=%v), want 69", days, ok)
		}
	})
}

// ---------------------------------------------------------------------------
// Target versus actual series
// ---------------------------------------------------------------------------

func TestBuildSeries(t *testing.T) {
	dates := []BusinessDate{"2026-12-01", "2026-12-02", "2026-12-03", "2026-12-04"}
	target := map[BusinessDate]Dec{
		"2026-12-01": D("16788"), "2026-12-02": D("16788"),
		"2026-12-03": D("16788"), "2026-12-04": D("16788"),
	}
	actual := map[BusinessDate]Dec{
		"2026-12-01": D("15000"), "2026-12-02": D("17500"), "2026-12-03": D("16788"),
		// 4 December has no actual yet.
	}
	pts := BuildSeries(dates, target, actual)

	eq(t, pts[0].CumTarget, D("16788.000"), "cum target day 1")
	eq(t, pts[0].Variance, D("-1788.000"), "variance day 1")
	eq(t, pts[2].CumActual, D("49288.000"), "cum actual day 3")
	eq(t, pts[2].CumTarget, D("50364.000"), "cum target day 3")
	eq(t, pts[2].CumVariance, D("-1076.000"), "cum variance day 3")
	// 49,288 / 50,364 = 97.8635533... %, rounded half away from zero.
	eq(t, pts[2].AchievementPct, D("97.864"), "achievement day 3")

	if pts[3].HasActual {
		t.Error("day 4 must be flagged as having no actual")
	}
	eq(t, pts[3].CumActual, D("49288.000"), "cum actual must not advance without an actual")
	eq(t, pts[3].AchievementPct, Zero, "achievement is not reported for a day without an actual")
}

func TestBuildSeriesEmptyAndZero(t *testing.T) {
	if got := BuildSeries(nil, nil, nil); len(got) != 0 {
		t.Errorf("empty series returned %d points", len(got))
	}
	pts := BuildSeries([]BusinessDate{"2026-12-01"},
		map[BusinessDate]Dec{"2026-12-01": Zero},
		map[BusinessDate]Dec{"2026-12-01": D("10")})
	eq(t, pts[0].AchievementPct, Zero, "division by a zero target must be zero, not a panic")
}

func TestRollingAverage(t *testing.T) {
	dates := []BusinessDate{"2026-12-01", "2026-12-02", "2026-12-03", "2026-12-04", "2026-12-05"}
	target := map[BusinessDate]Dec{}
	actual := map[BusinessDate]Dec{
		"2026-12-01": D("10"), "2026-12-02": D("20"), "2026-12-03": D("30"),
		"2026-12-04": D("40"), "2026-12-05": D("50"),
	}
	for _, d := range dates {
		target[d] = D("25")
	}
	avg := RollingAverage(BuildSeries(dates, target, actual), 3)
	eq(t, avg[0], D("10.000"), "window of one")
	eq(t, avg[2], D("20.000"), "(10+20+30)/3")
	eq(t, avg[4], D("40.000"), "(30+40+50)/3")
}

func TestRollingAverageIgnoresDaysWithoutActuals(t *testing.T) {
	dates := []BusinessDate{"2026-12-01", "2026-12-02", "2026-12-03"}
	pts := BuildSeries(dates, map[BusinessDate]Dec{}, map[BusinessDate]Dec{"2026-12-01": D("30")})
	avg := RollingAverage(pts, 7)
	eq(t, avg[2], D("30.000"), "only the recorded day counts")
}

func TestForecastCompletion(t *testing.T) {
	cases := []struct {
		name      string
		from      BusinessDate
		remaining string
		rate      string
		wantDate  BusinessDate
		wantDays  int
		wantOK    bool
	}{
		{"exact division", "2026-12-01", "20000", "1000", "2026-12-21", 20, true},
		{"rounds up a partial day", "2026-12-01", "20001", "1000", "2026-12-22", 21, true},
		{"already complete", "2026-12-01", "0", "1000", "2026-12-01", 0, true},
		{"negative remainder is complete", "2026-12-01", "-500", "1000", "2026-12-01", 0, true},
		{"zero rate never completes", "2026-12-01", "20000", "0", "", 0, false},
		{"negative rate never completes", "2026-12-01", "20000", "-5", "", 0, false},
		{"crosses a month boundary", "2026-12-20", "3000", "250", "2027-01-01", 12, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, days, ok := ForecastCompletion(c.from, D(c.remaining), D(c.rate))
			if ok != c.wantOK || days != c.wantDays || d != c.wantDate {
				t.Errorf("got (%s, %d, %v), want (%s, %d, %v)", d, days, ok, c.wantDate, c.wantDays, c.wantOK)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Crushing and recovery
// ---------------------------------------------------------------------------

func TestCrushingCapacityAndUtilisation(t *testing.T) {
	eq(t, EffectiveCrushCapacity(D("750"), D("24"), D("2.5")), D("16125.000"), "750 t/h over 21.5 h")
	eq(t, UtilisationPct(D("24"), D("2.5")), D("89.583"), "utilisation")
	eq(t, UtilisationPct(D("24"), D("30")), Zero, "stoppage beyond the shift floors at zero")
	eq(t, UtilisationPct(Zero, Zero), Zero, "no available hours is zero, not a panic")
}

func TestRecovery(t *testing.T) {
	eq(t, ActualRecoveryPct(D("1650"), D("15000")), D("11.000"), "recovery")
	eq(t, ActualRecoveryPct(D("1650"), Zero), Zero, "no cane crushed is zero, not a panic")
	eq(t, ExpectedRawSugar(D("16788.321"), D("11.25")), D("1888.686"), "expected raw sugar rounds to 3 dp")
}

func TestRecoveryVerdict(t *testing.T) {
	cases := []struct {
		rec  string
		want Severity
	}{
		{"11.00", SeveritySuccess},
		{"9.50", SeverityWarning},
		{"13.50", SeverityWarning},
		{"-1", SeverityError},
		{"101", SeverityError},
		{"9.80", SeveritySuccess},
	}
	for _, c := range cases {
		if got := RecoveryVerdict(D(c.rec), D("9.8"), D("13.0")); got != c.want {
			t.Errorf("RecoveryVerdict(%s) = %s, want %s", c.rec, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Stock ledger and capacity
// ---------------------------------------------------------------------------

func TestRollLedgerContinuity(t *testing.T) {
	phys := D("2380")
	movs := []LedgerMovement{
		{Date: "2026-12-01", ProductionReceipt: D("1800"), ShipmentQty: D("500")},
		{Date: "2026-12-02", ProductionReceipt: D("1800"), ShipmentQty: D("500"), ProcessLoss: D("20")},
		{Date: "2026-12-03", ProductionReceipt: D("1800"), RemeltIssue: D("900"), ShipmentQty: D("500"),
			PhysicalBalance: &phys},
	}
	days := RollLedger(Zero, D("110000"), movs)

	eq(t, days[0].EndingBalance, D("1300.000"), "day 1 ending")
	eq(t, days[1].BeginningBalance, D("1300.000"), "day 2 beginning = day 1 ending")
	eq(t, days[1].EndingBalance, D("2580.000"), "day 2 ending")
	eq(t, days[2].BeginningBalance, D("2580.000"), "day 3 beginning")
	eq(t, days[2].EndingBalance, D("2980.000"), "day 3 ending")

	// Continuity must hold for every date.
	for i := 1; i < len(days); i++ {
		if !days[i].BeginningBalance.Equal(days[i-1].EndingBalance) {
			t.Fatalf("continuity broken at %s", days[i].Date)
		}
	}
	if days[2].Reconciliation == nil {
		t.Fatal("reconciliation difference missing on the counted day")
	}
	eq(t, *days[2].Reconciliation, D("600.000"), "2,980 calculated - 2,380 counted")
	if days[0].Reconciliation != nil {
		t.Error("days without a physical count must not report a difference")
	}
}

func TestRollLedgerAdjustmentsAndHolds(t *testing.T) {
	movs := []LedgerMovement{
		{Date: "2026-12-01", ProductionReceipt: D("1000"), Adjustment: D("-40"), HoldQty: D("100")},
		{Date: "2026-12-02", TransferIn: D("200"), TransferOut: D("50"), RepackIn: D("30"), RepackOut: D("30")},
	}
	days := RollLedger(D("500"), D("2000"), movs)
	eq(t, days[0].EndingBalance, D("1460.000"), "opening + receipt - negative adjustment")
	eq(t, days[0].AvailableBalance, D("1360.000"), "available excludes quality hold")
	eq(t, days[0].CapacityUsePct, D("73.000"), "1,460 of 2,000 usable")
	eq(t, days[1].EndingBalance, D("1610.000"), "transfers and repack net")
}

func TestCapacityUseAndFirstBreach(t *testing.T) {
	eq(t, CapacityUsePct(D("99000"), D("110000")), D("90.000"), "capacity use")
	eq(t, CapacityUsePct(D("99000"), Zero), Zero, "no capacity configured is zero, not a panic")

	days := []LedgerDay{
		{Date: "2026-12-01", EndingBalance: D("80000")},
		{Date: "2026-12-02", EndingBalance: D("95000")},
		{Date: "2026-12-03", EndingBalance: D("99500")},
	}
	// 90 % of 110,000 = 99,000.
	d, ok := FirstBreach(days, D("99000"))
	if !ok || d != "2026-12-03" {
		t.Errorf("first breach = %s (%v), want 2026-12-03", d, ok)
	}
	if _, ok := FirstBreach(days, D("200000")); ok {
		t.Error("no breach expected below an unreachable limit")
	}
	if _, ok := FirstBreach(nil, D("1")); ok {
		t.Error("an empty ledger cannot breach")
	}
}

func TestRequiredDailyShipment(t *testing.T) {
	// 110,000 t of usable raw sugar capacity, 1,800 t/day arriving, opening
	// 100,000 t: on day 6 the stock would reach 110,800 t.
	receipts := make([]Dec, 10)
	for i := range receipts {
		receipts[i] = D("1800")
	}
	got := RequiredDailyShipment(D("100000"), D("110000"), receipts)
	// Binding day is the last one: (100,000 + 18,000 - 110,000)/10 = 800.
	eq(t, got, D("800.000"), "required daily shipment")

	// Comfortably inside capacity: nothing needs to move.
	eq(t, RequiredDailyShipment(D("1000"), D("110000"), receipts), Zero, "no shipment required")
	eq(t, RequiredDailyShipment(D("1000"), D("110000"), nil), Zero, "no horizon, no requirement")
}

func TestRequiredDailyShipmentEarlyPeakBinds(t *testing.T) {
	// A large receipt on day one and nothing afterwards: the first day sets
	// the requirement even though later days are quiet.
	receipts := []Dec{D("50000"), Zero, Zero, Zero}
	got := RequiredDailyShipment(D("70000"), D("110000"), receipts)
	// Day 1: (120,000-110,000)/1 = 10,000. Day 4: 10,000/4 = 2,500. Max = 10,000.
	eq(t, got, D("10000.000"), "early peak binds")
}

func TestShipmentToClearBy(t *testing.T) {
	receipts := []Dec{D("100"), D("100"), D("100")}
	eq(t, ShipmentToClearBy(D("900"), Zero, receipts, 3), D("400.000"), "clear 1,200 t in 3 days")
	eq(t, ShipmentToClearBy(D("900"), D("600"), receipts, 3), D("200.000"), "clear down to 600 t")
	eq(t, ShipmentToClearBy(D("100"), D("600"), receipts, 3), Zero, "already below target")
	eq(t, ShipmentToClearBy(D("900"), Zero, receipts, 0), Zero, "zero days is zero, not a panic")
}

func TestMassBalanceTolerance(t *testing.T) {
	diff := MassBalanceDiff(D("2980"), D("2975"))
	eq(t, diff, D("5.000"), "mass balance difference")
	if !WithinTolerance(diff, D("1800"), D("0.5")) {
		t.Error("5 t on a 1,800 t throughput is within 0.5 %")
	}
	if WithinTolerance(D("50"), D("1800"), D("0.5")) {
		t.Error("50 t on a 1,800 t throughput is outside 0.5 %")
	}
	if !WithinTolerance(D("-5"), D("1800"), D("0.5")) {
		t.Error("tolerance must be symmetric around zero")
	}
	if WithinTolerance(D("1"), Zero, D("0.5")) {
		t.Error("any difference against a zero base is outside tolerance")
	}
}

// ---------------------------------------------------------------------------
// Packaging materials
// ---------------------------------------------------------------------------

func TestRequiredPackages(t *testing.T) {
	cases := []struct {
		name  string
		tons  string
		netKg string
		scrap string
		want  int64
	}{
		{"50 kg bags, whole result", "100", "50", "0", 2000},
		{"50 kg bags with 0.5 % scrap", "100", "50", "0.5", 2010},
		{"jumbo bags round up", "20700", "1100", "0", 18819},
		{"one kilo packs", "1", "1", "0", 1000},
		{"a partial bag still needs a bag", "0.075", "50", "0", 2},
		{"zero tonnage needs nothing", "0", "50", "0", 0},
		{"invalid net weight yields nothing", "100", "0", "0", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := RequiredPackages(D(c.tons), D(c.netKg), D(c.scrap)); got != c.want {
				t.Errorf("RequiredPackages = %d, want %d", got, c.want)
			}
		})
	}
}

func TestPurchaseRequirement(t *testing.T) {
	eq(t, PurchaseRequirement(D("20000"), D("2000"), D("5000"), D("3000")), D("14000.000"), "shortage")
	eq(t, PurchaseRequirement(D("2000"), D("500"), D("5000"), D("0")), Zero, "covered by stock")
}

func TestSuggestedOrderDate(t *testing.T) {
	if got := SuggestedOrderDate("2026-12-20", 30); got != "2026-11-20" {
		t.Errorf("suggested order date = %s, want 2026-11-20", got)
	}
}

func TestRemeltInputFactor(t *testing.T) {
	eq(t, RemeltInput(D("1000"), D("1.05")), D("1050.000"), "1,000 t output needs 1,050 t raw sugar")
}

func TestConvertQty(t *testing.T) {
	eq(t, ConvertQty(D("1"), D("1000")), D("1000.000"), "tons to kilograms")
}

// ---------------------------------------------------------------------------
// Downtime
// ---------------------------------------------------------------------------

func TestDowntime(t *testing.T) {
	eq(t, AvailabilityPct(D("24"), D("3")), D("87.500"), "availability")
	eq(t, LostTons(D("3"), D("750")), D("2250.000"), "lost tons")
	eq(t, ThroughputPct(D("15750"), D("21"), D("750")), D("100.000"), "at rated throughput")
	eq(t, ThroughputPct(D("15000"), Zero, D("750")), Zero, "no run hours is zero, not a panic")
}

// ---------------------------------------------------------------------------
// Numeric helpers
// ---------------------------------------------------------------------------

func TestDecimalHelpers(t *testing.T) {
	eq(t, SafeDiv(D("10"), Zero), Zero, "safe division")
	eq(t, SafePct(D("1"), Zero), Zero, "safe percentage")
	eq(t, ClampNonNegative(D("-1")), Zero, "clamp")
	eq(t, MaxDec(D("1"), D("2")), D("2"), "max")
	eq(t, MinDec(D("1"), D("2")), D("1"), "min")

	// The precision that matters: 0.1 + 0.2 must be exactly 0.3.
	eq(t, D("0.1").Add(D("0.2")), D("0.3"), "decimal addition is exact")

	// A tonnage that binary floating point cannot hold exactly.
	eq(t, RoundQty(D("2300000").Mul(D("0.11"))), D("253000.000"), "no floating point drift")
}

func TestBusinessDate(t *testing.T) {
	d := BusinessDate("2026-12-31")
	if got := d.AddDays(1); got != "2027-01-01" {
		t.Errorf("AddDays across a year boundary = %s", got)
	}
	if got := BusinessDate("2026-12-01").DaysBetween("2026-12-31"); got != 30 {
		t.Errorf("DaysBetween = %d, want 30", got)
	}
	if BusinessDate("not-a-date").Valid() {
		t.Error("malformed dates must not validate")
	}
	if !d.Valid() {
		t.Error("an ISO date must validate")
	}
}

func TestValidityEffectiveDating(t *testing.T) {
	v := Validity{ValidFrom: "2026-01-01", ValidTo: "2026-12-31", Active: true}
	if !v.IsEffective("2026-06-01") {
		t.Error("inside the window")
	}
	if v.IsEffective("2025-12-31") || v.IsEffective("2027-01-01") {
		t.Error("outside the window")
	}
	v.Active = false
	if v.IsEffective("2026-06-01") {
		t.Error("an inactive record is never effective")
	}
	open := Validity{Active: true}
	if !open.IsEffective("2030-01-01") {
		t.Error("an open-ended record is always effective")
	}
}

func TestWarehouseUsableCapacity(t *testing.T) {
	w := Warehouse{CapacityTons: D("45000"), UsablePct: D("95")}
	eq(t, w.UsableCapacity(), D("42750.000"), "95 % of 45,000")
}
