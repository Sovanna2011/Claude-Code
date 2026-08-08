package api

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/report"
	"github.com/kss/sugarplan/internal/store"
)

// The reports that reconcile the process rather than describe the plan: where
// the sugar went, what the orders actually produced, what stopped the factory
// and what the laboratory found.
//
// They read through the execution service rather than the store, so the
// permission check and the company/factory scope are the same ones that guard
// the screens. A report is a second way to look at data, not a second set of
// rules about who may see it.

// ---------------------------------------------------------------------------
// Raw sugar recovery and mass balance
// ---------------------------------------------------------------------------

// reportRecovery reconciles cane crushed against raw sugar produced.
//
// It reports the actuals, not the plan, because that is what a mass balance is:
// two figures measured separately - one on the weighbridge, one at the end of
// the raw house - which never agree exactly. The plan has only one number, and
// reconciling it against itself would print a column of zeroes and call it
// assurance. What matters is whether the gap between the two measurements is
// inside the tolerance the season was signed off with, and that is what the
// last column says.
func (s *Server) reportRecovery(r *http.Request, t report.Table, season domain.Season,
	v domain.PlanVersion, from, to domain.BusinessDate) (report.Table, error) {

	ctx := r.Context()
	_, actual, err := s.analytics.Versions(ctx, season.ID, v.ID)
	if err != nil {
		return t, err
	}
	f := store.PlanFilter{
		VersionIDs: []string{actual.ID}, From: from, To: to, Series: domain.SeriesActual,
	}

	cane, err := s.store.Planning().ListCane(ctx, f)
	if err != nil {
		return t, err
	}
	products, err := s.store.Planning().ListProducts(ctx, f)
	if err != nil {
		return t, err
	}
	index, err := s.productIndex(ctx)
	if err != nil {
		return t, err
	}
	// The tolerances belong to the plan the season was approved with, not to the
	// actuals container, which carries no assumptions of its own.
	target, minPct, maxPct, tolerance := recoveryAssumptions(ctx, s, v.ID)

	t.Title = "Raw sugar recovery and mass balance"
	t.Columns = []report.Column{
		{Header: "Date", Width: 12},
		{Header: "Cane crushed (t)", Width: 16, Align: report.AlignRight, Numeric: true},
		{Header: "Expected raw (t)", Width: 16, Align: report.AlignRight, Numeric: true},
		{Header: "Raw produced (t)", Width: 16, Align: report.AlignRight, Numeric: true},
		{Header: "Recovery %", Width: 12, Align: report.AlignRight, Numeric: true},
		{Header: "Recovery variance", Width: 17, Align: report.AlignRight, Numeric: true},
		{Header: "Balance difference (t)", Width: 20, Align: report.AlignRight, Numeric: true},
		{Header: "Verdict", Width: 12},
	}

	type day struct{ crushed, raw domain.Dec }
	byDate := map[domain.BusinessDate]*day{}
	at := func(d domain.BusinessDate) *day {
		if x, ok := byDate[d]; ok {
			return x
		}
		x := &day{}
		byDate[d] = x
		return x
	}
	for _, c := range cane {
		x := at(c.BusinessDate)
		x.crushed = x.crushed.Add(c.CaneCrushed)
	}
	for _, p := range products {
		if index[p.ProductID].Stage != domain.StageRawSugar {
			continue
		}
		x := at(p.BusinessDate)
		x.raw = x.raw.Add(p.Quantity)
	}

	var totals day
	exceptions := 0
	for _, d := range sortedKeys(byDate) {
		x := byDate[d]
		expected := domain.ExpectedRawSugar(x.crushed, target)
		achieved := domain.ActualRecoveryPct(x.raw, x.crushed)
		diff := domain.MassBalanceDiff(expected, x.raw)

		verdict := domain.RecoveryVerdict(achieved, minPct, maxPct)
		if !domain.WithinTolerance(diff, expected, tolerance) {
			// The recovery can sit inside its operating range while the tonnages
			// still fail to reconcile, and that is an exception in its own
			// right. A day that read SUCCESS because the percentage looked fine
			// would hide it.
			exceptions++
			if verdict == domain.SeveritySuccess || verdict == domain.SeverityInfo {
				verdict = domain.SeverityWarning
			}
		}

		t.Rows = append(t.Rows, []report.Cell{
			report.Date(d),
			report.Num(x.crushed, 3), report.Num(expected, 3), report.Num(x.raw, 3),
			report.Num(achieved, 3), report.Num(achieved.Sub(target), 3),
			report.Num(diff, 3), report.Text(string(verdict)),
		})
		totals.crushed = totals.crushed.Add(x.crushed)
		totals.raw = totals.raw.Add(x.raw)
	}

	expectedTotal := domain.ExpectedRawSugar(totals.crushed, target)
	t.Totals = []report.Cell{
		report.Text("Total"),
		report.Num(totals.crushed, 3), report.Num(expectedTotal, 3), report.Num(totals.raw, 3),
		report.Num(domain.ActualRecoveryPct(totals.raw, totals.crushed), 3), report.Text(""),
		report.Num(domain.MassBalanceDiff(expectedTotal, totals.raw), 3), report.Text(""),
	}
	t.Notes = append(t.Notes,
		"The figures are actuals from "+actual.Code+". A mass balance compares two measurements; "+
			"the plan holds only one, so there is nothing in it to reconcile.",
		"Expected raw sugar = cane crushed x "+target.String()+" % ("+domain.AsmRecoveryPct+"), "+
			"the assumption of plan version "+v.Code+".",
		"Balance difference = expected raw sugar - raw sugar produced; the tolerance is "+
			tolerance.String()+" % of expected ("+domain.AsmMassBalanceTolPct+").",
		"Recovery is judged against the range "+minPct.String()+" % to "+maxPct.String()+" %.",
		plural(exceptions, "%d day is outside the mass balance tolerance.",
			"%d days are outside the mass balance tolerance."))
	if len(t.Rows) == 0 {
		t.Notes = append(t.Notes,
			"No cane has been recorded as crushed in this period, so there is nothing to reconcile yet.")
	}
	return t, nil
}

// recoveryAssumptions reads the four figures the reconciliation is judged
// against. A version that does not carry one falls back to zero, which shows
// as an unjudgeable column rather than a fabricated verdict.
func recoveryAssumptions(ctx context.Context, s *Server, versionID string) (target, minPct, maxPct, tolerance domain.Dec) {
	assumptions, err := s.store.Planning().ListAssumptions(ctx, versionID)
	if err != nil {
		return
	}
	for _, a := range assumptions {
		switch a.Code {
		case domain.AsmRecoveryPct:
			target = a.Value
		case domain.AsmRecoveryMinPct:
			minPct = a.Value
		case domain.AsmRecoveryMaxPct:
			maxPct = a.Value
		case domain.AsmMassBalanceTolPct:
			tolerance = a.Value
		}
	}
	return
}

// ---------------------------------------------------------------------------
// Remelt and refining
// ---------------------------------------------------------------------------

// reportRemelt shows what the refinery consumed and what came out of it.
//
// The yield column is the one people read: raw sugar goes in, refined and white
// sugar come out, and the difference is process loss that has to be accounted
// for somewhere.
func (s *Server) reportRemelt(r *http.Request, t report.Table, season domain.Season,
	v domain.PlanVersion, from, to domain.BusinessDate) (report.Table, error) {

	ctx := r.Context()
	versions, err := s.planAndActual(ctx, season, v)
	if err != nil {
		return t, err
	}
	rows, err := s.store.Planning().ListProducts(ctx, store.PlanFilter{
		VersionIDs: versions, From: from, To: to,
	})
	if err != nil {
		return t, err
	}
	index, err := s.productIndex(ctx)
	if err != nil {
		return t, err
	}

	t.Title = "Remelt and refining report"
	t.Columns = []report.Column{
		{Header: "Date", Width: 12},
		{Header: "Series", Width: 9},
		{Header: "Product", Width: 22},
		{Header: "Remelt input (t)", Width: 16, Align: report.AlignRight, Numeric: true},
		{Header: "Output (t)", Width: 13, Align: report.AlignRight, Numeric: true},
		{Header: "Process loss (t)", Width: 16, Align: report.AlignRight, Numeric: true},
		{Header: "Rework (t)", Width: 12, Align: report.AlignRight, Numeric: true},
		{Header: "Rejected (t)", Width: 13, Align: report.AlignRight, Numeric: true},
		{Header: "Yield %", Width: 11, Align: report.AlignRight, Numeric: true},
	}

	// The two containers come back one after the other, so a plain listing puts
	// every plan day before every actual day. A refining report is read date by
	// date, with the plan and the actual for that day next to each other.
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.BusinessDate != b.BusinessDate {
			return a.BusinessDate < b.BusinessDate
		}
		if a.Series != b.Series {
			return a.Series < b.Series
		}
		return index[a.ProductID].Code < index[b.ProductID].Code
	})

	var input, output, loss, rework, rejected domain.Dec
	for _, p := range rows {
		product := index[p.ProductID]
		// Only the stages that remelt: a jumbo bag packed straight off the raw
		// house consumes no remelt and would sit in this report as a row of
		// zeroes claiming an infinite yield.
		if product.Stage != domain.StageRemelt && product.Stage != domain.StageRefining {
			continue
		}
		t.Rows = append(t.Rows, []report.Cell{
			report.Date(p.BusinessDate), report.Text(string(p.Series)),
			report.Text(product.Code + " " + product.Name),
			report.Num(p.RemeltInput, 3), report.Num(p.Quantity, 3),
			report.Num(p.ProcessLoss, 3), report.Num(p.Rework, 3), report.Num(p.Rejected, 3),
			yieldCell(p.Quantity, p.RemeltInput),
		})
		input = input.Add(p.RemeltInput)
		output = output.Add(p.Quantity)
		loss = loss.Add(p.ProcessLoss)
		rework = rework.Add(p.Rework)
		rejected = rejected.Add(p.Rejected)
	}
	t.Totals = []report.Cell{
		report.Text("Total"), report.Text(""), report.Text(""),
		report.Num(input, 3), report.Num(output, 3), report.Num(loss, 3),
		report.Num(rework, 3), report.Num(rejected, 3),
		yieldCell(output, input),
	}
	t.Notes = append(t.Notes,
		"Yield % = output / remelt input.",
		"Only products at the remelt and refining stages appear; sugar packed directly from the raw house is in the packing report.")
	return t, nil
}

// yieldCell renders a ratio, or nothing at all when there is no denominator.
//
// A day that consumed no remelt has no yield; printing 0 % would read as a
// catastrophic shift rather than as an idle refinery.
func yieldCell(output, input domain.Dec) report.Cell {
	if input.IsZero() {
		return report.Text("")
	}
	return report.Num(domain.RoundPct(domain.SafePct(output, input)), 3)
}

// ---------------------------------------------------------------------------
// Packing
// ---------------------------------------------------------------------------

// reportPacking counts packages, not tons.
//
// Tonnage is what the plan is written in and packages are what the packing hall
// and the warehouse count, so the conversion is the whole point of the report:
// a figure in tons cannot be checked against a pallet count without it.
func (s *Server) reportPacking(r *http.Request, t report.Table, season domain.Season,
	v domain.PlanVersion, from, to domain.BusinessDate) (report.Table, error) {

	ctx := r.Context()
	versions, err := s.planAndActual(ctx, season, v)
	if err != nil {
		return t, err
	}
	rows, err := s.store.Planning().ListProducts(ctx, store.PlanFilter{
		VersionIDs: versions, From: from, To: to,
	})
	if err != nil {
		return t, err
	}
	products, err := s.productIndex(ctx)
	if err != nil {
		return t, err
	}
	packagings, err := s.packagingIndex(ctx)
	if err != nil {
		return t, err
	}

	t.Title = "Packing report by package and product"
	t.Columns = []report.Column{
		{Header: "Product", Width: 22},
		{Header: "Package", Width: 22},
		{Header: "Net weight (kg)", Width: 15, Align: report.AlignRight, Numeric: true},
		{Header: "Series", Width: 9},
		{Header: "Quantity (t)", Width: 14, Align: report.AlignRight, Numeric: true},
		{Header: "Packages", Width: 14, Align: report.AlignRight, Numeric: true},
	}

	// Packing is asked about by package and product over a period, not day by
	// day: "how many jumbo bags this month" is the question, and a daily list
	// of 137 rows per package is a worse answer to it.
	type key struct {
		productID, packagingID string
		series                 domain.Series
	}
	totalsByKey := map[key]domain.Dec{}
	for _, p := range rows {
		if p.PackagingID == "" {
			continue
		}
		k := key{p.ProductID, p.PackagingID, p.Series}
		totalsByKey[k] = totalsByKey[k].Add(p.Quantity)
	}

	keys := make([]key, 0, len(totalsByKey))
	for k := range totalsByKey {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if products[a.productID].Code != products[b.productID].Code {
			return products[a.productID].Code < products[b.productID].Code
		}
		if packagings[a.packagingID].Code != packagings[b.packagingID].Code {
			return packagings[a.packagingID].Code < packagings[b.packagingID].Code
		}
		return a.series < b.series
	})

	tons, packages := domain.Zero, domain.Zero
	for _, k := range keys {
		qty := totalsByKey[k]
		pack := packagings[k.packagingID]
		count := domain.TonsToUnits(qty, pack.NetWeightKg)
		t.Rows = append(t.Rows, []report.Cell{
			report.Text(products[k.productID].Code + " " + products[k.productID].Name),
			report.Text(pack.Code + " " + pack.Name),
			report.Num(pack.NetWeightKg, 3), report.Text(string(k.series)),
			report.Num(qty, 3), report.Num(count, 0),
		})
		tons = tons.Add(qty)
		packages = packages.Add(count)
	}
	t.Totals = []report.Cell{
		report.Text("Total"), report.Text(""), report.Text(""), report.Text(""),
		report.Num(tons, 3), report.Num(packages, 0),
	}
	t.Notes = append(t.Notes,
		"Packages = quantity in tons / net weight per package. Package counts are rounded up to whole packages "+
			"when a requirement is derived; here they are the exact conversion of the planned tonnage.",
		"Rows with no packaging assigned are bulk output and do not appear.")
	return t, nil
}

// ---------------------------------------------------------------------------
// Production order variance
// ---------------------------------------------------------------------------

// reportOrderVariance compares what each order was released for against what it
// confirmed.
//
// The reason column is the one that makes the report worth running: an order
// that under-delivered without a reason is an unanswered question, and the
// report is where those become visible in one place rather than one order at a
// time.
func (s *Server) reportOrderVariance(r *http.Request, t report.Table, v domain.PlanVersion,
	from, to domain.BusinessDate) (report.Table, error) {

	ctx := r.Context()
	page, err := s.execution.ListOrders(ctx, store.ExecutionFilter{
		VersionID: v.ID, From: from, To: to, Top: 1000,
	})
	if err != nil {
		return t, err
	}
	products, err := s.productIndex(ctx)
	if err != nil {
		return t, err
	}
	lines, err := s.lineIndex(ctx)
	if err != nil {
		return t, err
	}

	t.Title = "Production order variance report"
	t.Columns = []report.Column{
		{Header: "Order", Width: 20},
		{Header: "Date", Width: 12},
		{Header: "Line", Width: 16},
		{Header: "Product", Width: 22},
		{Header: "Planned (t)", Width: 14, Align: report.AlignRight, Numeric: true},
		{Header: "Confirmed (t)", Width: 14, Align: report.AlignRight, Numeric: true},
		{Header: "Variance (t)", Width: 14, Align: report.AlignRight, Numeric: true},
		{Header: "Achievement %", Width: 14, Align: report.AlignRight, Numeric: true},
		{Header: "Status", Width: 12},
		{Header: "Variance reason", Width: 30},
	}

	planned, confirmed := domain.Zero, domain.Zero
	authorised, open := 0, 0
	for _, o := range page.Items {
		variance := domain.Variance(o.ConfirmedQty, o.PlannedQty)
		// An order cannot be closed outside tolerance without a reason, so a
		// recorded reason marks an exception somebody signed off. Counting them
		// is what makes this report worth running over a month: the individual
		// rows say what happened, the count says how often it had to.
		if o.VarianceReason != "" {
			authorised++
		}
		// An order still in process is allowed to be short of its plan. Saying
		// how many are still open stops the totals being read as final.
		if o.Status != domain.OrderCompleted && o.Status != domain.OrderTechnicallyClosed &&
			o.Status != domain.OrderCancelled {
			open++
		}
		t.Rows = append(t.Rows, []report.Cell{
			report.Text(o.OrderNo), report.Date(o.BusinessDate),
			report.Text(lines[o.LineID]),
			report.Text(products[o.ProductID].Code + " " + products[o.ProductID].Name),
			report.Num(o.PlannedQty, 3), report.Num(o.ConfirmedQty, 3),
			report.Num(variance, 3),
			report.Num(domain.AchievementPct(o.ConfirmedQty, o.PlannedQty), 3),
			report.Text(string(o.Status)), report.Text(o.VarianceReason),
		})
		planned = planned.Add(o.PlannedQty)
		confirmed = confirmed.Add(o.ConfirmedQty)
	}
	t.Totals = []report.Cell{
		report.Text("Total"), report.Text(""), report.Text(""), report.Text(""),
		report.Num(planned, 3), report.Num(confirmed, 3),
		report.Num(domain.Variance(confirmed, planned), 3),
		report.Num(domain.AchievementPct(confirmed, planned), 3),
		report.Text(""), report.Text(""),
	}
	t.Notes = append(t.Notes,
		"Variance = confirmed quantity - planned quantity; a negative figure is a shortfall.",
		"An order outside the close tolerance cannot be closed without a variance reason, "+
			"so an authorised exception is one somebody accepted rather than one nobody noticed.",
		plural(authorised, "%d order closed with an authorised variance reason.",
			"%d orders closed with an authorised variance reason."),
		plural(open, "%d order is still open and may yet confirm more.",
			"%d orders are still open and may yet confirm more."))
	return t, nil
}

// ---------------------------------------------------------------------------
// Downtime and lost production
// ---------------------------------------------------------------------------

// reportDowntime lists every stoppage with the tonnage it cost.
//
// Lost tons are an estimate - the line's rated throughput multiplied by the
// hours it stood - and the report says so, because a figure that looks measured
// and is not will end up in somebody's variance explanation.
func (s *Server) reportDowntime(r *http.Request, t report.Table, season domain.Season,
	from, to domain.BusinessDate) (report.Table, error) {

	ctx := r.Context()
	if err := requirePermission(r, domain.PermPlanRead); err != nil {
		return t, err
	}
	events, err := s.store.Planning().ListDowntime(ctx, store.PlanFilter{
		FactoryID: season.FactoryID, From: from, To: to,
	})
	if err != nil {
		return t, err
	}
	lines, err := s.lineIndex(ctx)
	if err != nil {
		return t, err
	}
	rated, err := s.ratedTPH(ctx, season.FactoryID)
	if err != nil {
		return t, err
	}
	reasons, err := s.reasonIndex(ctx)
	if err != nil {
		return t, err
	}
	shifts, err := s.shiftIndex(ctx)
	if err != nil {
		return t, err
	}

	t.Title = "Downtime and lost production report"
	t.Columns = []report.Column{
		{Header: "Date", Width: 12},
		{Header: "Line", Width: 16},
		{Header: "Shift", Width: 12},
		{Header: "Type", Width: 11},
		{Header: "Reason", Width: 24},
		{Header: "Hours", Width: 10, Align: report.AlignRight, Numeric: true},
		{Header: "Lost tons", Width: 13, Align: report.AlignRight, Numeric: true},
		{Header: "Root cause", Width: 30},
		{Header: "Corrective action", Width: 30},
	}

	sort.SliceStable(events, func(i, j int) bool {
		if events[i].BusinessDate != events[j].BusinessDate {
			return events[i].BusinessDate < events[j].BusinessDate
		}
		return events[i].StartAt.Before(events[j].StartAt)
	})

	hours, lost := domain.Zero, domain.Zero
	for _, e := range events {
		kind := "Unplanned"
		if e.Planned {
			kind = "Planned"
		}
		eventLost := domain.LostTons(e.DurationHrs, rated[e.LineID])
		t.Rows = append(t.Rows, []report.Cell{
			report.Date(e.BusinessDate), report.Text(lines[e.LineID]),
			report.Text(shifts[e.ShiftID]), report.Text(kind),
			report.Text(reasonLabel(reasons, e.ReasonCode)),
			report.Num(e.DurationHrs, 2), report.Num(eventLost, 3),
			report.Text(e.RootCause), report.Text(e.Action),
		})
		hours = hours.Add(e.DurationHrs)
		lost = lost.Add(eventLost)
	}
	t.Totals = []report.Cell{
		report.Text("Total"), report.Text(""), report.Text(""), report.Text(""),
		report.Text(""), report.Num(hours, 2), report.Num(lost, 3),
		report.Text(""), report.Text(""),
	}
	t.Notes = append(t.Notes,
		"Lost tons are an estimate: stoppage hours multiplied by the line's rated throughput. "+
			"An event on no particular line has no rated throughput and so costs no estimated tonnage.",
		"The ranking of reasons by hours lost is on the executive overview.")
	return t, nil
}

// ---------------------------------------------------------------------------
// Quality results and holds
// ---------------------------------------------------------------------------

// reportQuality lists every laboratory result with the limits it was judged
// against and the hold it caused, if any.
//
// The limits are the ones recorded on the result rather than the specification
// in force today. A specification that changed after the test would otherwise
// silently re-judge results that were passed months ago.
func (s *Server) reportQuality(r *http.Request, t report.Table, season domain.Season,
	from, to domain.BusinessDate) (report.Table, error) {

	ctx := r.Context()
	page, err := s.execution.ListSamples(ctx, store.ExecutionFilter{
		FactoryID: season.FactoryID, From: from, To: to, Top: 1000,
	})
	if err != nil {
		return t, err
	}
	holds, err := s.execution.ListHolds(ctx, store.ExecutionFilter{
		FactoryID: season.FactoryID, Top: 1000,
	})
	if err != nil {
		return t, err
	}
	products, err := s.productIndex(ctx)
	if err != nil {
		return t, err
	}
	parameters, err := s.parameterIndex(ctx)
	if err != nil {
		return t, err
	}
	batches, err := s.batchIndex(ctx, season.FactoryID)
	if err != nil {
		return t, err
	}

	holdBySample := map[string]domain.QualityHold{}
	for _, h := range holds {
		holdBySample[h.SampleID] = h
	}

	t.Title = "Quality results and hold report"
	t.Columns = []report.Column{
		{Header: "Date", Width: 12},
		{Header: "Sample", Width: 20},
		{Header: "Product", Width: 22},
		{Header: "Batch", Width: 16},
		{Header: "Parameter", Width: 22},
		{Header: "Value", Width: 12, Align: report.AlignRight, Numeric: true},
		{Header: "UOM", Width: 8},
		{Header: "Lower", Width: 12, Align: report.AlignRight, Numeric: true},
		{Header: "Upper", Width: 12, Align: report.AlignRight, Numeric: true},
		{Header: "Result", Width: 11},
		{Header: "Hold", Width: 30},
	}

	limit := func(d *domain.Dec) report.Cell {
		if d == nil {
			return report.Text("")
		}
		return report.Num(*d, 3)
	}

	failures, open := 0, 0
	for _, sample := range page.Items {
		hold := holdText(holdBySample[sample.ID])
		if h, ok := holdBySample[sample.ID]; ok && h.ReleasedOn == "" {
			open++
		}
		if len(sample.Results) == 0 {
			// A sample taken and not yet tested is a fact worth reporting: it is
			// why a batch is still waiting.
			t.Rows = append(t.Rows, []report.Cell{
				report.Date(sample.BusinessDate), report.Text(sample.SampleNo),
				report.Text(products[sample.ProductID].Code + " " + products[sample.ProductID].Name),
				report.Text(batches[sample.BatchID]), report.Text("(no results)"),
				report.Text(""), report.Text(""), report.Text(""), report.Text(""),
				report.Text(sample.Status), report.Text(hold),
			})
			continue
		}
		for _, res := range sample.Results {
			if res.Status == domain.QualityFail {
				failures++
			}
			t.Rows = append(t.Rows, []report.Cell{
				report.Date(sample.BusinessDate), report.Text(sample.SampleNo),
				report.Text(products[sample.ProductID].Code + " " + products[sample.ProductID].Name),
				report.Text(batches[sample.BatchID]),
				report.Text(parameters[res.ParameterID]),
				report.Num(res.Value, 3), report.Text(res.UOM),
				limit(res.LowerLimit), limit(res.UpperLimit),
				report.Text(string(res.Status)), report.Text(hold),
			})
		}
	}
	t.Notes = append(t.Notes,
		"Limits are the ones recorded against the result when it was entered, not the specification in force today.",
		plural(failures, "%d result outside specification.", "%d results outside specification."),
		plural(open, "%d hold still open.", "%d holds still open."))
	return t, nil
}

// holdText describes a sample's hold in the one column the report has for it.
func holdText(h domain.QualityHold) string {
	switch {
	case h.ID == "":
		return ""
	case h.ReleasedOn != "":
		return "Released " + string(h.ReleasedOn) + " (" + h.Reason + ")"
	default:
		return "Held since " + string(h.PlacedOn) + " (" + h.Reason + ")"
	}
}

// plural renders a count with the wording that fits it, because "1 results
// outside specification" in a printed report looks like a broken system.
func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf(one, n)
	}
	return fmt.Sprintf(many, n)
}

// reasonLabel prefers the configured name and falls back to the raw code, so an
// event recorded against a reason somebody has since deactivated still reads as
// something rather than as a blank.
func reasonLabel(reasons map[string]string, code string) string {
	if code == "" {
		return ""
	}
	if name, ok := reasons[code]; ok {
		return code + " " + name
	}
	return code
}

// ---------------------------------------------------------------------------
// Lookups
// ---------------------------------------------------------------------------

// planAndActual returns the two version containers a target-versus-actual
// report has to read: the plan the user asked for and the season's actuals.
//
// They are separate containers, so a report that read only the version in the
// URL would have a Series column that could never say anything but PLAN.
func (s *Server) planAndActual(ctx context.Context, season domain.Season,
	v domain.PlanVersion) ([]string, error) {

	_, actual, err := s.analytics.Versions(ctx, season.ID, v.ID)
	if err != nil {
		return nil, err
	}
	if actual.ID == v.ID {
		return []string{v.ID}, nil
	}
	return []string{v.ID, actual.ID}, nil
}

func (s *Server) productIndex(ctx context.Context) (map[string]domain.Product, error) {
	page, err := s.store.MasterData().Products().List(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return nil, err
	}
	out := make(map[string]domain.Product, len(page.Items))
	for _, p := range page.Items {
		out[p.ID] = p
	}
	return out, nil
}

func (s *Server) packagingIndex(ctx context.Context) (map[string]domain.PackagingType, error) {
	page, err := s.store.MasterData().PackagingTypes().List(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return nil, err
	}
	out := make(map[string]domain.PackagingType, len(page.Items))
	for _, p := range page.Items {
		out[p.ID] = p
	}
	return out, nil
}

func (s *Server) lineIndex(ctx context.Context) (map[string]string, error) {
	page, err := s.store.MasterData().Lines().List(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(page.Items))
	for _, l := range page.Items {
		out[l.ID] = l.Code + " " + l.Name
	}
	return out, nil
}

// ratedTPH is the throughput used to price a stoppage. It is read the same way
// the dashboard reads it so that the report and the KPI cannot disagree about
// how many tons an hour of downtime cost.
func (s *Server) ratedTPH(ctx context.Context, factoryID string) (map[string]domain.Dec, error) {
	page, err := s.store.MasterData().Lines().List(ctx, store.ListOptions{
		Top: 1000, ParentID: factoryID,
	})
	if err != nil {
		return nil, err
	}
	out := make(map[string]domain.Dec, len(page.Items))
	for _, l := range page.Items {
		out[l.ID] = l.RatedTPH
	}
	return out, nil
}

func (s *Server) shiftIndex(ctx context.Context) (map[string]string, error) {
	page, err := s.store.MasterData().Shifts().List(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(page.Items))
	for _, sh := range page.Items {
		out[sh.ID] = sh.Code
	}
	return out, nil
}

func (s *Server) reasonIndex(ctx context.Context) (map[string]string, error) {
	page, err := s.store.MasterData().ReasonCodes().List(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(page.Items))
	for _, rc := range page.Items {
		out[rc.Code] = rc.Name
	}
	return out, nil
}

func (s *Server) parameterIndex(ctx context.Context) (map[string]string, error) {
	params, err := s.store.Execution().ListParameters(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(params))
	for _, p := range params {
		out[p.ID] = p.Code + " " + p.Name
	}
	return out, nil
}

func (s *Server) batchIndex(ctx context.Context, factoryID string) (map[string]string, error) {
	page, err := s.store.Execution().ListBatches(ctx, store.ExecutionFilter{
		FactoryID: factoryID, Top: 1000,
	})
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(page.Items))
	for _, b := range page.Items {
		out[b.ID] = b.Code
	}
	return out, nil
}

// planKey identifies one reconciled cell: a date within a series. Plan and
// actual are kept apart because a recovery calculated from one series' cane and
// the other's raw sugar would be a number describing nothing.
type planKey struct {
	date   domain.BusinessDate
	series domain.Series
}

// sortedPlanKeys orders those pairs so that a report reads chronologically
// rather than in map order.
func sortedPlanKeys[V any](m map[planKey]V) []planKey {
	out := make([]planKey, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].date != out[j].date {
			return out[i].date < out[j].date
		}
		return out[i].series < out[j].series
	})
	return out
}
