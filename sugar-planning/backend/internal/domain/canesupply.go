package domain

import "fmt"

// Where the cane comes from.
//
// Section 5 of the specification asks the crushing plan to carry "cane
// source/zone/farm, delivery schedule, transport capacity, queue, and expected
// quality". Until now the plan answered how much cane the mill would crush each
// day and could not answer where any of it came from: 2,300,000 t arrived as a
// single assumption with nothing behind it.
//
// These three types are that missing half.
//
//	CaneSource       a farm, estate or outgrower - master data
//	CaneSupplyEntry  what one source is committed to for one season
//	DailyCaneSupply  the delivery schedule, a row per source per day
//
// The daily rows are generated from the entries the same way the crushing plan
// is generated from the product mix, and they carry the same PLAN and ACTUAL
// series, so a schedule and what actually arrived at the gate are the same
// shape and never overwrite each other.

// SourceType says who owns the cane and therefore how it is paid for and how
// much notice the mill has of it arriving.
type SourceType string

const (
	// SourceEstate is land the company farms itself. The mill controls the
	// harvest date, so an estate block is what a planner moves first when the
	// schedule has to change.
	SourceEstate SourceType = "ESTATE"
	// SourceContract is a grower under a season contract: a committed tonnage
	// and an agreed window, but somebody else's field.
	SourceContract SourceType = "CONTRACT"
	// SourceOutgrower is an independent smallholder selling at the gate. The
	// tonnage is a forecast rather than a commitment, which is why the plan
	// treats it as the least reliable band of supply.
	SourceOutgrower SourceType = "OUTGROWER"
)

// ValidSourceType reports whether the value is one the plan understands.
func ValidSourceType(t SourceType) bool {
	switch t {
	case SourceEstate, SourceContract, SourceOutgrower:
		return true
	}
	return false
}

// CaneSource is a farm, estate block or outgrower group that supplies cane.
type CaneSource struct {
	ID        string     `json:"id"`
	FactoryID string     `json:"factoryId"`
	Code      string     `json:"code"`
	Name      string     `json:"name"`
	Type      SourceType `json:"sourceType"`
	// Zone is the harvesting district. It is what a haulage schedule is built
	// around, because trucks are allocated to a zone rather than to a field.
	Zone string `json:"zone,omitempty"`
	// DistanceKm drives haulage cost and, more importantly, staleness: cane
	// deteriorates from the moment it is cut, so distance is a quality input
	// and not only a cost one.
	DistanceKm Dec    `json:"distanceKm"`
	Hectares   Dec    `json:"hectares"`
	Variety    string `json:"variety,omitempty"`
	// ExpectedYieldTPH is tons of cane per hectare. With Hectares it gives the
	// tonnage this source can be expected to produce, which is the figure a
	// season commitment should be sense-checked against.
	ExpectedYieldTPH Dec `json:"expectedYieldTonsPerHectare"`
	// ExpectedPolPct is the polarisation the laboratory expects from this
	// source. A block that reliably comes in low is worth knowing about before
	// it is scheduled into a week that needs high-purity cane.
	ExpectedPolPct Dec `json:"expectedPolPct"`
	// TruckCapacityTons and TrucksPerDay are the transport capacity: the most
	// this source can physically deliver in a day however much is standing in
	// the field. TrucksPerDay counts vehicle movements rather than vehicles -
	// a lorry doing three round trips is three movements, and movements are
	// what the gate and the weighbridge actually see.
	TruckCapacityTons Dec `json:"truckCapacityTons"`
	TrucksPerDay      int `json:"trucksPerDay"`
	Validity
	AuditFields
}

// ExpectedTons is what the area should yield: hectares times yield per hectare.
//
// C44. This is the sense check on a season commitment - a source contracted
// for more than its land can grow is a plan that fails in the field rather
// than in the mill.
func (c CaneSource) ExpectedTons() Dec {
	return RoundQty(c.Hectares.Mul(c.ExpectedYieldTPH))
}

// DailyTransportCapacity is the most this source can move in a day.
//
// C45. Movements times what a vehicle carries. A commitment that needs more than
// this every day of its window cannot be delivered whatever the field holds,
// and the generator says so.
func (c CaneSource) DailyTransportCapacity() Dec {
	return RoundQty(c.TruckCapacityTons.Mul(DI(int64(c.TrucksPerDay))))
}

// Validate checks a source before it is stored.
func (c CaneSource) Validate() error {
	v := &ValidationError{}
	if c.FactoryID == "" {
		v.Add("factoryId", "REQUIRED", "a cane source belongs to a factory")
	}
	if c.Code == "" {
		v.Add("code", "REQUIRED", "a cane source needs a code")
	}
	if c.Name == "" {
		v.Add("name", "REQUIRED", "a cane source needs a name")
	}
	if !ValidSourceType(c.Type) {
		v.Add("sourceType", "INVALID",
			"the source type is ESTATE, CONTRACT or OUTGROWER")
	}
	for _, f := range []struct {
		name string
		val  Dec
	}{
		{"hectares", c.Hectares}, {"distanceKm", c.DistanceKm},
		{"expectedYieldTonsPerHectare", c.ExpectedYieldTPH},
		{"expectedPolPct", c.ExpectedPolPct}, {"truckCapacityTons", c.TruckCapacityTons},
	} {
		if f.val.IsNegative() {
			v.Add(f.name, "NEGATIVE", f.name+" cannot be negative")
		}
	}
	if c.TrucksPerDay < 0 {
		v.Add("trucksPerDay", "NEGATIVE", "the truck count cannot be negative")
	}
	if c.ExpectedPolPct.GreaterThan(DI(100)) {
		v.Add("expectedPolPct", "OUT_OF_RANGE", "polarisation is a percentage")
	}
	return v.OrNil()
}

// CaneSupplyEntry is what one source is committed to for one season, and the
// window it will be cut in.
//
// It is the cane-side counterpart of ProductMixEntry: the planner states the
// tonnage and the window, and the generator turns that into daily rows.
type CaneSupplyEntry struct {
	ID        string `json:"id"`
	VersionID string `json:"versionId"`
	SourceID  string `json:"sourceId"`
	// HarvestFrom and HarvestTo are the window this source can be cut in.
	// Outside it the cane is either not ready or already over-ripe, so the
	// generator schedules nothing there however much tonnage is outstanding.
	HarvestFrom   BusinessDate `json:"harvestFrom"`
	HarvestTo     BusinessDate `json:"harvestTo"`
	CommittedTons Dec          `json:"committedTons"`
	Note          string       `json:"note,omitempty"`
	AuditFields
}

// Validate checks a supply commitment.
func (e CaneSupplyEntry) Validate() error {
	v := &ValidationError{}
	if e.VersionID == "" {
		v.Add("versionId", "REQUIRED", "a supply commitment belongs to a plan version")
	}
	if e.SourceID == "" {
		v.Add("sourceId", "REQUIRED", "a supply commitment needs a cane source")
	}
	if e.CommittedTons.IsNegative() {
		v.Add("committedTons", "NEGATIVE", "a commitment cannot be negative")
	}
	if e.HarvestFrom == "" || e.HarvestTo == "" {
		v.Add("harvestFrom", "REQUIRED", "a commitment needs a harvest window")
	} else if e.HarvestTo < e.HarvestFrom {
		v.Add("harvestTo", "INVALID", "the harvest window ends before it starts")
	}
	return v.OrNil()
}

// HarvestDays is how many days the window is open, inclusive.
func (e CaneSupplyEntry) HarvestDays() int {
	if e.HarvestFrom == "" || e.HarvestTo == "" {
		return 0
	}
	n := e.HarvestFrom.DaysBetween(e.HarvestTo)
	if n < 0 {
		return 0
	}
	return n + 1
}

// RequiredDailyRate is the tonnage this source must move on every day of its
// window to meet its commitment.
//
// C46. It is the figure to compare against transport capacity: a commitment
// that needs more per day than the trucks can carry is one nobody can keep.
func (e CaneSupplyEntry) RequiredDailyRate() Dec {
	days := e.HarvestDays()
	if days == 0 {
		return Zero
	}
	return RoundRate(e.CommittedTons.Div(DI(int64(days))))
}

// DailyCaneSupply is one source's delivery on one day: the schedule when the
// series is PLAN, and what came through the gate when it is ACTUAL.
type DailyCaneSupply struct {
	ID           string       `json:"id"`
	VersionID    string       `json:"versionId"`
	SourceID     string       `json:"sourceId"`
	FactoryID    string       `json:"factoryId"`
	BusinessDate BusinessDate `json:"businessDate"`
	Series       Series       `json:"series"`
	// Tons is scheduled tonnage on a PLAN row and delivered tonnage on an
	// ACTUAL one.
	Tons Dec `json:"tons"`
	// Trips is the haulage that implies, rounded up: a half-full truck still
	// makes the trip, so the queue at the gate is counted in whole vehicles.
	Trips int `json:"trips"`
	// PolPct is the expected polarisation on a plan row and the measured one
	// on an actual, so the two can be compared per source rather than only in
	// the laboratory's own records.
	PolPct Dec    `json:"polPct"`
	Note   string `json:"note,omitempty"`
	AuditFields
}

// Validate checks one delivery row.
func (d DailyCaneSupply) Validate() error {
	v := &ValidationError{}
	if d.VersionID == "" {
		v.Add("versionId", "REQUIRED", "a delivery belongs to a plan version")
	}
	if d.SourceID == "" {
		v.Add("sourceId", "REQUIRED", "a delivery needs a cane source")
	}
	if _, err := d.BusinessDate.Time(); err != nil {
		v.Add("businessDate", "INVALID", "the business date is not an ISO date")
	}
	if d.Series != SeriesPlan && d.Series != SeriesActual {
		v.Add("series", "INVALID", "the series is PLAN or ACTUAL")
	}
	if d.Tons.IsNegative() {
		v.Add("tons", "NEGATIVE", "a delivery cannot be negative")
	}
	if d.Trips < 0 {
		v.Add("trips", "NEGATIVE", "the trip count cannot be negative")
	}
	return v.OrNil()
}

// TripsFor is how many vehicle movements a tonnage needs, rounded up.
//
// C47. Ceiling, not rounding: the last truck of the day travels whether it is
// full or not, and a queue planned on fractional vehicles is a queue that is
// always one vehicle short.
func TripsFor(tons, truckCapacity Dec) int {
	if truckCapacity.LessThanOrEqual(Zero) || tons.LessThanOrEqual(Zero) {
		return 0
	}
	return int(tons.Div(truckCapacity).Ceil().IntPart())
}

// SupplyReconciliation compares what the sources are committed to against what
// the season plan intends to crush.
//
// C48. This is the reason the module exists. A cane target with no sources
// behind it is a number somebody typed; a target that exceeds the commitments
// is a mill that will run out of cane, and one below them is cane that will
// stand in the field or go to a competitor. Either way the planner should be
// told before the season starts rather than during it.
type SupplyReconciliation struct {
	TargetTons    Dec `json:"targetTons"`
	CommittedTons Dec `json:"committedTons"`
	// Difference is committed minus target: positive is surplus, negative is
	// a shortfall.
	Difference  Dec `json:"differenceTons"`
	CoveragePct Dec `json:"coveragePct"`
	Sources     int `json:"sources"`
	// ExpectedTons is what the land should actually yield, which can differ
	// from what has been committed - a contract is a promise and a hectare is
	// a fact.
	ExpectedTons Dec `json:"expectedTons"`
}

// ReconcileSupply works out whether the committed cane covers the target.
func ReconcileSupply(target Dec, entries []CaneSupplyEntry, sources map[string]CaneSource) SupplyReconciliation {
	r := SupplyReconciliation{TargetTons: target, Sources: len(entries)}
	for _, e := range entries {
		r.CommittedTons = r.CommittedTons.Add(e.CommittedTons)
		if s, ok := sources[e.SourceID]; ok {
			r.ExpectedTons = r.ExpectedTons.Add(s.ExpectedTons())
		}
	}
	r.CommittedTons = RoundQty(r.CommittedTons)
	r.ExpectedTons = RoundQty(r.ExpectedTons)
	r.Difference = RoundQty(r.CommittedTons.Sub(target))
	r.CoveragePct = RoundPct(SafePct(r.CommittedTons, target))
	return r
}

// SupplyWarnings are the things a planner should be told about a supply plan
// before the season starts.
//
// Each one is a fact about the plan rather than an opinion about it: the
// tonnage does not add up, a source is committed to more than its land grows,
// or to more than its trucks can carry.
func SupplyWarnings(rec SupplyReconciliation, entries []CaneSupplyEntry,
	sources map[string]CaneSource, tolerancePct Dec) []Alert {

	var out []Alert
	if tolerancePct.LessThanOrEqual(Zero) {
		tolerancePct = D("2")
	}
	gap := rec.Difference.Abs()
	allowed := RoundQty(rec.TargetTons.Mul(tolerancePct).Div(DI(100)))
	if rec.TargetTons.GreaterThan(Zero) && gap.GreaterThan(allowed) {
		sev, word := SeverityError, "short of"
		if rec.Difference.GreaterThan(Zero) {
			sev, word = SeverityWarning, "more than"
		}
		out = append(out, Alert{
			Code: "SUPPLY_COVERAGE", Severity: sev,
			Title: "The committed cane does not match the season target",
			Detail: fmt.Sprintf(
				"%s sources are committed to %s t, which is %s t %s the %s t target (%s %% coverage)",
				fmt.Sprint(rec.Sources), rec.CommittedTons, gap, word, rec.TargetTons, rec.CoveragePct),
			Entity: "supply",
		})
	}

	for _, e := range entries {
		s, ok := sources[e.SourceID]
		if !ok {
			continue
		}
		if expected := s.ExpectedTons(); expected.GreaterThan(Zero) &&
			e.CommittedTons.GreaterThan(expected) {
			out = append(out, Alert{
				Code: "SUPPLY_OVER_YIELD", Severity: SeverityWarning,
				Title: "A source is committed to more than its land should yield",
				Detail: fmt.Sprintf("%s is committed to %s t from %s ha at %s t/ha, which is %s t",
					s.Code, e.CommittedTons, s.Hectares, s.ExpectedYieldTPH, expected),
				Entity: "caneSource", EntityID: s.ID,
			})
		}
		capacity := s.DailyTransportCapacity()
		if rate := e.RequiredDailyRate(); capacity.GreaterThan(Zero) && rate.GreaterThan(capacity) {
			out = append(out, Alert{
				Code: "SUPPLY_OVER_HAULAGE", Severity: SeverityWarning,
				Title: "A source needs more haulage than it has",
				Detail: fmt.Sprintf(
					"%s must move %s t a day across its window; %d trucks at %s t carry %s t",
					s.Code, rate, s.TrucksPerDay, s.TruckCapacityTons, capacity),
				Entity: "caneSource", EntityID: s.ID,
			})
		}
	}
	return out
}
