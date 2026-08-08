package domain

import (
	"errors"
	"math"
	"testing"
	"time"
)

// The formulas are the whole planning system in miniature: every engine, report and screen is
// arithmetic over these. The specification's own worked examples are the tests that matter most.

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func near(t *testing.T, got, want float64, what string) {
	t.Helper()
	if math.Abs(got-want) > 0.00005 {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

// ---------------------------------------------------------------- the worked example

// The specification states it outright: 2,600 ha at 4 ha per tractor per day over 90 working days
// is 7.22 tractors, and you cannot hire 0.22 of a tractor.
func TestRequiredTractors_theSpecificationsWorkedExample(t *testing.T) {
	got, err := RequiredTractors(2600, 4, 90)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 8 {
		t.Errorf("required tractors = %d, want 8", got)
	}
	near(t, ExactMachineRequirement(2600, 4, 90), 7.2222, "exact requirement")
}

func TestRequiredTractors_roundsUpEvenForATinyRemainder(t *testing.T) {
	// 360.0001 ha over exactly 360 ha of capacity still needs the second machine.
	got, _ := RequiredTractors(360.0001, 4, 90)
	if got != 2 {
		t.Errorf("required tractors = %d, want 2", got)
	}
	exact, _ := RequiredTractors(360, 4, 90)
	if exact != 1 {
		t.Errorf("a requirement that fits exactly needs one machine, got %d", exact)
	}
}

func TestRequiredTractors_refusesImpossibleArguments(t *testing.T) {
	if _, err := RequiredTractors(100, 0, 90); !errors.Is(err, ErrInvalidInput) {
		t.Error("a capacity of zero must be refused, not divided by")
	}
	if _, err := RequiredTractors(100, 4, 0); !errors.Is(err, ErrInvalidInput) {
		t.Error("zero working days must be refused")
	}
	if got, err := RequiredTractors(0, 4, 90); err != nil || got != 0 {
		t.Errorf("no area needs no tractors, got %d (%v)", got, err)
	}
}

// ---------------------------------------------------------------- projection

func TestHarvestableAreaAndProduction(t *testing.T) {
	area, err := HarvestableArea(128.8, 5)
	if err != nil {
		t.Fatal(err)
	}
	near(t, area, 122.36, "harvestable area")

	tons, err := ExpectedCaneProduction(area, 92)
	if err != nil {
		t.Fatal(err)
	}
	near(t, tons, 11257.12, "expected production")
}

func TestHarvestableArea_refusesALossOutsideItsRange(t *testing.T) {
	if _, err := HarvestableArea(100, 101); !errors.Is(err, ErrInvalidInput) {
		t.Error("a loss above 100% must be refused")
	}
	if _, err := HarvestableArea(100, -1); !errors.Is(err, ErrInvalidInput) {
		t.Error("a negative loss must be refused")
	}
	if area, _ := HarvestableArea(100, 100); area != 0 {
		t.Error("a total loss harvests nothing")
	}
}

// ---------------------------------------------------------------- the working calendar

func TestWorkingDays_skipsTheDaysTheCompanyDoesNotWork(t *testing.T) {
	// Monday 2 March 2026 to Sunday 8 March 2026 is seven days.
	start, end := date(2026, time.March, 2), date(2026, time.March, 8)

	if got := WorkingDays(start, end, DefaultCalendar()); got != 6 {
		t.Errorf("Saturdays worked, Sundays not = %d days, want 6", got)
	}
	if got := WorkingDays(start, end, WorkingCalendar{WorkOnSaturday: false, WorkOnSunday: false}); got != 5 {
		t.Errorf("neither weekend day worked = %d days, want 5", got)
	}
	if got := WorkingDays(start, end, WorkingCalendar{WorkOnSaturday: true, WorkOnSunday: true}); got != 7 {
		t.Errorf("a seven-day week = %d days, want 7", got)
	}
}

func TestWorkingDays_honoursHolidays(t *testing.T) {
	calendar := DefaultCalendar()
	calendar.Holidays["2026-03-04"] = true

	got := WorkingDays(date(2026, time.March, 2), date(2026, time.March, 6), calendar)
	if got != 4 {
		t.Errorf("a holiday in the week = %d days, want 4", got)
	}
}

func TestWorkingDays_ofAnInvertedRangeIsZero(t *testing.T) {
	if got := WorkingDays(date(2026, time.March, 8), date(2026, time.March, 2), DefaultCalendar()); got != 0 {
		t.Errorf("an end before its start = %d days, want 0", got)
	}
}

func TestAddWorkingDays_landsOnAWorkedDay(t *testing.T) {
	// Twelve working days from Monday 2 March, Sundays off: 2-7 is six, 9-14 is another six.
	end := AddWorkingDays(date(2026, time.March, 2), 12, DefaultCalendar())
	if !end.Equal(date(2026, time.March, 14)) {
		t.Errorf("end = %s, want 2026-03-14", end.Format("2006-01-02"))
	}
	// The count and the arithmetic have to agree, or a plan's duration and its dates diverge.
	if got := WorkingDays(date(2026, time.March, 2), end, DefaultCalendar()); got != 12 {
		t.Errorf("the range it produced counts %d working days, want 12", got)
	}
}

func TestAddWorkingDays_startingOnADayOff(t *testing.T) {
	// Sunday 1 March is not worked, so one working day from it is Monday the 2nd.
	end := AddWorkingDays(date(2026, time.March, 1), 1, DefaultCalendar())
	if !end.Equal(date(2026, time.March, 2)) {
		t.Errorf("end = %s, want 2026-03-02", end.Format("2006-01-02"))
	}
}

func TestNextWorkingDay(t *testing.T) {
	if got := NextWorkingDay(date(2026, time.March, 1), DefaultCalendar()); !got.Equal(date(2026, time.March, 2)) {
		t.Errorf("a Sunday moves to Monday, got %s", got.Format("2006-01-02"))
	}
	if got := NextWorkingDay(date(2026, time.March, 3), DefaultCalendar()); !got.Equal(date(2026, time.March, 3)) {
		t.Errorf("a working day stays put, got %s", got.Format("2006-01-02"))
	}
}

// ---------------------------------------------------------------- materials

func TestTotalMaterialRequirement(t *testing.T) {
	// 100 ha at 250 kg/ha with 5% waste: 25,000 base plus 1,250 waste.
	got, err := TotalMaterialRequirement(100, 250, 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	near(t, got, 26250, "total requirement")
}

func TestTotalMaterialRequirement_multipliesByTheApplications(t *testing.T) {
	one, _ := TotalMaterialRequirement(100, 2, 0, 1)
	three, _ := TotalMaterialRequirement(100, 2, 0, 3)
	near(t, three, one*3, "three applications")

	// Zero or a negative count means once, not never — a standard that forgot to say how many
	// times it is applied is still applied.
	none, _ := TotalMaterialRequirement(100, 2, 0, 0)
	near(t, none, one, "an unstated application count")
}

func TestWasteQuantity_refusesNegativeWaste(t *testing.T) {
	if _, err := WasteQuantity(100, -1); !errors.Is(err, ErrInvalidInput) {
		t.Error("negative waste would reduce the requirement below the standard")
	}
}

func TestSeedCaneAndChemical(t *testing.T) {
	near(t, RequiredSeedCane(128.8, 8), 1030.4, "seed cane")
	near(t, RequiredChemical(50, 2.5, 3), 375, "chemical over three applications")
}

// ---------------------------------------------------------------- stock

func TestStockArithmetic(t *testing.T) {
	net := NetAvailableQuantity(1000, 200, 150)
	near(t, net, 1050, "net available")
	near(t, ShortageQuantity(1200, net), 150, "shortage")
	near(t, SurplusQuantity(1200, net), 0, "surplus when short")
	near(t, SurplusQuantity(900, net), 150, "surplus")
	near(t, ShortageQuantity(900, net), 0, "shortage when covered")
}

// ---------------------------------------------------------------- fuel and labour

func TestFuelAndLabour(t *testing.T) {
	near(t, ProjectedFuelByArea(128.8, 22.9), 2949.52, "fuel by area")
	near(t, ProjectedFuelByHour(257.6, 11.45), 2949.52, "fuel by hour")
	near(t, RequiredLabourDays(128.8, 1.2), 154.56, "labour-days")

	workers, err := RequiredWorkers(154.56, 12)
	if err != nil {
		t.Fatal(err)
	}
	if workers != 13 {
		t.Errorf("required workers = %d, want 13", workers)
	}
}

func TestRequiredWorkers_refusesZeroDays(t *testing.T) {
	if _, err := RequiredWorkers(10, 0); !errors.Is(err, ErrInvalidInput) {
		t.Error("zero working days must be refused")
	}
	if got, err := RequiredWorkers(0, 10); err != nil || got != 0 {
		t.Errorf("no labour needs no workers, got %d (%v)", got, err)
	}
}

// ---------------------------------------------------------------- capacity

func TestEvaluateCapacity(t *testing.T) {
	cases := []struct {
		name                string
		required, available float64
		want                string
	}{
		{"nothing required", 0, 0, CapacitySufficient},
		{"nothing available", 100, 0, CapacityUnavailable},
		{"fully covered", 100, 100, CapacitySufficient},
		{"more than covered", 100, 120, CapacitySufficient},
		{"just inside the threshold", 100, 90, CapacityAtRisk},
		{"just outside it", 100, 89.9, CapacityShortage},
		{"barely covered at all", 100, 5, CapacityShortage},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := EvaluateCapacity(tc.required, tc.available, DefaultAtRiskThreshold); got != tc.want {
				t.Errorf("%v of %v = %s, want %s", tc.available, tc.required, got, tc.want)
			}
		})
	}
}

func TestCoveragePercent(t *testing.T) {
	near(t, CoveragePercent(100, 90), 90, "coverage")
	near(t, CoveragePercent(0, 50), 100, "nothing required is fully covered")
	// A requirement of almost nothing against a full store would otherwise report a number nobody
	// can read.
	near(t, CoveragePercent(0.0001, 1000), 999.99, "capped coverage")
}

// ---------------------------------------------------------------- variance

func TestVariance(t *testing.T) {
	near(t, AreaVariance(287.4, 367.9), -80.5, "area variance")
	near(t, MaterialVariance(1200, 1000), 200, "material variance")
	near(t, FuelVariance(2949.52, 3000), -50.48, "fuel variance")
	near(t, CompletionPercentage(45, 128.8), 34.9379, "completion")
	near(t, CompletionPercentage(45, 0), 0, "completion against nothing planned")
}

func TestScheduleVarianceDays(t *testing.T) {
	planned := date(2026, time.March, 14)
	late := date(2026, time.March, 18)

	if got := ScheduleVarianceDays(&late, &planned); got == nil || *got != 4 {
		t.Errorf("four days late = %v, want 4", got)
	}
	early := date(2026, time.March, 12)
	if got := ScheduleVarianceDays(&early, &planned); got == nil || *got != -2 {
		t.Errorf("two days early = %v, want -2", got)
	}
	// An activity still running is not on time and not late; it has no answer yet.
	if got := ScheduleVarianceDays(nil, &planned); got != nil {
		t.Errorf("an open activity has no variance, got %v", *got)
	}
}

// ---------------------------------------------------------------- units

func TestUnitConversionRoundTrips(t *testing.T) {
	base, err := ToBaseUnit(5, 1000) // five tonnes into kilogrammes
	if err != nil {
		t.Fatal(err)
	}
	near(t, base, 5000, "to base unit")

	back, err := ToAlternativeUnit(base, 1000)
	if err != nil {
		t.Fatal(err)
	}
	near(t, back, 5, "and back again")

	if _, err := ToBaseUnit(5, 0); !errors.Is(err, ErrInvalidInput) {
		t.Error("a conversion factor of zero must be refused")
	}
}

// ---------------------------------------------------------------- overlaps

func TestDateRangesOverlap(t *testing.T) {
	cases := []struct {
		name                       string
		aStart, aEnd, bStart, bEnd time.Time
		want                       bool
	}{
		{"apart", date(2026, 3, 1), date(2026, 3, 5), date(2026, 3, 6), date(2026, 3, 9), false},
		{"touching", date(2026, 3, 1), date(2026, 3, 5), date(2026, 3, 5), date(2026, 3, 9), true},
		{"contained", date(2026, 3, 1), date(2026, 3, 9), date(2026, 3, 3), date(2026, 3, 4), true},
		{"identical", date(2026, 3, 1), date(2026, 3, 5), date(2026, 3, 1), date(2026, 3, 5), true},
		{"before", date(2026, 3, 6), date(2026, 3, 9), date(2026, 3, 1), date(2026, 3, 5), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DateRangesOverlap(tc.aStart, tc.aEnd, tc.bStart, tc.bEnd); got != tc.want {
				t.Errorf("overlap = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTimeRangesOverlap_isHalfOpen(t *testing.T) {
	at := func(h, m int) time.Time { return time.Date(2026, 3, 2, h, m, 0, 0, time.UTC) }

	// A machine freed at noon can be booked again from noon.
	if TimeRangesOverlap(at(8, 0), at(12, 0), at(12, 0), at(16, 0)) {
		t.Error("bookings that meet at noon do not overlap")
	}
	if !TimeRangesOverlap(at(8, 0), at(12, 1), at(12, 0), at(16, 0)) {
		t.Error("one minute of overlap is an overlap")
	}
}

// ---------------------------------------------------------------- rounding

func TestRound4_matchesTheDotNetImplementationItWasPortedFrom(t *testing.T) {
	// .NET rounds halves away from zero; Go's math.Round does too, but the naive
	// float64 multiply-and-truncate does not, which is why Round4 exists.
	cases := []struct{ in, want float64 }{
		{1.00005, 1.0001},
		{-1.00005, -1.0001},
		{2.00004999, 2.0},
		{0.12345, 0.1235},
		{-0.12345, -0.1235},
	}
	for _, tc := range cases {
		if got := Round4(tc.in); math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("Round4(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestRound4_keepsASumOfStoredFiguresEqualToTheStoredSum(t *testing.T) {
	// Three thirds of a hectare, each stored at four decimals, must add up to what the database
	// would hold — not to 0.9999999999999999.
	third := Round4(1.0 / 3.0)
	total := Round4(third * 3)
	if total != 0.9999 {
		t.Errorf("three stored thirds = %v, want 0.9999", total)
	}
}
