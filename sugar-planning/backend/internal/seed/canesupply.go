package seed

import (
	"context"
	"fmt"
	"sort"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
)

// A year of cane supply: where the 2,300,000 t comes from.
//
// The reference figures give a season target and nothing behind it. This is the
// other half: eight sources across four zones, each with an area, a yield, a
// harvest window and the trucks to move it, adding up to the target and then
// spread across the season into a delivery schedule.
//
// The tonnages are deliberately made to sum to slightly more than the target -
// 2,320,000 t against 2,300,000 - because that is what a real season looks
// like. A mill contracts a margin above what it intends to crush, since cane
// standing in a field is not cane at the gate. The supply plan reports the
// surplus rather than hiding it, and the 2 % tolerance is what decides whether
// it is worth saying out loud.

// SupplyResult reports what the cane supply seed created.
type SupplyResult struct {
	Sources       int
	Entries       int
	ScheduleRows  int
	CommittedTons domain.Dec
	DeliveryRows  int
}

// caneSource is one row of the supply master data, before it has an id.
type caneSource struct {
	Code, Name string
	Type       domain.SourceType
	Zone       string
	Distance   string
	Hectares   string
	Variety    string
	Yield      string // tons per hectare
	Pol        string
	TruckTons  string
	Trucks     int
	// Committed is the season commitment, and Cut is the harvest window as an
	// offset in days from the season start and a length in days.
	Committed  string
	CutFromDay int
	CutDays    int
}

// The estate blocks are cut first and last: they are the mill's own land, so
// they are what fills the shoulders of the season when the contracted growers
// are not ready. The outgrowers sit in the middle, when the crop is at its
// best and the gate is busiest.
var caneSources = []caneSource{
	{Code: "CS-E01", Name: "Chbar Mon estate, block A", Type: domain.SourceEstate,
		Zone: "Chbar Mon", Distance: "6.5", Hectares: "4200", Variety: "K88-92",
		Yield: "68", Pol: "12.8", TruckTons: "18", Trucks: 360,
		Committed: "285600", CutFromDay: 0, CutDays: 46},
	{Code: "CS-E02", Name: "Chbar Mon estate, block B", Type: domain.SourceEstate,
		Zone: "Chbar Mon", Distance: "8.2", Hectares: "3800", Variety: "K88-92",
		Yield: "71", Pol: "12.9", TruckTons: "18", Trucks: 350,
		Committed: "269800", CutFromDay: 92, CutDays: 45},
	{Code: "CS-E03", Name: "Thpong estate", Type: domain.SourceEstate,
		Zone: "Thpong", Distance: "23.0", Hectares: "5100", Variety: "KPS01-25",
		Yield: "74", Pol: "13.1", TruckTons: "20", Trucks: 330,
		Committed: "377400", CutFromDay: 14, CutDays: 60},
	{Code: "CS-C01", Name: "Oral growers co-operative", Type: domain.SourceContract,
		Zone: "Oral", Distance: "41.5", Hectares: "4600", Variety: "K93-219",
		Yield: "66", Pol: "12.4", TruckTons: "20", Trucks: 230,
		Committed: "303600", CutFromDay: 21, CutDays: 70},
	{Code: "CS-C02", Name: "Phnom Sruoch contract farms", Type: domain.SourceContract,
		Zone: "Phnom Sruoch", Distance: "29.0", Hectares: "5400", Variety: "K93-219",
		Yield: "69", Pol: "12.6", TruckTons: "20", Trucks: 260,
		Committed: "372600", CutFromDay: 30, CutDays: 75},
	{Code: "CS-C03", Name: "Samraong Tong contract farms", Type: domain.SourceContract,
		Zone: "Samraong Tong", Distance: "17.5", Hectares: "4900", Variety: "KPS01-25",
		Yield: "72", Pol: "13.0", TruckTons: "18", Trucks: 300,
		Committed: "352800", CutFromDay: 38, CutDays: 68},
	{Code: "CS-O01", Name: "Oral smallholders", Type: domain.SourceOutgrower,
		Zone: "Oral", Distance: "44.0", Hectares: "2600", Variety: "mixed",
		Yield: "58", Pol: "11.9", TruckTons: "12", Trucks: 190,
		Committed: "150800", CutFromDay: 45, CutDays: 62},
	{Code: "CS-O02", Name: "Kong Pisei smallholders", Type: domain.SourceOutgrower,
		Zone: "Kong Pisei", Distance: "33.5", Hectares: "3400", Variety: "mixed",
		Yield: "61", Pol: "12.1", TruckTons: "12", Trucks: 280,
		Committed: "207400", CutFromDay: 52, CutDays: 66},
}

// LoadCaneSupply adds the cane sources, commits them to the season and builds
// the delivery schedule.
//
// Like the rest of the seed it goes through the services rather than the store,
// so the commitments are validated, audited and reconciled exactly as a
// planner's would be - and the warnings it raises are the system's own.
func LoadCaneSupply(ctx context.Context, s store.Store, planning *service.Planning,
	factoryID, versionID string, seasonStart domain.BusinessDate) (SupplyResult, error) {

	var out SupplyResult
	md := s.MasterData()

	byCode := map[string]string{}
	for _, c := range caneSources {
		saved, err := upsert(ctx, md.CaneSources(), c.Code, domain.CaneSource{
			FactoryID: factoryID, Code: c.Code, Name: c.Name, Type: c.Type, Zone: c.Zone,
			DistanceKm: domain.D(c.Distance), Hectares: domain.D(c.Hectares),
			Variety: c.Variety, ExpectedYieldTPH: domain.D(c.Yield),
			ExpectedPolPct: domain.D(c.Pol), TruckCapacityTons: domain.D(c.TruckTons),
			TrucksPerDay: c.Trucks, Validity: active(),
		})
		if err != nil {
			return out, fmt.Errorf("cane source %s: %w", c.Code, err)
		}
		byCode[c.Code] = saved.ID
		out.Sources++
	}

	for _, c := range caneSources {
		entry := domain.CaneSupplyEntry{
			VersionID:     versionID,
			SourceID:      byCode[c.Code],
			HarvestFrom:   seasonStart.AddDays(c.CutFromDay),
			HarvestTo:     seasonStart.AddDays(c.CutFromDay + c.CutDays - 1),
			CommittedTons: domain.D(c.Committed),
			Note:          c.Zone + " zone",
		}
		if _, err := planning.SaveSupplyEntry(ctx, entry); err != nil {
			return out, fmt.Errorf("cane supply commitment %s: %w", c.Code, err)
		}
		out.Entries++
		out.CommittedTons = out.CommittedTons.Add(domain.D(c.Committed))
	}

	// The schedule: a row per source per day of its window. This is the year of
	// data - roughly 490 delivery rows across the season.
	schedule, err := planning.GenerateSupplySchedule(ctx, versionID)
	if err != nil {
		return out, fmt.Errorf("generate the delivery schedule: %w", err)
	}
	out.ScheduleRows = schedule.Rows
	return out, nil
}

// LoadCaneDeliveries records what actually came through the gate for the days
// the demonstration has actuals for.
//
// The deliveries are deliberately not equal to the schedule: a source runs
// short, another over-delivers, and the totals differ from the plan by a few
// per cent. A demonstration where the gate matches the schedule every day
// teaches nobody what the screen is for.
func LoadCaneDeliveries(ctx context.Context, s store.Store, planning *service.Planning,
	planVersionID, actualVersionID string, days int) (int, error) {

	scheduled, err := s.Planning().ListCaneSupply(ctx, store.PlanFilter{
		VersionIDs: []string{planVersionID}, Series: domain.SeriesPlan, Top: 100000,
	})
	if err != nil {
		return 0, err
	}
	if len(scheduled) == 0 {
		return 0, nil
	}

	// The earliest `days` distinct dates: the same fortnight the rest of the
	// demonstration records. Sorted here rather than trusted from the store, so
	// this does not quietly depend on a repository's ordering.
	seen := map[domain.BusinessDate]bool{}
	var dates []domain.BusinessDate
	for _, r := range scheduled {
		if !seen[r.BusinessDate] {
			seen[r.BusinessDate] = true
			dates = append(dates, r.BusinessDate)
		}
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i] < dates[j] })
	if len(dates) > days {
		dates = dates[:days]
	}
	within := map[domain.BusinessDate]bool{}
	for _, d := range dates {
		within[d] = true
	}

	sources, err := s.MasterData().CaneSources().List(ctx, store.ListOptions{Top: 200})
	if err != nil {
		return 0, err
	}
	truck := map[string]domain.Dec{}
	pol := map[string]domain.Dec{}
	for _, c := range sources.Items {
		truck[c.ID], pol[c.ID] = c.TruckCapacityTons, c.ExpectedPolPct
	}

	var rows []domain.DailyCaneSupply
	for i, r := range scheduled {
		if !within[r.BusinessDate] {
			continue
		}
		// A deterministic swing of about ±8 %, so the same fortnight every run
		// and no two days alike.
		swing := domain.D(fmt.Sprintf("%d", (i*37)%17-8)).Div(domain.DI(100))
		tons := domain.RoundQty(r.Tons.Mul(domain.DI(1).Add(swing)))
		polActual := domain.RoundPct(pol[r.SourceID].Add(
			domain.D(fmt.Sprintf("%d", (i*23)%9-4)).Div(domain.DI(10))))
		rows = append(rows, domain.DailyCaneSupply{
			VersionID: actualVersionID, SourceID: r.SourceID, FactoryID: r.FactoryID,
			BusinessDate: r.BusinessDate, Series: domain.SeriesActual,
			Tons:   tons,
			Trips:  domain.TripsFor(tons, truck[r.SourceID]),
			PolPct: polActual,
		})
	}
	if len(rows) == 0 {
		return 0, nil
	}
	res, err := planning.UpsertCaneSupply(ctx, actualVersionID, rows)
	if err != nil {
		return 0, fmt.Errorf("record the deliveries: %w", err)
	}
	return res.Accepted, nil
}
