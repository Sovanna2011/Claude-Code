package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/report"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
)

// reportDef describes one report in the catalogue.
type reportDef struct {
	Code        string   `json:"code"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Parameters  []string `json:"parameters"`
	Formats     []string `json:"formats"`
}

// reportCatalogue is the list the Reports page renders. Each entry maps to a
// builder below; adding a report means adding to both, and the test asserts
// that the two stay in step.
var reportCatalogue = []reportDef{
	{Code: "daily-plan", Title: "Daily production plan",
		Description: "Daily cane, raw sugar and finished goods targets for a plan version.",
		Parameters:  []string{"versionId", "from", "to"}, Formats: []string{"csv", "xlsx", "pdf"}},
	{Code: "target-vs-actual", Title: "Daily target versus actual",
		Description: "Cane crushed by day with cumulative target, actual, variance and achievement.",
		Parameters:  []string{"seasonId", "versionId", "from", "to"}, Formats: []string{"csv", "xlsx", "pdf"}},
	{Code: "cane-crushing", Title: "Cane crushing report",
		Description: "Cane delivered, accepted, rejected and crushed with rate and utilisation.",
		Parameters:  []string{"versionId", "from", "to"}, Formats: []string{"csv", "xlsx", "pdf"}},
	{Code: "stock-ledger", Title: "Warehouse and silo daily stock ledger",
		Description: "Beginning balance, movements, ending balance and capacity use per store.",
		Parameters:  []string{"versionId", "warehouseId", "from", "to"}, Formats: []string{"csv", "xlsx", "pdf"}},
	{Code: "capacity-forecast", Title: "Capacity forecast and overflow risk",
		Description: "Stock against capacity per warehouse with the first breach date and the shipment rate needed.",
		Parameters:  []string{"seasonId", "versionId"}, Formats: []string{"csv", "xlsx", "pdf"}},
	{Code: "shipment-plan", Title: "Shipment plan and actual by channel",
		Description: "Planned and actual shipment by customer channel.",
		Parameters:  []string{"seasonId", "versionId", "from", "to"}, Formats: []string{"csv", "xlsx", "pdf"}},
	{Code: "material-requirements", Title: "Packaging material requirement and shortage",
		Description: "Gross requirement, cover and purchase requirement per packaging material.",
		Parameters:  []string{"versionId"}, Formats: []string{"csv", "xlsx", "pdf"}},
	{Code: "season-summary", Title: "Seasonal production summary",
		Description: "Season totals for cane, raw sugar, finished goods and shipment against plan.",
		Parameters:  []string{"seasonId", "versionId"}, Formats: []string{"csv", "xlsx", "pdf"}},
	{Code: "approval-history", Title: "Plan version and approval history",
		Description: "Every workflow action on a plan version with actor, time and reason.",
		Parameters:  []string{"versionId"}, Formats: []string{"csv", "xlsx", "pdf"}},
}

func (s *Server) handleListReports(w http.ResponseWriter, r *http.Request) {
	if err := requirePermission(r, domain.PermReportRead); err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[reportDef]{Value: reportCatalogue, Count: len(reportCatalogue)})
}

// handleRunReport builds the report and renders it in the requested format.
func (s *Server) handleRunReport(w http.ResponseWriter, r *http.Request) {
	if err := requirePermission(r, domain.PermReportRead); err != nil {
		writeProblem(w, r, err)
		return
	}
	code := r.PathValue("code")
	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "" {
		format = "json"
	}

	table, err := s.buildReport(r, code)
	if err != nil {
		writeProblem(w, r, err)
		return
	}

	switch format {
	case "json":
		writeJSON(w, tableToJSON(table))
	case "csv":
		body, err := report.CSV(table)
		if err != nil {
			writeProblem(w, r, err)
			return
		}
		sendFile(w, table.FileName("csv"), "text/csv; charset=utf-8", body)
	case "xlsx":
		body, err := report.XLSX(table)
		if err != nil {
			writeProblem(w, r, err)
			return
		}
		sendFile(w, table.FileName("xlsx"),
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", body)
	case "pdf":
		body, err := report.PDF(table)
		if err != nil {
			writeProblem(w, r, err)
			return
		}
		sendFile(w, table.FileName("pdf"), "application/pdf", body)
	default:
		writeProblem(w, r, wrapValidation(
			fmt.Sprintf("%q is not a supported format; use json, csv, xlsx or pdf", format)))
	}
}

func sendFile(w http.ResponseWriter, name, contentType string, body []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	w.Header().Set("Content-Length", fmt.Sprint(len(body)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(body)
}

// tableToJSON renders the report as JSON for the SAPUI5 preview, keeping the
// same metadata as the file exports.
func tableToJSON(t report.Table) map[string]any {
	columns := make([]map[string]any, len(t.Columns))
	for i, c := range t.Columns {
		columns[i] = map[string]any{
			"header": c.Header, "align": string(c.Align), "numeric": c.Numeric,
		}
	}
	rows := make([][]string, len(t.Rows))
	for i, row := range t.Rows {
		cells := make([]string, len(t.Columns))
		for j := range cells {
			if j < len(row) {
				cells[j] = row[j].Text
			}
		}
		rows[i] = cells
	}
	var totals []string
	if len(t.Totals) > 0 {
		totals = make([]string, len(t.Columns))
		for j := range totals {
			if j < len(t.Totals) {
				totals[j] = t.Totals[j].Text
			}
		}
	}
	return map[string]any{
		"code": t.Code, "title": t.Title, "metadata": t.MetadataLines(),
		"columns": columns, "rows": rows, "totals": totals, "notes": t.Notes,
	}
}

// ---------------------------------------------------------------------------
// Report builders
// ---------------------------------------------------------------------------

// buildReport dispatches on the report code and assembles the table.
func (s *Server) buildReport(r *http.Request, code string) (report.Table, error) {
	ctx := r.Context()
	q := r.URL.Query()
	caller := auth.FromContext(ctx)

	t := report.Table{
		Code: code, GeneratedAt: time.Now().UTC(), GeneratedBy: caller.DisplayName,
	}
	if t.GeneratedBy == "" {
		t.GeneratedBy = caller.Username
	}

	from := domain.BusinessDate(q.Get("from"))
	to := domain.BusinessDate(q.Get("to"))
	if from != "" {
		t.Filters = append(t.Filters, report.Filter{Label: "From", Value: string(from)})
	}
	if to != "" {
		t.Filters = append(t.Filters, report.Filter{Label: "To", Value: string(to)})
	}

	// Resolve the plan version and its season for the metadata block, which
	// every report carries.
	versionID := q.Get("versionId")
	var version domain.PlanVersion
	var season domain.Season
	if versionID != "" {
		detail, err := s.planning.GetVersion(ctx, versionID)
		if err != nil {
			return t, err
		}
		version = detail.Version
		season, err = s.planning.GetSeason(ctx, version.SeasonID)
		if err != nil {
			return t, err
		}
	} else if seasonID := q.Get("seasonId"); seasonID != "" {
		var err error
		season, err = s.planning.GetSeason(ctx, seasonID)
		if err != nil {
			return t, err
		}
	} else {
		return t, wrapValidation("the report needs either a versionId or a seasonId")
	}

	t.Season = season.Code
	t.PlanVersion = version.Code
	if factory, err := s.store.MasterData().Factories().Get(ctx, season.FactoryID); err == nil {
		t.Factory = factory.Code + " " + factory.Name
		if company, err := s.store.MasterData().Companies().Get(ctx, factory.CompanyID); err == nil {
			t.Company = company.Name
		}
	}

	switch code {
	case "daily-plan":
		return s.reportDailyPlan(r, t, version, from, to)
	case "target-vs-actual":
		return s.reportTargetVsActual(r, t, season, version)
	case "cane-crushing":
		return s.reportCaneCrushing(r, t, version, from, to)
	case "stock-ledger":
		return s.reportStockLedger(r, t, version, from, to)
	case "capacity-forecast":
		return s.reportCapacityForecast(r, t, season, version)
	case "shipment-plan":
		return s.reportShipment(r, t, season, version, from, to)
	case "material-requirements":
		return s.reportMaterials(r, t, version)
	case "season-summary":
		return s.reportSeasonSummary(r, t, season, version)
	case "approval-history":
		return s.reportApprovalHistory(r, t, version)
	default:
		return t, fmt.Errorf("%w: report %q", domain.ErrNotFound, code)
	}
}

func (s *Server) reportDailyPlan(r *http.Request, t report.Table, v domain.PlanVersion,
	from, to domain.BusinessDate) (report.Table, error) {

	ctx := r.Context()
	t.Title = "Daily production plan"
	t.Columns = []report.Column{
		{Header: "Date", Width: 12},
		{Header: "Cane crushed (t)", Width: 16, Align: report.AlignRight, Numeric: true},
		{Header: "Raw sugar (t)", Width: 16, Align: report.AlignRight, Numeric: true},
		{Header: "Finished goods (t)", Width: 18, Align: report.AlignRight, Numeric: true},
		{Header: "Remelt input (t)", Width: 16, Align: report.AlignRight, Numeric: true},
		{Header: "Shipment (t)", Width: 14, Align: report.AlignRight, Numeric: true},
	}

	f := store.PlanFilter{VersionIDs: []string{v.ID}, From: from, To: to, Series: domain.SeriesPlan}
	cane, err := s.store.Planning().ListCane(ctx, f)
	if err != nil {
		return t, err
	}
	products, err := s.store.Planning().ListProducts(ctx, f)
	if err != nil {
		return t, err
	}
	shipments, err := s.store.Planning().ListShipments(ctx, f)
	if err != nil {
		return t, err
	}

	assumptions, err := s.store.Planning().ListAssumptions(ctx, v.ID)
	if err != nil {
		return t, err
	}
	recovery := domain.Zero
	for _, a := range assumptions {
		if a.Code == domain.AsmRecoveryPct {
			recovery = a.Value
		}
	}

	type day struct{ cane, finished, remelt, shipment domain.Dec }
	byDate := map[domain.BusinessDate]*day{}
	get := func(d domain.BusinessDate) *day {
		if x, ok := byDate[d]; ok {
			return x
		}
		x := &day{}
		byDate[d] = x
		return x
	}
	for _, c := range cane {
		get(c.BusinessDate).cane = get(c.BusinessDate).cane.Add(c.CaneCrushed)
	}
	for _, p := range products {
		d := get(p.BusinessDate)
		d.finished = d.finished.Add(p.Quantity)
		d.remelt = d.remelt.Add(p.RemeltInput)
	}
	for _, sh := range shipments {
		get(sh.BusinessDate).shipment = get(sh.BusinessDate).shipment.Add(sh.Quantity)
	}

	var totals day
	for _, d := range sortedKeys(byDate) {
		x := byDate[d]
		raw := domain.ExpectedRawSugar(x.cane, recovery)
		t.Rows = append(t.Rows, []report.Cell{
			report.Date(d), report.Num(x.cane, 3), report.Num(raw, 3),
			report.Num(x.finished, 3), report.Num(x.remelt, 3), report.Num(x.shipment, 3),
		})
		totals.cane = totals.cane.Add(x.cane)
		totals.finished = totals.finished.Add(x.finished)
		totals.remelt = totals.remelt.Add(x.remelt)
		totals.shipment = totals.shipment.Add(x.shipment)
	}
	t.Totals = []report.Cell{
		report.Text("Total"), report.Num(totals.cane, 3),
		report.Num(domain.ExpectedRawSugar(totals.cane, recovery), 3),
		report.Num(totals.finished, 3), report.Num(totals.remelt, 3), report.Num(totals.shipment, 3),
	}
	t.Notes = append(t.Notes, fmt.Sprintf(
		"Raw sugar is calculated as cane crushed x %s %% recovery (assumption %s).",
		recovery, domain.AsmRecoveryPct))
	return t, nil
}

func (s *Server) reportTargetVsActual(r *http.Request, t report.Table,
	season domain.Season, v domain.PlanVersion) (report.Table, error) {

	dash, err := s.analytics.Dashboard(r.Context(), service.DashboardRequest{
		SeasonID: season.ID, PlanVersionID: v.ID,
	})
	if err != nil {
		return t, err
	}
	t.Title = "Daily cane target versus actual"
	t.Columns = []report.Column{
		{Header: "Date", Width: 12},
		{Header: "Target (t)", Width: 14, Align: report.AlignRight, Numeric: true},
		{Header: "Actual (t)", Width: 14, Align: report.AlignRight, Numeric: true},
		{Header: "Variance (t)", Width: 14, Align: report.AlignRight, Numeric: true},
		{Header: "Cum. target (t)", Width: 16, Align: report.AlignRight, Numeric: true},
		{Header: "Cum. actual (t)", Width: 16, Align: report.AlignRight, Numeric: true},
		{Header: "Achievement %", Width: 14, Align: report.AlignRight, Numeric: true},
	}
	for _, p := range dash.CaneTrend {
		row := []report.Cell{
			report.Date(p.Date), report.Num(p.Target, 3),
		}
		if p.HasActual {
			row = append(row, report.Num(p.Actual, 3), report.Num(p.Variance, 3))
		} else {
			row = append(row, report.Text(""), report.Text(""))
		}
		row = append(row, report.Num(p.CumTarget, 3))
		if p.HasActual {
			row = append(row, report.Num(p.CumActual, 3), report.Num(p.AchievementPct, 3))
		} else {
			row = append(row, report.Text(""), report.Text(""))
		}
		t.Rows = append(t.Rows, row)
	}
	t.Notes = append(t.Notes,
		"Days without a recorded actual are left blank rather than shown as zero.",
		fmt.Sprintf("As of %s: %s t crushed against a target of %s t (%s %%).",
			dash.AsOf, dash.Cane.CumulativeActual, dash.Cane.CumulativeTarget, dash.Cane.AchievementPct))
	return t, nil
}

func (s *Server) reportCaneCrushing(r *http.Request, t report.Table, v domain.PlanVersion,
	from, to domain.BusinessDate) (report.Table, error) {

	rows, err := s.store.Planning().ListCane(r.Context(), store.PlanFilter{
		VersionIDs: []string{v.ID}, From: from, To: to,
	})
	if err != nil {
		return t, err
	}
	t.Title = "Cane crushing report"
	t.Columns = []report.Column{
		{Header: "Date", Width: 12},
		{Header: "Series", Width: 9},
		{Header: "Delivered (t)", Width: 14, Align: report.AlignRight, Numeric: true},
		{Header: "Accepted (t)", Width: 14, Align: report.AlignRight, Numeric: true},
		{Header: "Rejected (t)", Width: 13, Align: report.AlignRight, Numeric: true},
		{Header: "Crushed (t)", Width: 14, Align: report.AlignRight, Numeric: true},
		{Header: "Rate (t/h)", Width: 12, Align: report.AlignRight, Numeric: true},
		{Header: "Stoppage (h)", Width: 13, Align: report.AlignRight, Numeric: true},
		{Header: "Utilisation %", Width: 13, Align: report.AlignRight, Numeric: true},
	}
	crushed := domain.Zero
	for _, c := range rows {
		t.Rows = append(t.Rows, []report.Cell{
			report.Date(c.BusinessDate), report.Text(string(c.Series)),
			report.Num(c.CaneDelivered, 3), report.Num(c.CaneAccepted, 3),
			report.Num(c.CaneRejected, 3), report.Num(c.CaneCrushed, 3),
			report.Num(c.CrushRateTPH, 3), report.Num(c.StoppageHrs, 2),
			report.Num(domain.UtilisationPct(c.AvailableHrs, c.StoppageHrs), 3),
		})
		crushed = crushed.Add(c.CaneCrushed)
	}
	t.Totals = []report.Cell{
		report.Text("Total"), report.Text(""), report.Text(""), report.Text(""),
		report.Text(""), report.Num(crushed, 3),
	}
	return t, nil
}

func (s *Server) reportStockLedger(r *http.Request, t report.Table, v domain.PlanVersion,
	from, to domain.BusinessDate) (report.Table, error) {

	ctx := r.Context()
	warehouseID := r.URL.Query().Get("warehouseId")
	rows, err := s.store.Planning().ListStorage(ctx, store.PlanFilter{
		VersionIDs: []string{v.ID}, From: from, To: to,
		WarehouseIDs: splitCSV(warehouseID),
	})
	if err != nil {
		return t, err
	}
	names, capacities, err := s.warehouseIndex(ctx)
	if err != nil {
		return t, err
	}
	products, err := s.productNames(ctx)
	if err != nil {
		return t, err
	}

	t.Title = "Warehouse and silo daily stock ledger"
	t.Columns = []report.Column{
		{Header: "Warehouse", Width: 18},
		{Header: "Product", Width: 14},
		{Header: "Date", Width: 12},
		{Header: "Beginning (t)", Width: 14, Align: report.AlignRight, Numeric: true},
		{Header: "Receipt (t)", Width: 13, Align: report.AlignRight, Numeric: true},
		{Header: "Remelt (t)", Width: 12, Align: report.AlignRight, Numeric: true},
		{Header: "Shipment (t)", Width: 13, Align: report.AlignRight, Numeric: true},
		{Header: "Adjustment (t)", Width: 14, Align: report.AlignRight, Numeric: true},
		{Header: "Ending (t)", Width: 13, Align: report.AlignRight, Numeric: true},
		{Header: "Capacity use %", Width: 14, Align: report.AlignRight, Numeric: true},
	}
	for _, x := range rows {
		usable := capacities[x.WarehouseID]
		t.Rows = append(t.Rows, []report.Cell{
			report.Text(names[x.WarehouseID]), report.Text(products[x.ProductID]),
			report.Date(x.BusinessDate),
			report.Num(x.BeginningBalance, 3), report.Num(x.ProductionReceipt.Add(x.TransferIn), 3),
			report.Num(x.RemeltIssue, 3), report.Num(x.ShipmentQty, 3),
			report.Num(x.Adjustment, 3), report.Num(x.EndingBalance, 3),
			report.Num(domain.CapacityUsePct(x.EndingBalance, usable), 3),
		})
	}
	t.Notes = append(t.Notes,
		"Capacity use is measured against usable capacity, not nominal capacity.")
	return t, nil
}

func (s *Server) reportCapacityForecast(r *http.Request, t report.Table,
	season domain.Season, v domain.PlanVersion) (report.Table, error) {

	dash, err := s.analytics.Dashboard(r.Context(), service.DashboardRequest{
		SeasonID: season.ID, PlanVersionID: v.ID,
	})
	if err != nil {
		return t, err
	}
	t.Title = "Capacity forecast and overflow risk"
	t.Columns = []report.Column{
		{Header: "Warehouse", Width: 20},
		{Header: "Class", Width: 10},
		{Header: "Capacity (t)", Width: 14, Align: report.AlignRight, Numeric: true},
		{Header: "Usable (t)", Width: 13, Align: report.AlignRight, Numeric: true},
		{Header: "Closing stock (t)", Width: 16, Align: report.AlignRight, Numeric: true},
		{Header: "Use %", Width: 10, Align: report.AlignRight, Numeric: true},
		{Header: "First warning", Width: 14},
		{Header: "First full", Width: 13},
		{Header: "Shipment needed (t/day)", Width: 20, Align: report.AlignRight, Numeric: true},
		{Header: "Status", Width: 10},
	}
	for _, k := range dash.Storage {
		t.Rows = append(t.Rows, []report.Cell{
			report.Text(k.WarehouseCode + " " + k.WarehouseName),
			report.Text(string(k.StorageClass)),
			report.Num(k.CapacityTons, 3), report.Num(k.UsableTons, 3),
			report.Num(k.BalanceTons, 3), report.Num(k.CapacityUsePct, 3),
			report.Date(k.FirstWarnDate), report.Date(k.FirstFullDate),
			report.Num(k.RequiredShipTPD, 3), report.Text(string(k.Severity)),
		})
	}
	t.Notes = append(t.Notes,
		"Shipment needed is the smallest constant daily rate that keeps the store within its alert threshold "+
			"across the whole plan horizon.")
	return t, nil
}

func (s *Server) reportShipment(r *http.Request, t report.Table, season domain.Season,
	v domain.PlanVersion, from, to domain.BusinessDate) (report.Table, error) {

	dash, err := s.analytics.Dashboard(r.Context(), service.DashboardRequest{
		SeasonID: season.ID, PlanVersionID: v.ID, From: from, To: to,
	})
	if err != nil {
		return t, err
	}
	t.Title = "Shipment plan and actual by channel"
	t.Columns = []report.Column{
		{Header: "Channel", Width: 22},
		{Header: "Planned (t)", Width: 14, Align: report.AlignRight, Numeric: true},
		{Header: "Actual (t)", Width: 14, Align: report.AlignRight, Numeric: true},
		{Header: "Variance (t)", Width: 14, Align: report.AlignRight, Numeric: true},
	}
	planned, actual, variance := domain.Zero, domain.Zero, domain.Zero
	for _, c := range dash.Shipments {
		t.Rows = append(t.Rows, []report.Cell{
			report.Text(c.ChannelCode + " " + c.ChannelName),
			report.Num(c.PlannedTons, 3), report.Num(c.ActualTons, 3), report.Num(c.VarianceTons, 3),
		})
		planned = planned.Add(c.PlannedTons)
		actual = actual.Add(c.ActualTons)
		variance = variance.Add(c.VarianceTons)
	}
	t.Totals = []report.Cell{
		report.Text("Total"), report.Num(planned, 3), report.Num(actual, 3), report.Num(variance, 3),
	}
	return t, nil
}

func (s *Server) reportMaterials(r *http.Request, t report.Table, v domain.PlanVersion) (report.Table, error) {
	reqs, err := s.materials.Requirements(r.Context(), service.RequirementsRequest{VersionID: v.ID})
	if err != nil {
		return t, err
	}
	t.Title = "Packaging material requirement and shortage"
	t.Columns = []report.Column{
		{Header: "Material", Width: 22},
		{Header: "Unit", Width: 8},
		{Header: "Required", Width: 14, Align: report.AlignRight, Numeric: true},
		{Header: "Safety stock", Width: 13, Align: report.AlignRight, Numeric: true},
		{Header: "Available", Width: 13, Align: report.AlignRight, Numeric: true},
		{Header: "On order", Width: 13, Align: report.AlignRight, Numeric: true},
		{Header: "Shortage", Width: 13, Align: report.AlignRight, Numeric: true},
		{Header: "Purchase", Width: 13, Align: report.AlignRight, Numeric: true},
		{Header: "Required by", Width: 13},
		{Header: "Order by", Width: 13},
		{Header: "Status", Width: 10},
	}
	for _, m := range reqs {
		t.Rows = append(t.Rows, []report.Cell{
			report.Text(m.MaterialCode + " " + m.MaterialName), report.Text(m.UOM),
			report.Num(m.GrossRequired, 0), report.Num(m.SafetyStock, 0),
			report.Num(m.OnHand, 0), report.Num(m.OnOrder, 0),
			report.Num(m.Shortage, 0), report.Num(m.PurchaseQty, 0),
			report.Date(m.RequiredBy), report.Date(m.SuggestedOrder),
			report.Text(string(m.Severity)),
		})
	}
	t.Notes = append(t.Notes,
		"Purchase requirement = max(0, gross requirement + safety stock - available - on order).",
		"The suggested order date is the required-by date less the material lead time.")
	return t, nil
}

func (s *Server) reportSeasonSummary(r *http.Request, t report.Table,
	season domain.Season, v domain.PlanVersion) (report.Table, error) {

	dash, err := s.analytics.Dashboard(r.Context(), service.DashboardRequest{
		SeasonID: season.ID, PlanVersionID: v.ID,
	})
	if err != nil {
		return t, err
	}
	t.Title = "Seasonal production summary"
	t.Columns = []report.Column{
		{Header: "Measure", Width: 30},
		{Header: "Target", Width: 16, Align: report.AlignRight, Numeric: true},
		{Header: "Actual", Width: 16, Align: report.AlignRight, Numeric: true},
		{Header: "Variance", Width: 16, Align: report.AlignRight, Numeric: true},
		{Header: "Unit", Width: 8},
	}
	add := func(label string, target, actual domain.Dec, unit string) {
		t.Rows = append(t.Rows, []report.Cell{
			report.Text(label), report.Num(target, 3), report.Num(actual, 3),
			report.Num(actual.Sub(target), 3), report.Text(unit),
		})
	}
	add("Cane crushed", dash.Cane.SeasonTarget, dash.Cane.CumulativeActual, "t")
	add("Raw sugar", dash.RawSugar.TargetTons, dash.RawSugar.ActualTons, "t")
	add("Raw sugar recovery", dash.RawSugar.TargetRecoveryPct, dash.RawSugar.ActualRecoveryPct, "%")
	for _, p := range dash.Products {
		add(p.ProductCode+" "+p.ProductName, p.TargetTons, p.ActualTons, "t")
	}
	t.Notes = append(t.Notes,
		fmt.Sprintf("Planned campaign %s to %s; forecast completion %s.",
			season.StartDate, season.EndDate, dash.Cane.ForecastEndDate),
		fmt.Sprintf("Downtime %s hours, estimated %s t of lost production.",
			dash.Downtime.Hours, dash.Downtime.LostTons))
	return t, nil
}

func (s *Server) reportApprovalHistory(r *http.Request, t report.Table, v domain.PlanVersion) (report.Table, error) {
	if err := requirePermission(r, domain.PermPlanRead); err != nil {
		return t, err
	}
	page, err := s.store.Audit().List(r.Context(), store.AuditFilter{
		Entity: "plan_version", EntityID: v.ID, Top: 1000,
	})
	if err != nil {
		return t, err
	}
	t.Title = "Plan version and approval history"
	t.Columns = []report.Column{
		{Header: "When (UTC)", Width: 20},
		{Header: "Action", Width: 14},
		{Header: "By", Width: 18},
		{Header: "Reason", Width: 40},
		{Header: "Correlation id", Width: 26},
	}
	for _, e := range page.Items {
		t.Rows = append(t.Rows, []report.Cell{
			report.Text(e.OccurredAt.Format("2006-01-02 15:04:05")),
			report.Text(e.Action), report.Text(e.Actor),
			report.Text(e.Reason), report.Text(e.CorrelationID),
		})
	}
	t.Notes = append(t.Notes, "Audit records are append only and cannot be amended by application users.")
	return t, nil
}

// ---------------------------------------------------------------------------
// Lookup helpers
// ---------------------------------------------------------------------------

func (s *Server) warehouseIndex(ctx context.Context) (map[string]string, map[string]domain.Dec, error) {
	page, err := s.store.MasterData().Warehouses().List(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return nil, nil, err
	}
	names := map[string]string{}
	usable := map[string]domain.Dec{}
	for _, w := range page.Items {
		names[w.ID] = w.Code + " " + w.Name
		usable[w.ID] = w.UsableCapacity()
	}
	return names, usable, nil
}

func (s *Server) productNames(ctx context.Context) (map[string]string, error) {
	page, err := s.store.MasterData().Products().List(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, p := range page.Items {
		out[p.ID] = p.Code
	}
	return out, nil
}

func sortedKeys[V any](m map[domain.BusinessDate]V) []domain.BusinessDate {
	out := make([]domain.BusinessDate, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
