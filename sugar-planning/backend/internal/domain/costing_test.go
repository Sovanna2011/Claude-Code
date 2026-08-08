package domain_test

import (
	"errors"
	"testing"

	"github.com/kss/sugarplan/internal/domain"
)

// costFixture is a small but realistic cost structure: cane paid per ton
// crushed, fuel per hour run, bags per ton of sugar, and salaried staff per
// calendar day.
func costFixture() ([]domain.CostElement, []domain.CostRate) {
	active := domain.Validity{Active: true}
	elements := []domain.CostElement{
		{ID: "e-cane", Code: "CANE", Name: "Cane payment", Category: domain.CategoryCane,
			Driver: domain.DriverCaneTon, Variable: true, Validity: active},
		{ID: "e-fuel", Code: "FUEL", Name: "Boiler fuel", Category: domain.CategoryEnergy,
			Driver: domain.DriverRunHour, Variable: true, Validity: active},
		{ID: "e-bags", Code: "BAGS", Name: "Packaging", Category: domain.CategoryPackaging,
			Driver: domain.DriverSugarTon, Variable: true, Validity: active},
		{ID: "e-staff", Code: "STAFF", Name: "Salaried staff", Category: domain.CategoryLabour,
			Driver: domain.DriverCalendarDay, Variable: false, Validity: active},
	}
	rates := []domain.CostRate{
		{ElementID: "e-cane", RateType: domain.RateStandard, Rate: domain.D("22.50"),
			Currency: "USD", ValidFrom: "2026-12-01"},
		{ElementID: "e-fuel", RateType: domain.RateStandard, Rate: domain.D("140.00"),
			Currency: "USD", ValidFrom: "2026-12-01"},
		{ElementID: "e-bags", RateType: domain.RateStandard, Rate: domain.D("8.00"),
			Currency: "USD", ValidFrom: "2026-12-01"},
		{ElementID: "e-staff", RateType: domain.RateStandard, Rate: domain.D("3100.00"),
			Currency: "USD", ValidFrom: "2026-12-01"},
	}
	return elements, rates
}

func TestCostRunPricesEachElementAgainstItsDriver(t *testing.T) {
	elements, rates := costFixture()

	out := domain.CalculateCost(domain.CostInput{
		Elements: elements, Rates: rates, On: "2026-12-31", Currency: "USD",
		Planned: domain.DriverQuantities{
			CaneTons: domain.D("100000"), SugarTons: domain.D("11000"),
			RunHours: domain.D("720"), CalendarDays: domain.D("30"),
		},
		Actual: domain.DriverQuantities{
			CaneTons: domain.D("100000"), SugarTons: domain.D("11000"),
			RunHours: domain.D("720"), CalendarDays: domain.D("30"),
		},
	})

	if len(out.Lines) != 4 {
		t.Fatalf("lines = %d, want 4", len(out.Lines))
	}
	// Each element is priced against its own driver, not against a single
	// volume: cane on tons crushed, fuel on hours run, bags on sugar made,
	// staff on days elapsed.
	want := map[string]string{
		"CANE":  "2250000", // 100,000 t × 22.50
		"FUEL":  "100800",  // 720 h × 140.00
		"BAGS":  "88000",   // 11,000 t × 8.00
		"STAFF": "93000",   // 30 d × 3,100.00
	}
	for _, line := range out.Lines {
		if got := line.PlannedCost; !got.Equal(domain.D(want[line.ElementCode])) {
			t.Errorf("%s planned cost = %s, want %s", line.ElementCode, got, want[line.ElementCode])
		}
	}

	// 2,250,000 + 100,800 + 88,000 + 93,000
	if !out.Totals.PlannedCost.Equal(domain.D("2531800")) {
		t.Errorf("planned cost = %s, want 2,531,800", out.Totals.PlannedCost)
	}
	// 2,531,800 / 11,000 t
	if !out.Totals.PlannedUnitCost.Equal(domain.D("230.16")) {
		t.Errorf("planned unit cost = %s, want 230.16", out.Totals.PlannedUnitCost)
	}
	// Staff is the only fixed element.
	if !out.Totals.FixedCost.Equal(domain.D("93000")) {
		t.Errorf("fixed cost = %s, want 93,000", out.Totals.FixedCost)
	}
	if !out.Totals.VariableCost.Equal(domain.D("2438800")) {
		t.Errorf("variable cost = %s, want 2,438,800", out.Totals.VariableCost)
	}
	// Nothing moved, so nothing varied.
	if !out.Totals.TotalVariance.IsZero() {
		t.Errorf("total variance = %s, want 0", out.Totals.TotalVariance)
	}
}

// TestVarianceSplitsIntoRateAndUsageAndTheHalvesReconcile is the point of the
// whole model: a controller must be able to say how much of an overspend was
// paying more and how much was using more, and the two must add up to the
// number on the report.
func TestVarianceSplitsIntoRateAndUsageAndTheHalvesReconcile(t *testing.T) {
	elements, rates := costFixture()
	// Fuel was invoiced at 155.00 rather than the 140.00 budgeted, and the mill
	// ran 800 hours rather than 720.
	rates = append(rates, domain.CostRate{
		ElementID: "e-fuel", RateType: domain.RateActual, Rate: domain.D("155.00"),
		Currency: "USD", ValidFrom: "2026-12-01",
	})

	out := domain.CalculateCost(domain.CostInput{
		Elements: elements, Rates: rates, On: "2026-12-31", Currency: "USD",
		Planned: domain.DriverQuantities{
			CaneTons: domain.D("100000"), SugarTons: domain.D("11000"),
			RunHours: domain.D("720"), CalendarDays: domain.D("30"),
		},
		Actual: domain.DriverQuantities{
			CaneTons: domain.D("100000"), SugarTons: domain.D("11000"),
			RunHours: domain.D("800"), CalendarDays: domain.D("30"),
		},
	})

	var fuel domain.CostLine
	for _, line := range out.Lines {
		if line.ElementCode == "FUEL" {
			fuel = line
		}
	}
	// Rate variance: (155.00 - 140.00) × 800 h = 12,000
	if !fuel.RateVariance.Equal(domain.D("12000")) {
		t.Errorf("fuel rate variance = %s, want 12,000", fuel.RateVariance)
	}
	// Usage variance: (800 - 720) h × 140.00 = 11,200
	if !fuel.UsageVariance.Equal(domain.D("11200")) {
		t.Errorf("fuel usage variance = %s, want 11,200", fuel.UsageVariance)
	}
	// Actual 800 × 155.00 = 124,000 against a plan of 100,800 → 23,200
	if !fuel.TotalVariance.Equal(domain.D("23200")) {
		t.Errorf("fuel total variance = %s, want 23,200", fuel.TotalVariance)
	}
	if !fuel.RateVariance.Add(fuel.UsageVariance).Equal(fuel.TotalVariance) {
		t.Error("the two halves must reconstruct the total")
	}

	if !domain.VarianceCheck(out.Totals.ActualCost, out.Totals.PlannedCost,
		out.Totals.RateVariance, out.Totals.UsageVariance) {
		t.Errorf("the run does not reconcile: actual %s, planned %s, rate %s, usage %s",
			out.Totals.ActualCost, out.Totals.PlannedCost,
			out.Totals.RateVariance, out.Totals.UsageVariance)
	}
	for _, w := range out.Warnings {
		if w.Code == "COST_VARIANCE_ROUNDING" {
			t.Errorf("unexpected rounding remainder: %s", w.Detail)
		}
	}
}

// TestUsageVarianceIsValuedAtStandard pins the accounting convention. Valuing
// the extra quantity at the actual rate would count the price difference twice
// and the halves would no longer add up.
func TestUsageVarianceIsValuedAtStandard(t *testing.T) {
	got := domain.UsageVariance(domain.D("800"), domain.D("720"), domain.D("140.00"))
	if !got.Equal(domain.D("11200")) {
		t.Errorf("usage variance = %s, want 80 h × the standard 140.00 = 11,200", got)
	}
	// Had it been valued at the actual 155.00 it would read 12,400, and
	// 12,000 + 12,400 = 24,400 against a real difference of 23,200.
	if got.Equal(domain.D("12400")) {
		t.Error("the usage variance must not be valued at the actual rate")
	}
}

func TestAMissingStandardRateIsReportedRatherThanCostedAsZero(t *testing.T) {
	elements, rates := costFixture()
	// Drop the fuel rate entirely.
	kept := rates[:0]
	for _, r := range rates {
		if r.ElementID != "e-fuel" {
			kept = append(kept, r)
		}
	}

	out := domain.CalculateCost(domain.CostInput{
		Elements: elements, Rates: kept, On: "2026-12-31", Currency: "USD",
		Planned: domain.DriverQuantities{RunHours: domain.D("720"), SugarTons: domain.D("11000")},
		Actual:  domain.DriverQuantities{RunHours: domain.D("720"), SugarTons: domain.D("11000")},
	})

	var fuel domain.CostLine
	for _, line := range out.Lines {
		if line.ElementCode == "FUEL" {
			fuel = line
		}
	}
	if !fuel.PlannedCost.IsZero() {
		t.Errorf("with no rate the cost is zero, got %s", fuel.PlannedCost)
	}
	if len(fuel.Missing) == 0 {
		t.Error("the line must say the rate is missing, not look free")
	}
	found := false
	for _, w := range out.Warnings {
		if w.Code == "COST_NO_STANDARD_RATE" {
			found = true
		}
	}
	if !found {
		t.Errorf("a missing rate must raise a warning, got %+v", out.Warnings)
	}
}

// TestAnUninvoicedElementIsChargedAtStandard is the ordinary accounting
// treatment before the invoice arrives: the quantity is known, the price is
// assumed, and the rate variance stays zero until somebody knows better.
func TestAnUninvoicedElementIsChargedAtStandard(t *testing.T) {
	elements, rates := costFixture()

	out := domain.CalculateCost(domain.CostInput{
		Elements: elements, Rates: rates, On: "2026-12-31", Currency: "USD",
		Planned: domain.DriverQuantities{CaneTons: domain.D("100000"), SugarTons: domain.D("11000")},
		Actual:  domain.DriverQuantities{CaneTons: domain.D("105000"), SugarTons: domain.D("11500")},
	})

	for _, line := range out.Lines {
		if line.ElementCode != "CANE" {
			continue
		}
		if !line.ActualRate.Equal(line.StandardRate) {
			t.Errorf("actual rate = %s, want the standard %s", line.ActualRate, line.StandardRate)
		}
		if !line.RateVariance.IsZero() {
			t.Errorf("rate variance = %s, want 0 while the price is still assumed", line.RateVariance)
		}
		// 5,000 extra tons at the standard 22.50
		if !line.UsageVariance.Equal(domain.D("112500")) {
			t.Errorf("usage variance = %s, want 112,500", line.UsageVariance)
		}
	}
}

func TestTheLaterRateInForceWins(t *testing.T) {
	rates := []domain.CostRate{
		{ElementID: "e-fuel", RateType: domain.RateStandard, Rate: domain.D("140.00"),
			ValidFrom: "2026-12-01"},
		{ElementID: "e-fuel", RateType: domain.RateStandard, Rate: domain.D("152.00"),
			ValidFrom: "2027-01-15"},
	}

	before, ok := domain.RateOn(rates, "e-fuel", domain.RateStandard, "2027-01-10")
	if !ok || !before.Rate.Equal(domain.D("140.00")) {
		t.Errorf("before the rise the rate is 140.00, got %+v", before)
	}
	after, ok := domain.RateOn(rates, "e-fuel", domain.RateStandard, "2027-02-01")
	if !ok || !after.Rate.Equal(domain.D("152.00")) {
		t.Errorf("after the rise the rate is 152.00, got %+v", after)
	}
	if _, ok := domain.RateOn(rates, "e-fuel", domain.RateActual, "2027-02-01"); ok {
		t.Error("a standard rate must not answer as an actual one")
	}
}

func TestCurrencyConversionUsesTheRateInForceAndItsInverse(t *testing.T) {
	rates := []domain.ExchangeRate{
		{FromCurrency: "USD", ToCurrency: "KHR", Rate: domain.D("4100"), ValidFrom: "2026-12-01"},
		{FromCurrency: "USD", ToCurrency: "KHR", Rate: domain.D("4150"), ValidFrom: "2027-02-01"},
	}

	got, err := domain.Convert(domain.D("100"), "USD", "KHR", rates, "2026-12-15")
	if err != nil || !got.Equal(domain.D("410000")) {
		t.Errorf("100 USD on 15 Dec = %s (%v), want 410,000 KHR", got, err)
	}
	got, err = domain.Convert(domain.D("100"), "USD", "KHR", rates, "2027-03-01")
	if err != nil || !got.Equal(domain.D("415000")) {
		t.Errorf("100 USD in March = %s (%v), want 415,000 KHR", got, err)
	}

	// A site that entered USD→KHR should not have to enter KHR→USD as well.
	got, err = domain.Convert(domain.D("410000"), "KHR", "USD", rates, "2026-12-15")
	if err != nil || !got.Equal(domain.D("100")) {
		t.Errorf("410,000 KHR = %s (%v), want 100 USD by the inverse", got, err)
	}

	// The identity needs no rate at all, so a single-currency site configures
	// nothing.
	got, err = domain.Convert(domain.D("100"), "USD", "USD", nil, "2026-12-15")
	if err != nil || !got.Equal(domain.D("100")) {
		t.Errorf("the identity conversion = %s (%v)", got, err)
	}

	if _, err := domain.Convert(domain.D("100"), "USD", "EUR", rates, "2026-12-15"); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("a missing pair must be an error, got %v", err)
	}
}

func TestACostRunReportsInOneCurrency(t *testing.T) {
	active := domain.Validity{Active: true}
	elements := []domain.CostElement{
		{ID: "e-cane", Code: "CANE", Name: "Cane payment", Category: domain.CategoryCane,
			Driver: domain.DriverCaneTon, Variable: true, Validity: active},
	}
	// Cane is paid in riel; the board reports in dollars.
	rates := []domain.CostRate{
		{ElementID: "e-cane", RateType: domain.RateStandard, Rate: domain.D("92250"),
			Currency: "KHR", ValidFrom: "2026-12-01"},
	}
	exchange := []domain.ExchangeRate{
		{FromCurrency: "USD", ToCurrency: "KHR", Rate: domain.D("4100"), ValidFrom: "2026-12-01"},
	}

	out := domain.CalculateCost(domain.CostInput{
		Elements: elements, Rates: rates, Exchange: exchange,
		On: "2026-12-31", Currency: "USD",
		Planned: domain.DriverQuantities{CaneTons: domain.D("1000"), SugarTons: domain.D("110")},
		Actual:  domain.DriverQuantities{CaneTons: domain.D("1000"), SugarTons: domain.D("110")},
	})

	// 92,250 KHR / 4,100 = 22.50 USD per ton
	if !out.Lines[0].StandardRate.Equal(domain.D("22.5")) {
		t.Errorf("converted rate = %s, want 22.50 USD", out.Lines[0].StandardRate)
	}
	if !out.Totals.PlannedCost.Equal(domain.D("22500")) {
		t.Errorf("planned cost = %s, want 22,500 USD", out.Totals.PlannedCost)
	}
}

func TestAnUnconvertibleRateIsAnErrorOnTheRunNotASilentZero(t *testing.T) {
	elements := []domain.CostElement{
		{ID: "e-cane", Code: "CANE", Name: "Cane payment", Category: domain.CategoryCane,
			Driver: domain.DriverCaneTon, Variable: true, Validity: domain.Validity{Active: true}},
	}
	rates := []domain.CostRate{
		{ElementID: "e-cane", RateType: domain.RateStandard, Rate: domain.D("20"),
			Currency: "EUR", ValidFrom: "2026-12-01"},
	}

	out := domain.CalculateCost(domain.CostInput{
		Elements: elements, Rates: rates, On: "2026-12-31", Currency: "USD",
		Planned: domain.DriverQuantities{CaneTons: domain.D("1000")},
		Actual:  domain.DriverQuantities{CaneTons: domain.D("1000")},
	})

	found := false
	for _, w := range out.Warnings {
		if w.Code == "COST_NO_EXCHANGE_RATE" && w.Severity == domain.SeverityError {
			found = true
		}
	}
	if !found {
		t.Errorf("an unconvertible rate must be an error on the run, got %+v", out.Warnings)
	}
}

func TestAFixedSeasonalCostIsChargedOnce(t *testing.T) {
	elements := []domain.CostElement{
		{ID: "e-shut", Code: "SHUTDOWN", Name: "Annual shutdown", Category: domain.CategoryMaintenance,
			Driver: domain.DriverFixedSeason, Variable: false, Validity: domain.Validity{Active: true}},
	}
	rates := []domain.CostRate{
		{ElementID: "e-shut", RateType: domain.RateStandard, Rate: domain.D("450000"),
			Currency: "USD", ValidFrom: "2026-12-01"},
	}

	out := domain.CalculateCost(domain.CostInput{
		Elements: elements, Rates: rates, On: "2026-12-31", Currency: "USD",
		Planned: domain.DriverQuantities{CaneTons: domain.D("2300000"), SugarTons: domain.D("242100")},
		Actual:  domain.DriverQuantities{CaneTons: domain.D("2300000"), SugarTons: domain.D("242100")},
	})

	// The lump sum is charged once however much cane went through.
	if !out.Totals.PlannedCost.Equal(domain.D("450000")) {
		t.Errorf("planned cost = %s, want the lump sum of 450,000", out.Totals.PlannedCost)
	}
	if !out.Lines[0].PlannedQty.Equal(domain.DI(1)) {
		t.Errorf("driver quantity = %s, want 1", out.Lines[0].PlannedQty)
	}
}

func TestAnInactiveElementIsNotCosted(t *testing.T) {
	elements, rates := costFixture()
	elements[1].Active = false // fuel is retired

	out := domain.CalculateCost(domain.CostInput{
		Elements: elements, Rates: rates, On: "2026-12-31", Currency: "USD",
		Planned: domain.DriverQuantities{RunHours: domain.D("720")},
		Actual:  domain.DriverQuantities{RunHours: domain.D("720")},
	})

	for _, line := range out.Lines {
		if line.ElementCode == "FUEL" {
			t.Error("a retired element must not appear on the run")
		}
	}
}

func TestUnitCostSurvivesAZeroOutput(t *testing.T) {
	// Day one of a season: the cost is real, the output is not yet.
	if got := domain.UnitCost(domain.D("93000"), domain.Zero); !got.IsZero() {
		t.Errorf("unit cost with no output = %s, want 0 rather than an error", got)
	}
}

// TestTheVarianceSplitAlwaysReconciles is the property SplitVariance exists for.
// The identity is exact before rounding, but rounding each half to the cent
// independently breaks it, and a report whose columns do not add up is one
// nobody trusts. The quantities below are chosen to produce a remainder.
func TestTheVarianceSplitAlwaysReconciles(t *testing.T) {
	cases := []struct{ actualRate, standardRate, actualQty, plannedQty string }{
		{"155.005000", "140.003000", "800.333", "720.777"},
		{"1.150000", "1.080000", "163333.333", "170000.001"},
		{"0.000001", "0.000002", "999999.999", "1.001"},
		{"22.505000", "22.500000", "2300000.123", "2299999.877"},
	}
	for _, c := range cases {
		actualRate, standardRate := domain.D(c.actualRate), domain.D(c.standardRate)
		actualQty, plannedQty := domain.D(c.actualQty), domain.D(c.plannedQty)

		actualCost := domain.ElementCost(actualRate, actualQty)
		plannedCost := domain.ElementCost(standardRate, plannedQty)
		total, rate, usage := domain.SplitVariance(
			actualCost, plannedCost, actualRate, standardRate, actualQty)

		if !rate.Add(usage).Equal(total) {
			t.Errorf("%+v: rate %s + usage %s != total %s", c, rate, usage, total)
		}
		if !total.Equal(domain.RoundMoney(actualCost.Sub(plannedCost))) {
			t.Errorf("%+v: total %s is not actual - planned", c, total)
		}
		// The usage half absorbs the remainder, and the remainder is small: it
		// must stay a rounding artefact rather than become a real number.
		byFormula := domain.UsageVariance(actualQty, plannedQty, standardRate)
		if usage.Sub(byFormula).Abs().GreaterThan(domain.D("0.01")) {
			t.Errorf("%+v: the derived usage variance %s is %s away from the formula's %s",
				c, usage, usage.Sub(byFormula).Abs(), byFormula)
		}
	}
}
