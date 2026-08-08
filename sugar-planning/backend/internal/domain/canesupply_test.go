package domain_test

import (
	"testing"

	"github.com/kss/sugarplan/internal/domain"
)

// The cane supply calculations. Each is small; each is the difference between a
// plan somebody can keep and one they cannot.

func TestExpectedTonsIsAreaTimesYield(t *testing.T) {
	s := domain.CaneSource{Hectares: domain.D("4200"), ExpectedYieldTPH: domain.D("68")}
	if got := s.ExpectedTons(); !got.Equal(domain.D("285600")) {
		t.Errorf("4200 ha at 68 t/ha = %s, want 285600", got)
	}
}

func TestTransportCapacityIsMovementsTimesLoad(t *testing.T) {
	s := domain.CaneSource{TruckCapacityTons: domain.D("18"), TrucksPerDay: 360}
	if got := s.DailyTransportCapacity(); !got.Equal(domain.D("6480")) {
		t.Errorf("360 movements of 18 t = %s, want 6480", got)
	}
}

func TestTheHarvestWindowIsInclusive(t *testing.T) {
	// A block cut on the 1st and the 1st is cut for one day, not zero.
	e := domain.CaneSupplyEntry{HarvestFrom: "2026-12-01", HarvestTo: "2026-12-01"}
	if got := e.HarvestDays(); got != 1 {
		t.Errorf("a single-day window is %d days, want 1", got)
	}
	e = domain.CaneSupplyEntry{HarvestFrom: "2026-12-01", HarvestTo: "2027-01-15"}
	if got := e.HarvestDays(); got != 46 {
		t.Errorf("1 Dec to 15 Jan is %d days, want 46", got)
	}
}

func TestTheRequiredRateIsWhatTheHaulageIsJudgedAgainst(t *testing.T) {
	e := domain.CaneSupplyEntry{
		HarvestFrom: "2026-12-01", HarvestTo: "2027-01-15",
		CommittedTons: domain.D("285600"),
	}
	// 285,600 t across 46 days.
	if got := e.RequiredDailyRate(); !got.Equal(domain.D("6208.696")) {
		t.Errorf("required rate = %s, want 6208.696", got)
	}
}

func TestATripIsAWholeVehicle(t *testing.T) {
	// The last truck of the day travels whether it is full or not, so the
	// count rounds up. A queue planned on fractional vehicles is always one
	// vehicle short.
	for _, c := range []struct {
		tons, capacity string
		want           int
	}{
		{"36", "18", 2},
		{"37", "18", 3},  // the extra ton still needs a lorry
		{"0.5", "18", 1}, // and so does half a ton
		{"0", "18", 0},   // but nothing needs none
		{"100", "0", 0},  // a source with no vehicles cannot move anything
	} {
		if got := domain.TripsFor(domain.D(c.tons), domain.D(c.capacity)); got != c.want {
			t.Errorf("%s t at %s t a load = %d trips, want %d",
				c.tons, c.capacity, got, c.want)
		}
	}
}

func TestReconciliationAddsTheCommitmentsUp(t *testing.T) {
	sources := map[string]domain.CaneSource{
		"a": {Hectares: domain.D("100"), ExpectedYieldTPH: domain.D("70")},
		"b": {Hectares: domain.D("200"), ExpectedYieldTPH: domain.D("65")},
	}
	entries := []domain.CaneSupplyEntry{
		{SourceID: "a", CommittedTons: domain.D("7000")},
		{SourceID: "b", CommittedTons: domain.D("13000")},
	}
	r := domain.ReconcileSupply(domain.D("20000"), entries, sources)
	if !r.CommittedTons.Equal(domain.D("20000")) {
		t.Errorf("committed = %s, want 20000", r.CommittedTons)
	}
	if !r.Difference.Equal(domain.Zero) {
		t.Errorf("difference = %s, want 0", r.Difference)
	}
	if !r.CoveragePct.Equal(domain.D("100")) {
		t.Errorf("coverage = %s, want 100", r.CoveragePct)
	}
	// 100 ha at 70 plus 200 at 65 is 20,000 t of land against 20,000 committed.
	if !r.ExpectedTons.Equal(domain.D("20000")) {
		t.Errorf("expected tons = %s, want 20000", r.ExpectedTons)
	}
}

func TestAShortfallIsAnErrorAndASurplusIsOnlyAWarning(t *testing.T) {
	// Not the same thing. A mill short of cane stops; a mill with too much
	// leaves it standing, which costs money and does not stop anything.
	short := domain.ReconcileSupply(domain.D("100000"),
		[]domain.CaneSupplyEntry{{SourceID: "a", CommittedTons: domain.D("80000")}}, nil)
	warnings := domain.SupplyWarnings(short, nil, nil, domain.D("2"))
	if len(warnings) != 1 || warnings[0].Severity != domain.SeverityError {
		t.Fatalf("a 20 %% shortfall must be an error, got %+v", warnings)
	}

	over := domain.ReconcileSupply(domain.D("100000"),
		[]domain.CaneSupplyEntry{{SourceID: "a", CommittedTons: domain.D("120000")}}, nil)
	warnings = domain.SupplyWarnings(over, nil, nil, domain.D("2"))
	if len(warnings) != 1 || warnings[0].Severity != domain.SeverityWarning {
		t.Fatalf("a 20 %% surplus must be a warning, got %+v", warnings)
	}
}

func TestASmallGapIsInsideToleranceAndSaysNothing(t *testing.T) {
	// A season is contracted months ahead. Nobody expects it to land on the
	// tonne, and a system that complains about one per cent is a system people
	// learn to ignore.
	r := domain.ReconcileSupply(domain.D("2300000"),
		[]domain.CaneSupplyEntry{{SourceID: "a", CommittedTons: domain.D("2320000")}}, nil)
	if w := domain.SupplyWarnings(r, nil, nil, domain.D("2")); len(w) != 0 {
		t.Errorf("0.87 %% over is inside a 2 %% tolerance, got %+v", w)
	}
}

func TestASourceOverItsLandOrItsTrucksIsFlagged(t *testing.T) {
	sources := map[string]domain.CaneSource{
		// 100 ha at 60 t/ha is 6,000 t of land, committed to 9,000.
		"land": {ID: "land", Code: "OVER-LAND", Hectares: domain.D("100"),
			ExpectedYieldTPH:  domain.D("60"),
			TruckCapacityTons: domain.D("20"), TrucksPerDay: 500},
		// Plenty of land, nowhere near enough lorries.
		"haul": {ID: "haul", Code: "OVER-HAUL", Hectares: domain.D("1000"),
			ExpectedYieldTPH:  domain.D("60"),
			TruckCapacityTons: domain.D("12"), TrucksPerDay: 10},
	}
	entries := []domain.CaneSupplyEntry{
		{SourceID: "land", CommittedTons: domain.D("9000"),
			HarvestFrom: "2026-12-01", HarvestTo: "2026-12-30"},
		{SourceID: "haul", CommittedTons: domain.D("9000"),
			HarvestFrom: "2026-12-01", HarvestTo: "2026-12-30"},
	}
	r := domain.ReconcileSupply(domain.D("18000"), entries, sources)
	codes := map[string]bool{}
	for _, w := range domain.SupplyWarnings(r, entries, sources, domain.D("2")) {
		codes[w.Code] = true
	}
	if !codes["SUPPLY_OVER_YIELD"] {
		t.Error("a source committed to more than its land grows must be flagged")
	}
	if !codes["SUPPLY_OVER_HAULAGE"] {
		t.Error("a source with nowhere near enough lorries must be flagged")
	}
}

func TestADeliveryCannotBeMorePureThanItIs(t *testing.T) {
	// Polarisation is a percentage of the cane's mass. 140 % is not a reading
	// anybody could take, and it was accepted - while the same field on the
	// source the cane came from had been bounded in the database from the
	// start. The two disagreed, and the looser one held the real measurements.
	sound := domain.DailyCaneSupply{
		VersionID: "v", SourceID: "s", BusinessDate: "2026-12-05",
		Series: domain.SeriesActual, Tons: domain.D("4800"), Trips: 267,
		PolPct: domain.D("12.4"),
	}
	if err := sound.Validate(); err != nil {
		t.Fatalf("a 12.4 %% reading is ordinary: %v", err)
	}
	for _, pol := range []string{"140", "100.001", "-1"} {
		bad := sound
		bad.PolPct = domain.D(pol)
		if err := bad.Validate(); err == nil {
			t.Errorf("a polarisation of %s %% must be refused", pol)
		}
	}
	// The ends of the range are readings, not errors.
	for _, pol := range []string{"0", "100"} {
		edge := sound
		edge.PolPct = domain.D(pol)
		if err := edge.Validate(); err != nil {
			t.Errorf("a polarisation of %s %% must be accepted: %v", pol, err)
		}
	}
}

func TestValidationRefusesAWindowThatEndsBeforeItStarts(t *testing.T) {
	e := domain.CaneSupplyEntry{VersionID: "v", SourceID: "s",
		HarvestFrom: "2027-01-15", HarvestTo: "2026-12-01"}
	if err := e.Validate(); err == nil {
		t.Error("a harvest window that ends before it starts must be refused")
	}
}
