package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
)

// The demonstration scenario: a factory a fortnight into its campaign.
//
// Load and LoadWithActuals build the plan and the daily figures behind it, and
// that is enough to show the planning half of the system. It leaves every
// execution screen empty - no orders, no stoppages, no laboratory results, no
// stock movements - which is a poor demonstration of a system whose point is
// that the plan and the floor are the same set of books.
//
// LoadDemo fills them in. Two rules shape how.
//
// Everything goes through the **services**, never the store. A demonstration
// assembled by writing rows would be a demonstration of nothing: it would skip
// the validation, the audit trail, the outbox and the permission checks, and the
// first figure somebody questioned would turn out not to reconcile. Driving the
// services means every row here is a row the system produced, by the same code
// path a person uses.
//
// And it is deterministic. The same dates, the same quantities, the same
// failures every time it is loaded, so a demonstration can be scripted and a
// screenshot stays true.

// DemoResult reports what the scenario put on each screen, so the loader can
// say so in one line rather than the operator having to go and look.
type DemoResult struct {
	Result
	Downtime      int
	Orders        int
	Confirmations int
	Documents     int
	Samples       int
	Holds         int
	CostRuns      int
	Alerts        int
	Views         int
	// AlreadyPlayed says the scenario found execution data and left it alone.
	// Without it a restart logs a row of zeroes, which reads as a failure
	// rather than as the idempotency working.
	AlreadyPlayed bool
}

// LoadDemo builds the reference scenario and then plays a fortnight of factory
// life through it.
func LoadDemo(ctx context.Context, s store.Store, planning *service.Planning,
	analytics *service.Analytics, days int,
) (DemoResult, error) {

	base, err := LoadWithActuals(ctx, s, planning, days)
	out := DemoResult{Result: base}
	if err != nil {
		return out, err
	}

	ctx = auth.WithPrincipal(ctx, demoPrincipal(base.CompanyID, base.FactoryID))
	now := func() time.Time { return time.Now().UTC() }
	execution := service.NewExecution(s, now)

	// The days the actuals cover. Everything below happens inside them, so the
	// dashboard's as-of date and the execution screens describe the same
	// fortnight rather than two different ones.
	cane, err := s.Planning().ListCane(ctx, store.PlanFilter{
		VersionIDs: []string{base.ActualID}, Series: domain.SeriesActual, Top: 400,
	})
	if err != nil {
		return out, fmt.Errorf("read the recorded days: %w", err)
	}
	if len(cane) == 0 {
		// Nothing was recorded, so there is no fortnight to populate. That is
		// not an error: a demonstration of a season that has not started is a
		// legitimate thing to want.
		return out, nil
	}
	dates := make([]domain.BusinessDate, 0, len(cane))
	for _, row := range cane {
		dates = append(dates, row.BusinessDate)
	}

	// Load and LoadWithActuals are idempotent on business keys, and this has to
	// be too. A container that restarts would otherwise double the stoppages
	// and the orders every time, and a demonstration that grows on its own is
	// one nobody can quote a figure from.
	//
	// There is no business key to match on for a stoppage - two boiler failures
	// on the same day at the same hour are a thing that can genuinely happen -
	// so the test is whether this factory has any execution data at all. Either
	// the fortnight has been played through or it has not.
	played, err := alreadyPlayed(ctx, s, base.FactoryID)
	if err != nil {
		return out, err
	}
	if played {
		out.AlreadyPlayed = true
		return out, nil
	}

	lines, err := lineIndex(ctx, s, base.FactoryID)
	if err != nil {
		return out, err
	}

	if out.Downtime, err = demoDowntime(ctx, s, base, lines, dates); err != nil {
		return out, err
	}
	if err := demoProduction(ctx, execution, base, lines, dates, &out); err != nil {
		return out, err
	}
	if err := demoStockMovements(ctx, execution, base, dates, &out); err != nil {
		return out, err
	}
	if err := demoQuality(ctx, s, execution, base, dates, &out); err != nil {
		return out, err
	}
	if out.CostRuns, err = demoCosting(ctx, s, planning, base, dates, now); err != nil {
		return out, err
	}
	if out.Alerts, err = demoAlerts(ctx, s, analytics, now); err != nil {
		return out, err
	}
	if out.Views, err = demoViews(ctx, s, base, dates); err != nil {
		return out, err
	}

	// The document count is read back rather than added up along the way. The
	// hand-kept tally said eight and the ledger held ten: placing a quality
	// hold and releasing it are each their own document, which is right and
	// which the counter did not know. A demonstration that reports a figure it
	// computed by hand rather than by looking is the thing this system exists
	// to replace.
	if out.Documents, err = countDocuments(ctx, s, base.FactoryID); err != nil {
		return out, err
	}
	return out, nil
}

// countDocuments reads back how many inventory documents the scenario produced.
func countDocuments(ctx context.Context, s store.Store, factoryID string) (int, error) {
	page, err := s.Execution().ListDocuments(ctx, store.ExecutionFilter{
		FactoryID: factoryID, Top: 1,
	})
	if err != nil {
		return 0, fmt.Errorf("count the inventory documents: %w", err)
	}
	return page.Count, nil
}

// demoPrincipal is the seed principal plus the laboratory, because the
// demonstration records quality results as well as production.
func demoPrincipal(companyID, factoryID string) auth.Principal {
	return auth.NewPrincipal("seed", Actor, "Seed data loader", "",
		[]string{
			auth.RoleSystemAdmin, auth.RoleMasterDataAdmin, auth.RoleProductionPlanner,
			auth.RoleApprover, auth.RoleShipmentPlanner,
			auth.RoleShiftSupervisor, auth.RoleWarehouseOperator,
			auth.RoleCostController, auth.RoleQualityUser,
		},
		[]string{companyID}, []string{factoryID})
}

// alreadyPlayed reports whether this factory has been through the scenario.
//
// Production orders are the marker rather than, say, downtime: an order is the
// thing the scenario creates first that a real user would never have created
// before the demonstration ran, and one order is enough to know.
func alreadyPlayed(ctx context.Context, s store.Store, factoryID string) (bool, error) {
	page, err := s.Execution().ListOrders(ctx, store.ExecutionFilter{
		FactoryID: factoryID, Top: 1,
	})
	if err != nil {
		return false, fmt.Errorf("check for existing execution data: %w", err)
	}
	return page.Count > 0, nil
}

func lineIndex(ctx context.Context, s store.Store, factoryID string) (map[string]string, error) {
	page, err := s.MasterData().Lines().List(ctx, store.ListOptions{
		Top: 100, ParentID: factoryID,
	})
	if err != nil {
		return nil, fmt.Errorf("read the production lines: %w", err)
	}
	out := map[string]string{}
	for _, l := range page.Items {
		out[l.Code] = l.ID
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Downtime
// ---------------------------------------------------------------------------

// demoDowntime records a fortnight of stoppages.
//
// The spread matters more than the total. A Pareto of four reasons that each
// cost the same is a chart nobody learns anything from; this one is dominated by
// the boiler, which is what makes "go and fix the boiler" the obvious reading.
func demoDowntime(ctx context.Context, s store.Store, base Result,
	lines map[string]string, dates []domain.BusinessDate,
) (int, error) {

	type stoppage struct {
		day            int
		line           string
		reason         string
		startHr, endHr int
		planned        bool
		cause, action  string
	}
	// Day 3 is the bad one - it is why the seeded actuals dip to 61 % of target
	// that day, so the two halves of the demonstration agree with each other.
	script := []stoppage{
		{2, "MILL-1", "DT-BOILER", 6, 14, false,
			"Boiler tube leak on the second pass", "Tube plugged; full replacement at the next shutdown"},
		{4, "MILL-1", "DT-RAIN", 5, 9, false,
			"Heavy overnight rain, no cane arriving", "Waited for the fields to drain"},
		{6, "REF-1", "DT-POWER", 2, 4, false,
			"Grid dip tripped the refinery drives", "Drives restarted; ride-through setting reviewed"},
		{7, "MILL-1", "DT-BOILER", 21, 2, false,
			"Boiler pressure falling again", "Feedwater pump strainer cleaned"},
		{9, "PACK-1", "DT-MILL", 8, 10, true,
			"Planned changeover from 50 kg to 1 kg", "Ran to schedule"},
		{11, "MILL-1", "DT-BOILER", 4, 9, false,
			"Same tube pass leaking", "Escalated: boiler on the maintenance plan"},
		{12, "MILL-1", "DT-RAIN", 3, 6, false,
			"Rain, partial cane supply", "Crushed what was on the yard"},
	}

	count := 0
	for _, e := range script {
		if e.day >= len(dates) {
			continue
		}
		date := dates[e.day]
		day, err := time.Parse("2006-01-02", string(date))
		if err != nil {
			return count, fmt.Errorf("parse %s: %w", date, err)
		}
		start := day.Add(time.Duration(e.startHr) * time.Hour)
		end := day.Add(time.Duration(e.endHr) * time.Hour)
		if !end.After(start) {
			// A night-shift stoppage runs past midnight. The business date is
			// still the day the shift began, which is how a supervisor enters it.
			end = end.Add(24 * time.Hour)
		}
		minutes := int64(end.Sub(start) / time.Minute)

		if _, err := s.Planning().SaveDowntime(ctx, domain.DowntimeEvent{
			FactoryID: base.FactoryID, LineID: lines[e.line], BusinessDate: date,
			StartAt: start, EndAt: end,
			DurationHrs: domain.RoundRate(domain.DI(minutes).Div(domain.DI(60))),
			Planned:     e.planned, ReasonCode: e.reason,
			RootCause: e.cause, Action: e.action, Team: "Shift B",
		}, Actor); err != nil {
			return count, fmt.Errorf("record the %s stoppage on %s: %w", e.reason, date, err)
		}
		count++
	}
	return count, nil
}

// ---------------------------------------------------------------------------
// Production orders
// ---------------------------------------------------------------------------

// demoProduction releases and confirms a run of packing orders.
//
// One of them finishes short and is closed with a variance reason, because that
// is the case worth demonstrating: the system refuses a silent close, and the
// reason it demands ends up on the variance report.
func demoProduction(ctx context.Context, execution *service.Execution, base Result,
	lines map[string]string, dates []domain.BusinessDate, out *DemoResult,
) error {

	type run struct {
		day       int
		product   string
		packaging string
		planned   string
		yield     string
		batch     string
		reason    string // set when the order closes short
	}
	script := []run{
		{5, "REF", "P50KG", "300", "300", "REF-2612-01", ""},
		{6, "WHT", "P50KG", "420", "420", "WHT-2612-01", ""},
		{8, "REF", "P50KG", "300", "240", "REF-2612-02", "VAR-CANE"},
		{9, "SUP", "P1KG", "12", "12", "SUP-2612-01", ""},
		{11, "WHT", "P50KG", "420", "431", "WHT-2612-02", ""},
	}

	for _, r := range script {
		if r.day >= len(dates) {
			continue
		}
		date := dates[r.day]
		productID, ok := base.Products[r.product]
		if !ok {
			continue
		}

		order, err := execution.CreateOrder(ctx, service.OrderRequest{
			FactoryID: base.FactoryID, LineID: lines["PACK-1"], VersionID: base.BudgetID,
			BusinessDate: date, ProductID: productID,
			PackagingID: base.Packaging[r.packaging],
			PlannedQty:  domain.D(r.planned), Team: "Shift A",
		})
		if err != nil {
			return fmt.Errorf("create the %s order on %s: %w", r.product, date, err)
		}
		out.Orders++

		if _, err := execution.ActOnOrder(ctx, order.ID, service.OrderActionRequest{
			Action: domain.OrderActionRelease,
		}); err != nil {
			return fmt.Errorf("release order %s: %w", order.OrderNo, err)
		}

		if _, err := execution.Confirm(ctx, order.ID, service.ConfirmRequest{
			BusinessDate: date, YieldQty: domain.D(r.yield),
			BatchCode:   r.batch,
			WarehouseID: base.Warehouses["FG-WH1"],
			LabourHours: domain.D("16"), MachineHours: domain.D("7.5"),
		}); err != nil {
			return fmt.Errorf("confirm order %s: %w", order.OrderNo, err)
		}
		out.Confirmations++

		// Closing an order that missed its plan is an explained act: the system
		// refuses it without a reason, and the reason is what the variance
		// report reads back.
		if r.reason != "" {
			if _, err := execution.ActOnOrder(ctx, order.ID, service.OrderActionRequest{
				Action: domain.OrderActionClose, VarianceReason: r.reason,
				Reason: "Cane supply interrupted by rain; the shift was ended early",
			}); err != nil {
				return fmt.Errorf("close order %s short: %w", order.OrderNo, err)
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Stock movements
// ---------------------------------------------------------------------------

// demoStockMovements posts the movements a warehouse actually makes.
//
// A transfer, a stock-count correction, and a reversal - the last because a
// system that cannot show a mistake being undone has not demonstrated the thing
// that matters most about an inventory ledger.
func demoStockMovements(ctx context.Context, execution *service.Execution, base Result,
	dates []domain.BusinessDate, out *DemoResult,
) error {

	refined, ok := base.Products["REF"]
	if !ok {
		return nil
	}
	from, to := base.Warehouses["FG-WH1"], base.Warehouses["FG-WH3"]
	if from == "" || to == "" {
		return nil
	}
	last := dates[len(dates)-1]

	// Warehouse 1 fills first, so some of it moves to Warehouse 3. This is the
	// movement the capacity alert on the dashboard is about.
	if _, err := execution.Post(ctx, service.PostingRequest{
		DocType: domain.DocTransfer, BusinessDate: dates[len(dates)-3],
		FactoryID: base.FactoryID, Reference: "Relieving FG-WH1",
		Note: "Warehouse 1 is forecast to fill; moving refined sugar to Warehouse 3",
		Lines: []service.PostingLineInput{{
			WarehouseID: from, ToWarehouse: to, ProductID: refined,
			BatchCode: "REF-2612-01", Quantity: domain.D("120"),
		}},
	}); err != nil {
		return fmt.Errorf("post the transfer: %w", err)
	}

	// A stock count found less than the book said. The correction is a document
	// with a reason on it, not an edit.
	adjustment, err := execution.Post(ctx, service.PostingRequest{
		DocType: domain.DocAdjustment, BusinessDate: dates[len(dates)-2],
		FactoryID: base.FactoryID, Reference: "Stock count W50",
		ReasonCode: "ADJ-COUNT", Note: "Counted 4 t less than the book in FG-WH1",
		Lines: []service.PostingLineInput{{
			WarehouseID: from, ProductID: refined, BatchCode: "REF-2612-01",
			Quantity: domain.D("-4"),
		}},
	})
	if err != nil {
		return fmt.Errorf("post the adjustment: %w", err)
	}

	// And the count was wrong: the pallets were behind a stack. Reversing is
	// how a posted document is undone - the original stays, flagged, and the
	// reversal is its own document.
	if _, err := execution.Reverse(ctx, adjustment.ID, service.ReversalRequest{
		BusinessDate: last,
		Reason:       "Recount found the pallets behind a stack; the first count was wrong",
	}); err != nil {
		return fmt.Errorf("reverse the adjustment: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Quality
// ---------------------------------------------------------------------------

// demoQuality records the laboratory's fortnight.
//
// Three samples: one that passes, one that fails and blocks the material it was
// taken from, and one whose hold is then released after a rework. The failure is
// the point - a quality module that only ever passes has demonstrated nothing.
func demoQuality(ctx context.Context, s store.Store, execution *service.Execution,
	base Result, dates []domain.BusinessDate, out *DemoResult,
) error {

	params, err := s.Execution().ListParameters(ctx)
	if err != nil {
		return fmt.Errorf("read the quality parameters: %w", err)
	}
	byCode := map[string]string{}
	for _, p := range params {
		byCode[p.Code] = p.ID
	}
	if byCode["POL"] == "" {
		return nil
	}

	type sheet struct {
		day     int
		product string
		batch   string
		pol     string
		colour  string
		comment string
		hold    string // quantity to block, empty for none
		release string
	}
	// The refined specification is polarisation at 99.700 degZ or better, so
	// 99.640 fails: a figure just outside, which is what a real rejection looks
	// like rather than a number nobody would believe.
	script := []sheet{
		{5, "REF", "REF-2612-01", "99.820", "32", "Routine shift sample", "", ""},
		{8, "REF", "REF-2612-02", "99.640", "48", "Colour and polarisation both drifting", "240", "Reprocessed through the refinery; retested within specification"},
		{11, "WHT", "WHT-2612-02", "99.610", "118", "Routine shift sample", "", ""},
	}

	for _, sh := range script {
		if sh.day >= len(dates) {
			continue
		}
		date := dates[sh.day]
		productID, ok := base.Products[sh.product]
		if !ok {
			continue
		}

		sample, err := execution.CreateSample(ctx, service.SampleRequest{
			ProductID: productID, BatchCode: sh.batch, FactoryID: base.FactoryID,
			BusinessDate: date, Comment: sh.comment,
		})
		if err != nil {
			return fmt.Errorf("take the %s sample on %s: %w", sh.product, date, err)
		}
		out.Samples++

		req := service.ResultsRequest{
			Results: []service.ResultInput{
				{ParameterID: byCode["POL"], Value: domain.D(sh.pol)},
				{ParameterID: byCode["COLOUR"], Value: domain.D(sh.colour)},
			},
			Complete: true,
		}
		if sh.hold != "" {
			req.HoldWarehouse = base.Warehouses["FG-WH1"]
			req.HoldQuantity = domain.D(sh.hold)
		}
		outcome, err := execution.RecordResults(ctx, sample.ID, req)
		if err != nil {
			return fmt.Errorf("record the results for %s: %w", sample.SampleNo, err)
		}
		if outcome.Hold != nil {
			out.Holds++
			if sh.release != "" {
				if _, err := execution.ReleaseHold(ctx, outcome.Hold.ID, service.ReleaseRequest{
					ReleasedOn: dates[len(dates)-1], Reason: sh.release,
					RowVersion: outcome.Hold.RowVersion,
				}); err != nil {
					return fmt.Errorf("release the hold on %s: %w", sh.batch, err)
				}
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Costing
// ---------------------------------------------------------------------------

// demoCosting saves a cost run over the recorded fortnight, so the costing
// screen opens on a figure rather than on an empty form.
func demoCosting(ctx context.Context, s store.Store, planning *service.Planning,
	base Result, dates []domain.BusinessDate, now func() time.Time,
) (int, error) {

	costing := service.NewCosting(s, planning, now)
	if _, err := costing.Run(ctx, service.CostRunRequest{
		SeasonID: base.SeasonID, VersionID: base.BudgetID,
		From: dates[0], To: dates[len(dates)-1],
		Save: true, Code: "RUN-W49-W50",
		Note: "First fortnight of the campaign, standard rates against recorded output",
	}); err != nil {
		return 0, fmt.Errorf("save the cost run: %w", err)
	}
	return 1, nil
}

// ---------------------------------------------------------------------------
// Alerts and saved views
// ---------------------------------------------------------------------------

// demoAlerts runs the evaluation once, so the inboxes are not empty on a plan
// that has two real capacity problems in it.
func demoAlerts(ctx context.Context, s store.Store, analytics *service.Analytics,
	now func() time.Time,
) (int, error) {

	if analytics == nil {
		return 0, nil
	}
	// The job runs as the scheduler does: a system principal, whose data scope
	// is every factory and whose permissions are no wider than a reader's.
	jobCtx := auth.WithPrincipal(ctx, auth.NewSystemPrincipal(
		auth.RoleProductionPlanner, auth.RoleShipmentPlanner, auth.RoleApprover))
	result, err := service.NewNotifications(s, analytics, now).Evaluate(jobCtx)
	if err != nil {
		return 0, fmt.Errorf("evaluate the alerts: %w", err)
	}
	return result.Raised, nil
}

// demoViews gives the demonstration accounts a saved view each, so the variant
// control opens with something in it rather than being an empty dropdown
// somebody has to be told about.
func demoViews(ctx context.Context, s store.Store, base Result,
	dates []domain.BusinessDate,
) (int, error) {

	last := dates[len(dates)-1]
	first := dates[0]

	views := []domain.SavedView{
		{
			Owner: "planner", Page: "board", Name: "The recorded fortnight",
			FactoryID: base.FactoryID, IsDefault: true,
			Payload: json.RawMessage(fmt.Sprintf(
				`{"kind":"cane","series":"ACTUAL","from":%q,"to":%q}`, first, last)),
		},
		{
			// Shared, because the morning review is a meeting rather than one
			// person's habit.
			Owner: "supervisor", Page: "downtime", Name: "Morning review",
			FactoryID: base.FactoryID, Shared: true,
			Payload: json.RawMessage(fmt.Sprintf(
				`{"from":%q,"to":%q,"lineId":"","sort":{"sortKey":"durationHours","descending":true,"groupKey":"reasonName"}}`,
				first, last)),
		},
		{
			Owner: "supervisor", Page: "orders", Name: "Everything still open",
			FactoryID: base.FactoryID,
			Payload: json.RawMessage(
				`{"openOnly":true,"sort":{"sortKey":"businessDate","descending":true,"groupKey":"status"}}`),
		},
	}

	count := 0
	for _, v := range views {
		if _, err := s.SavedViews().Save(ctx, v); err != nil {
			return count, fmt.Errorf("save the %q view: %w", v.Name, err)
		}
		count++
	}
	return count, nil
}
