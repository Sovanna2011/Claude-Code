package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// Analytics builds the dashboard figures. It reads plan and actual rows and
// runs them through the domain calculations; it stores nothing.
type Analytics struct {
	store    store.Store
	planning *Planning
}

// NewAnalytics builds the service.
func NewAnalytics(s store.Store, p *Planning) *Analytics {
	return &Analytics{store: s, planning: p}
}

// DashboardRequest selects what to report on.
type DashboardRequest struct {
	SeasonID string `json:"seasonId"`
	// PlanVersionID defaults to the season's released version.
	PlanVersionID string              `json:"planVersionId,omitempty"`
	From          domain.BusinessDate `json:"from,omitempty"`
	To            domain.BusinessDate `json:"to,omitempty"`
	// AsOf is the date the KPIs are calculated "as of"; it defaults to the last
	// day with a recorded actual, which is what a morning review wants.
	AsOf domain.BusinessDate `json:"asOf,omitempty"`
	// RollingWindow is the number of days in the rolling average, default 7.
	RollingWindow int `json:"rollingWindow,omitempty"`
}

// Dashboard is the executive overview payload.
type Dashboard struct {
	Season        domain.Season       `json:"season"`
	PlanVersion   domain.PlanVersion  `json:"planVersion"`
	ActualVersion domain.PlanVersion  `json:"actualVersion"`
	AsOf          domain.BusinessDate `json:"asOf"`

	Cane      CaneKPIs             `json:"cane"`
	RawSugar  RecoveryKPIs         `json:"rawSugar"`
	Products  []ProductKPI         `json:"products"`
	Storage   []StorageKPI         `json:"storage"`
	Shipments []ChannelKPI         `json:"shipments"`
	Downtime  DowntimeKPIs         `json:"downtime"`
	Alerts    []domain.Alert       `json:"alerts"`
	CaneTrend []domain.SeriesPoint `json:"caneTrend"`

	// RecoveryTrend is the recovery achieved each day against the assumption.
	// A season average inside the range can hide a fortnight outside it, which
	// is exactly the fortnight somebody needed to know about.
	//
	// It reuses SeriesPoint for its date, target and actual. The cumulative
	// fields are left at zero: a running total of percentages is not a figure
	// anybody wants, and inventing one would be worse than leaving it empty.
	RecoveryTrend []domain.SeriesPoint `json:"recoveryTrend"`
	// ProductTrend is the daily output of each finished product, for the mix.
	ProductTrend []NamedSeries `json:"productTrend"`
	// ShipmentTrend is the daily planned and actual shipment of each channel.
	ShipmentTrend []NamedSeries `json:"shipmentTrend"`
	// CaneRollingAvg is aligned index for index with CaneTrend.
	CaneRollingAvg []domain.Dec `json:"caneRollingAverage"`
}

// CaneKPIs is the crushing headline.
type CaneKPIs struct {
	SeasonTarget       domain.Dec          `json:"seasonTargetTons"`
	CumulativeTarget   domain.Dec          `json:"cumulativeTargetTons"`
	CumulativeActual   domain.Dec          `json:"cumulativeActualTons"`
	Remaining          domain.Dec          `json:"remainingTons"`
	AchievementPct     domain.Dec          `json:"achievementPct"`
	TodayTarget        domain.Dec          `json:"todayTargetTons"`
	TodayActual        domain.Dec          `json:"todayActualTons"`
	RollingAvgPerDay   domain.Dec          `json:"rollingAverageTonsPerDay"`
	PlannedEndDate     domain.BusinessDate `json:"plannedEndDate"`
	ForecastEndDate    domain.BusinessDate `json:"forecastEndDate"`
	ForecastDaysToGo   int                 `json:"forecastDaysToGo"`
	ForecastReliable   bool                `json:"forecastReliable"`
	DaysBehindSchedule int                 `json:"daysBehindSchedule"`
}

// RecoveryKPIs is the raw sugar headline.
type RecoveryKPIs struct {
	TargetTons        domain.Dec `json:"targetTons"`
	ActualTons        domain.Dec `json:"actualTons"`
	TargetRecoveryPct domain.Dec `json:"targetRecoveryPct"`
	ActualRecoveryPct domain.Dec `json:"actualRecoveryPct"`
	RecoveryVariance  domain.Dec `json:"recoveryVariancePct"`
	// MinRecoveryPct and MaxRecoveryPct are the operating window the verdict is
	// reached by. They are published so the recovery chart can draw the band the
	// alert is raised on, rather than a band of its own that would eventually
	// disagree with it.
	MinRecoveryPct domain.Dec      `json:"minRecoveryPct"`
	MaxRecoveryPct domain.Dec      `json:"maxRecoveryPct"`
	Verdict        domain.Severity `json:"verdict"`
}

// ProductKPI is one finished product's target versus actual.
type ProductKPI struct {
	ProductID      string     `json:"productId"`
	ProductCode    string     `json:"productCode"`
	ProductName    string     `json:"productName"`
	TargetTons     domain.Dec `json:"targetTons"`
	ActualTons     domain.Dec `json:"actualTons"`
	VarianceTons   domain.Dec `json:"varianceTons"`
	AchievementPct domain.Dec `json:"achievementPct"`
}

// StorageKPI is one warehouse's stock position and capacity risk.
type StorageKPI struct {
	WarehouseID     string              `json:"warehouseId"`
	WarehouseCode   string              `json:"warehouseCode"`
	WarehouseName   string              `json:"warehouseName"`
	StorageClass    domain.StorageClass `json:"storageClass"`
	CapacityTons    domain.Dec          `json:"capacityTons"`
	UsableTons      domain.Dec          `json:"usableCapacityTons"`
	BalanceTons     domain.Dec          `json:"balanceTons"`
	CapacityUsePct  domain.Dec          `json:"capacityUsePct"`
	FirstFullDate   domain.BusinessDate `json:"firstFullDate,omitempty"`
	FirstWarnDate   domain.BusinessDate `json:"firstWarningDate,omitempty"`
	RequiredShipTPD domain.Dec          `json:"requiredShipmentTonsPerDay"`
	Severity        domain.Severity     `json:"severity"`
}

// ChannelKPI is planned versus actual shipment for one channel.
type ChannelKPI struct {
	ChannelID    string     `json:"channelId"`
	ChannelCode  string     `json:"channelCode"`
	ChannelName  string     `json:"channelName"`
	PlannedTons  domain.Dec `json:"plannedTons"`
	ActualTons   domain.Dec `json:"actualTons"`
	VarianceTons domain.Dec `json:"varianceTons"`
}

// DowntimeKPIs summarises lost time.
type DowntimeKPIs struct {
	Hours      domain.Dec `json:"hours"`
	LostTons   domain.Dec `json:"lostTons"`
	EventCount int        `json:"eventCount"`
	// ByReason ranks the reasons by the hours they cost, worst first. Three
	// numbers about downtime tell somebody there is a problem; this tells them
	// which one to go and fix.
	ByReason []DowntimeReasonKPI `json:"byReason"`
}

// DowntimeReasonKPI is one bar of the Pareto: how much of the lost time one
// reason accounts for, and how much of it the reasons above it and this one
// account for together.
type DowntimeReasonKPI struct {
	ReasonCode  string     `json:"reasonCode"`
	ReasonName  string     `json:"reasonName"`
	Hours       domain.Dec `json:"hours"`
	LostTons    domain.Dec `json:"lostTons"`
	EventCount  int        `json:"eventCount"`
	SharePct    domain.Dec `json:"sharePct"`
	CumSharePct domain.Dec `json:"cumulativeSharePct"`
}

// NamedSeries is a daily curve that belongs to something: a product, a shipment
// channel. The product mix and the shipment trend are the same shape, so they
// are the same type rather than two that drift apart.
type NamedSeries struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	// Total is the sum of the actuals, which is what a stacked chart orders its
	// bands by so the largest contributor sits at the bottom.
	Total  domain.Dec           `json:"total"`
	Points []domain.SeriesPoint `json:"points"`
}

// Dashboard assembles the executive overview.
func (a *Analytics) Dashboard(ctx context.Context, req DashboardRequest) (Dashboard, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return Dashboard{}, err
	}

	season, err := a.planning.GetSeason(ctx, req.SeasonID)
	if err != nil {
		return Dashboard{}, err
	}
	planVersion, actualVersion, err := a.resolveVersions(ctx, season.ID, req.PlanVersionID)
	if err != nil {
		return Dashboard{}, err
	}

	window := req.RollingWindow
	if window <= 0 {
		window = 7
	}
	planFilter := store.PlanFilter{VersionIDs: []string{planVersion.ID}, From: req.From, To: req.To}
	actualFilter := store.PlanFilter{VersionIDs: []string{actualVersion.ID}, From: req.From, To: req.To}

	pl := a.store.Planning()
	planCane, err := pl.ListCane(ctx, withSeries(planFilter, domain.SeriesPlan))
	if err != nil {
		return Dashboard{}, err
	}
	actualCane, err := pl.ListCane(ctx, withSeries(actualFilter, domain.SeriesActual))
	if err != nil {
		return Dashboard{}, err
	}

	dash := Dashboard{
		Season: season, PlanVersion: planVersion, ActualVersion: actualVersion,
	}

	// --- cane ---------------------------------------------------------------
	dates := map[domain.BusinessDate]bool{}
	targetByDate := map[domain.BusinessDate]domain.Dec{}
	actualByDate := map[domain.BusinessDate]domain.Dec{}
	for _, r := range planCane {
		dates[r.BusinessDate] = true
		targetByDate[r.BusinessDate] = targetByDate[r.BusinessDate].Add(r.CaneCrushed)
	}
	for _, r := range actualCane {
		dates[r.BusinessDate] = true
		actualByDate[r.BusinessDate] = actualByDate[r.BusinessDate].Add(r.CaneCrushed)
	}
	ordered := sortedDates(dates)

	// The cane axis and the campaign axis are not the same axis, and building
	// every chart on the first one silently truncated the others.
	//
	// The mill stops crushing on 16 April and goes on refining and shipping
	// until 2 September. Taking the dates from the cane rows alone - which is
	// what this did - cut the product mix and the shipment curve off at the
	// last day of cane, so the chart showed 125,300 t of a 242,100 t plan while
	// the KPI beside it showed the whole thing. Two numbers for the same
	// quantity, on the same screen.
	campaignDates := map[domain.BusinessDate]bool{}
	for d := range dates {
		campaignDates[d] = true
	}

	series := domain.BuildSeries(ordered, targetByDate, actualByDate)
	rolling := domain.RollingAverage(series, window)
	dash.CaneTrend, dash.CaneRollingAvg = series, rolling

	asOf := req.AsOf
	if asOf == "" {
		asOf = lastActualDate(series)
	}
	dash.AsOf = asOf

	idx := indexOfDate(series, asOf)
	seasonTarget := domain.Zero
	if len(series) > 0 {
		seasonTarget = series[len(series)-1].CumTarget
	}
	dash.Cane.SeasonTarget = seasonTarget
	// The last day of cane, not the end of the campaign. The season now runs
	// months past the last cane - the refinery lives off the silo and the quota
	// keeps shipping - so comparing a crushing forecast against season.EndDate
	// says the mill is five months ahead of schedule and silences the alert
	// that says it is behind.
	dash.Cane.PlannedEndDate = ordered[len(ordered)-1]
	if idx >= 0 {
		point := series[idx]
		dash.Cane.CumulativeTarget = point.CumTarget
		dash.Cane.CumulativeActual = point.CumActual
		dash.Cane.TodayTarget = point.Target
		dash.Cane.TodayActual = point.Actual
		dash.Cane.AchievementPct = point.AchievementPct
		dash.Cane.Remaining = domain.Remaining(seasonTarget, point.CumActual)
		dash.Cane.RollingAvgPerDay = rolling[idx]

		if end, days, ok := domain.ForecastCompletion(asOf, dash.Cane.Remaining, rolling[idx]); ok {
			dash.Cane.ForecastEndDate, dash.Cane.ForecastDaysToGo, dash.Cane.ForecastReliable = end, days, true
			// Against the last planned day of cane. The season's own end date is
			// the end of the campaign, which the refinery and the shipping gate
			// run to long after the mill has stopped.
			if dash.Cane.PlannedEndDate != "" {
				dash.Cane.DaysBehindSchedule = dash.Cane.PlannedEndDate.DaysBetween(end)
			}
		}
	}

	// --- raw sugar and recovery --------------------------------------------
	products, err := a.productIndex(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	planProd, err := pl.ListProducts(ctx, withSeries(planFilter, domain.SeriesPlan))
	if err != nil {
		return Dashboard{}, err
	}
	actualProd, err := pl.ListProducts(ctx, withSeries(actualFilter, domain.SeriesActual))
	if err != nil {
		return Dashboard{}, err
	}

	assumptions, err := a.assumptionMap(ctx, planVersion.ID)
	if err != nil {
		return Dashboard{}, err
	}
	dash.RawSugar = a.recoveryKPIs(assumptions, planCane, actualCane, planProd, actualProd, products)
	dash.Products = productKPIs(planProd, actualProd, products)
	for _, r := range planProd {
		campaignDates[r.BusinessDate] = true
	}
	for _, r := range actualProd {
		campaignDates[r.BusinessDate] = true
	}
	campaign := sortedDates(campaignDates)

	dash.RecoveryTrend = recoveryTrend(ordered, actualCane, actualProd, products,
		assumptions[domain.AsmRecoveryPct])
	dash.ProductTrend = productTrend(campaign, planProd, actualProd, products)

	// --- storage and capacity ----------------------------------------------
	dash.Storage, dash.Alerts, err = a.storageKPIs(ctx, planVersion, actualVersion, assumptions, req)
	if err != nil {
		return Dashboard{}, err
	}

	// --- shipments ----------------------------------------------------------
	dash.Shipments, dash.ShipmentTrend, err = a.shipmentKPIs(ctx, planFilter, actualFilter, campaign)
	if err != nil {
		return Dashboard{}, err
	}

	// --- downtime -----------------------------------------------------------
	dash.Downtime, err = a.downtimeKPIs(ctx, season, req)
	if err != nil {
		return Dashboard{}, err
	}

	// --- cross-cutting alerts ----------------------------------------------
	dash.Alerts = append(dash.Alerts, scheduleAlerts(dash.Cane, season)...)
	if dash.RawSugar.Verdict == domain.SeverityWarning || dash.RawSugar.Verdict == domain.SeverityError {
		dash.Alerts = append(dash.Alerts, domain.Alert{
			Code: "RECOVERY_OUT_OF_RANGE", Severity: dash.RawSugar.Verdict,
			Title: "Raw sugar recovery is outside the expected range",
			Detail: fmt.Sprintf("Actual recovery %s%% against a target of %s%%",
				dash.RawSugar.ActualRecoveryPct, dash.RawSugar.TargetRecoveryPct),
			Entity: "season", EntityID: season.ID, Date: asOf,
		})
	}
	sortAlerts(dash.Alerts)
	return dash, nil
}

// Versions resolves the pair a comparison needs: the plan version to measure
// against and the season's actuals container.
//
// It is exported because the reports need the same answer the dashboard gets. A
// report that resolved "the current plan" by its own rule would eventually
// disagree with the KPI beside it, and the first person to notice would be the
// one holding both printouts.
func (a *Analytics) Versions(ctx context.Context, seasonID, requested string) (plan, actual domain.PlanVersion, err error) {
	return a.resolveVersions(ctx, seasonID, requested)
}

// resolveVersions picks the plan version to report against and the season's
// actuals container.
func (a *Analytics) resolveVersions(ctx context.Context, seasonID, requested string) (plan, actual domain.PlanVersion, err error) {
	page, err := a.store.Planning().ListVersions(ctx, seasonID, store.ListOptions{Top: 1000})
	if err != nil {
		return plan, actual, err
	}
	for _, v := range page.Items {
		if v.PlanType == domain.PlanTypeActual {
			actual = v
			continue
		}
		if requested != "" && v.ID == requested {
			plan = v
		}
		if requested == "" && v.Status == domain.StatusReleased {
			plan = v
		}
	}
	if plan.ID == "" && requested == "" {
		// No released baseline yet: fall back to the most recent plan version so
		// the dashboard is useful during preparation.
		for _, v := range page.Items {
			if v.PlanType != domain.PlanTypeActual && v.VersionNo >= plan.VersionNo {
				plan = v
			}
		}
	}
	if plan.ID == "" {
		return plan, actual, fmt.Errorf("%w: the season has no plan version to report on", domain.ErrNotFound)
	}
	if actual.ID == "" {
		return plan, actual, fmt.Errorf("%w: the season has no actuals container", domain.ErrNotFound)
	}
	return plan, actual, nil
}

func (a *Analytics) assumptionMap(ctx context.Context, versionID string) (map[string]domain.Dec, error) {
	rows, err := a.store.Planning().ListAssumptions(ctx, versionID)
	if err != nil {
		return nil, err
	}
	out := map[string]domain.Dec{}
	for _, r := range rows {
		out[r.Code] = r.Value
	}
	return out, nil
}

func (a *Analytics) productIndex(ctx context.Context) (map[string]domain.Product, error) {
	page, err := a.store.MasterData().Products().List(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return nil, err
	}
	out := map[string]domain.Product{}
	for _, p := range page.Items {
		out[p.ID] = p
	}
	return out, nil
}

// recoveryKPIs compares raw sugar output with cane crushed.
func (a *Analytics) recoveryKPIs(assumptions map[string]domain.Dec,
	planCane, actualCane []domain.DailyCanePlan,
	planProd, actualProd []domain.DailyProductPlan,
	products map[string]domain.Product) RecoveryKPIs {

	planCrushed, actualCrushed := domain.Zero, domain.Zero
	for _, r := range planCane {
		planCrushed = planCrushed.Add(r.CaneCrushed)
	}
	for _, r := range actualCane {
		actualCrushed = actualCrushed.Add(r.CaneCrushed)
	}

	targetRecovery := assumptions[domain.AsmRecoveryPct]
	k := RecoveryKPIs{
		TargetRecoveryPct: targetRecovery,
		TargetTons:        domain.ExpectedRawSugar(planCrushed, targetRecovery),
	}
	// Actual raw sugar is what was produced against raw-class products.
	for _, r := range actualProd {
		if p, ok := products[r.ProductID]; ok && p.StorageClass == domain.StorageRaw {
			k.ActualTons = k.ActualTons.Add(r.Quantity)
		}
	}
	k.ActualRecoveryPct = domain.ActualRecoveryPct(k.ActualTons, actualCrushed)
	k.RecoveryVariance = domain.RoundPct(k.ActualRecoveryPct.Sub(targetRecovery))

	minPct, maxPct := assumptions[domain.AsmRecoveryMinPct], assumptions[domain.AsmRecoveryMaxPct]
	if minPct.IsZero() && maxPct.IsZero() {
		// No configured window: judge against the target with a one point band.
		minPct = targetRecovery.Sub(domain.DI(1))
		maxPct = targetRecovery.Add(domain.DI(1))
	}
	k.MinRecoveryPct, k.MaxRecoveryPct = minPct, maxPct
	if actualCrushed.IsZero() {
		k.Verdict = domain.SeverityInfo // the season has not started
	} else {
		k.Verdict = domain.RecoveryVerdict(k.ActualRecoveryPct, minPct, maxPct)
	}
	return k
}

func productKPIs(plan, actual []domain.DailyProductPlan, products map[string]domain.Product) []ProductKPI {
	targets := map[string]domain.Dec{}
	actuals := map[string]domain.Dec{}
	for _, r := range plan {
		targets[r.ProductID] = targets[r.ProductID].Add(r.Quantity)
	}
	for _, r := range actual {
		actuals[r.ProductID] = actuals[r.ProductID].Add(r.Quantity)
	}
	ids := map[string]bool{}
	for id := range targets {
		ids[id] = true
	}
	for id := range actuals {
		ids[id] = true
	}

	out := make([]ProductKPI, 0, len(ids))
	for id := range ids {
		p := products[id]
		k := ProductKPI{
			ProductID: id, ProductCode: p.Code, ProductName: p.Name,
			TargetTons: domain.RoundQty(targets[id]), ActualTons: domain.RoundQty(actuals[id]),
		}
		k.VarianceTons = domain.Variance(k.ActualTons, k.TargetTons)
		k.AchievementPct = domain.AchievementPct(k.ActualTons, k.TargetTons)
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProductCode < out[j].ProductCode })
	return out
}

// recoveryTrend is the recovery achieved each day, against the assumption as a
// flat target line.
//
// It is built from the actuals alone: the plan's recovery is the assumption by
// construction, so plotting it would draw the target line twice. A day with
// cane crushed but no raw sugar recorded yet carries no actual rather than a
// recovery of zero, which would drag the eye to a problem that is really a
// laboratory still working.
func recoveryTrend(dates []domain.BusinessDate, cane []domain.DailyCanePlan,
	produced []domain.DailyProductPlan, products map[string]domain.Product,
	targetPct domain.Dec) []domain.SeriesPoint {

	crushed := map[domain.BusinessDate]domain.Dec{}
	for _, r := range cane {
		crushed[r.BusinessDate] = crushed[r.BusinessDate].Add(r.CaneCrushed)
	}
	raw := map[domain.BusinessDate]domain.Dec{}
	for _, r := range produced {
		if products[r.ProductID].Stage != domain.StageRawSugar {
			continue
		}
		raw[r.BusinessDate] = raw[r.BusinessDate].Add(r.Quantity)
	}

	out := make([]domain.SeriesPoint, 0, len(dates))
	for _, d := range dates {
		point := domain.SeriesPoint{Date: d, Target: domain.RoundPct(targetPct)}
		c, r := crushed[d], raw[d]
		if !c.IsZero() && !r.IsZero() {
			point.Actual = domain.ActualRecoveryPct(r, c)
			point.Variance = domain.RoundPct(point.Actual.Sub(point.Target))
			point.HasActual = true
		}
		out = append(out, point)
	}
	return out
}

// productTrend is the daily output of each finished product, for the mix chart.
//
// Bulk raw sugar is left out: a stacked chart of finished goods with raw sugar
// in it is a chart of two different things, and the raw sugar band would be
// twice the height of everything else.
func productTrend(dates []domain.BusinessDate, plan, actual []domain.DailyProductPlan,
	products map[string]domain.Product) []NamedSeries {

	target := map[string]map[domain.BusinessDate]domain.Dec{}
	got := map[string]map[domain.BusinessDate]domain.Dec{}
	add := func(m map[string]map[domain.BusinessDate]domain.Dec, r domain.DailyProductPlan) {
		if !products[r.ProductID].IsFinished {
			return
		}
		if m[r.ProductID] == nil {
			m[r.ProductID] = map[domain.BusinessDate]domain.Dec{}
		}
		m[r.ProductID][r.BusinessDate] = m[r.ProductID][r.BusinessDate].Add(r.Quantity)
	}
	for _, r := range plan {
		add(target, r)
	}
	for _, r := range actual {
		add(got, r)
	}

	ids := map[string]bool{}
	for id := range target {
		ids[id] = true
	}
	for id := range got {
		ids[id] = true
	}

	out := make([]NamedSeries, 0, len(ids))
	for id := range ids {
		p := products[id]
		s := NamedSeries{ID: id, Code: p.Code, Name: p.Name,
			Points: domain.BuildSeries(dates, target[id], got[id])}
		if n := len(s.Points); n > 0 {
			s.Total = s.Points[n-1].CumActual
		}
		out = append(out, s)
	}
	// Largest first, so the stacked bands are ordered by how much they matter
	// rather than alphabetically.
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Total.Equal(out[j].Total) {
			return out[i].Total.GreaterThan(out[j].Total)
		}
		return out[i].Code < out[j].Code
	})
	return out
}

// shipmentTrend is the daily planned and actual shipment of each channel.
func shipmentTrend(dates []domain.BusinessDate, plan, actual []domain.DailyShipmentPlan,
	channels map[string]domain.ShipmentChannel) []NamedSeries {

	target := map[string]map[domain.BusinessDate]domain.Dec{}
	got := map[string]map[domain.BusinessDate]domain.Dec{}
	add := func(m map[string]map[domain.BusinessDate]domain.Dec, r domain.DailyShipmentPlan) {
		if m[r.ChannelID] == nil {
			m[r.ChannelID] = map[domain.BusinessDate]domain.Dec{}
		}
		m[r.ChannelID][r.BusinessDate] = m[r.ChannelID][r.BusinessDate].Add(r.Quantity)
	}
	for _, r := range plan {
		add(target, r)
	}
	for _, r := range actual {
		add(got, r)
	}

	ids := map[string]bool{}
	for id := range target {
		ids[id] = true
	}
	for id := range got {
		ids[id] = true
	}

	out := make([]NamedSeries, 0, len(ids))
	for id := range ids {
		c := channels[id]
		s := NamedSeries{ID: id, Code: c.Code, Name: c.Name,
			Points: domain.BuildSeries(dates, target[id], got[id])}
		if n := len(s.Points); n > 0 {
			s.Total = s.Points[n-1].CumActual
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Total.Equal(out[j].Total) {
			return out[i].Total.GreaterThan(out[j].Total)
		}
		return out[i].Code < out[j].Code
	})
	return out
}

// storageKPIs replays each warehouse ledger and works out the capacity risk.
func (a *Analytics) storageKPIs(ctx context.Context, planVersion, actualVersion domain.PlanVersion,
	assumptions map[string]domain.Dec, req DashboardRequest) ([]StorageKPI, []domain.Alert, error) {

	warehouses, err := a.store.MasterData().Warehouses().List(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return nil, nil, err
	}
	rows, err := a.store.Planning().ListStorage(ctx, store.PlanFilter{
		VersionIDs: []string{planVersion.ID}, From: req.From, To: req.To, Series: domain.SeriesPlan,
	})
	if err != nil {
		return nil, nil, err
	}

	warnPct := assumptions[domain.AsmCapacityWarnPct]
	if warnPct.IsZero() {
		warnPct = domain.DI(80)
	}
	alertPct := assumptions[domain.AsmCapacityAlertPct]
	if alertPct.IsZero() {
		alertPct = domain.DI(90)
	}

	// Aggregate movements per warehouse across all products, because capacity
	// is shared by everything stored in the building.
	byWarehouse := map[string]map[domain.BusinessDate]*domain.LedgerMovement{}
	for _, r := range rows {
		days, ok := byWarehouse[r.WarehouseID]
		if !ok {
			days = map[domain.BusinessDate]*domain.LedgerMovement{}
			byWarehouse[r.WarehouseID] = days
		}
		m, ok := days[r.BusinessDate]
		if !ok {
			m = &domain.LedgerMovement{Date: r.BusinessDate}
			days[r.BusinessDate] = m
		}
		m.ProductionReceipt = m.ProductionReceipt.Add(r.ProductionReceipt)
		m.TransferIn = m.TransferIn.Add(r.TransferIn)
		m.TransferOut = m.TransferOut.Add(r.TransferOut)
		m.RepackIn = m.RepackIn.Add(r.RepackIn)
		m.RepackOut = m.RepackOut.Add(r.RepackOut)
		m.RemeltIssue = m.RemeltIssue.Add(r.RemeltIssue)
		m.ShipmentQty = m.ShipmentQty.Add(r.ShipmentQty)
		m.Adjustment = m.Adjustment.Add(r.Adjustment)
		m.ProcessLoss = m.ProcessLoss.Add(r.ProcessLoss)
		m.HoldQty = m.HoldQty.Add(r.HoldQty)
	}

	var out []StorageKPI
	var alerts []domain.Alert
	for _, wh := range warehouses.Items {
		days, ok := byWarehouse[wh.ID]
		if !ok {
			continue
		}
		dateSet := map[domain.BusinessDate]bool{}
		for d := range days {
			dateSet[d] = true
		}
		ordered := sortedDates(dateSet)
		movements := make([]domain.LedgerMovement, len(ordered))
		netReceipts := make([]domain.Dec, len(ordered))
		for i, d := range ordered {
			movements[i] = *days[d]
			// Net inflow excluding the shipment we may be solving for.
			netReceipts[i] = domain.SumDec(movements[i].ProductionReceipt, movements[i].TransferIn,
				movements[i].RepackIn).
				Sub(domain.SumDec(movements[i].RemeltIssue, movements[i].TransferOut,
					movements[i].RepackOut, movements[i].ProcessLoss))
		}

		usable := wh.UsableCapacity()
		ledger := domain.RollLedger(wh.OpeningBalance, usable, movements)

		k := StorageKPI{
			WarehouseID: wh.ID, WarehouseCode: wh.Code, WarehouseName: wh.Name,
			StorageClass: wh.StorageClass, CapacityTons: wh.CapacityTons, UsableTons: usable,
			Severity: domain.SeveritySuccess,
		}
		if len(ledger) > 0 {
			last := ledger[len(ledger)-1]
			k.BalanceTons, k.CapacityUsePct = last.EndingBalance, last.CapacityUsePct
		}
		if d, ok := domain.FirstBreach(ledger, usable.Mul(warnPct).Div(domain.DI(100))); ok {
			k.FirstWarnDate, k.Severity = d, domain.SeverityWarning
		}
		if d, ok := domain.FirstBreach(ledger, usable); ok {
			k.FirstFullDate, k.Severity = d, domain.SeverityError
			alerts = append(alerts, domain.Alert{
				Code: "CAPACITY_EXCEEDED", Severity: domain.SeverityError,
				Title:  fmt.Sprintf("%s runs out of space", wh.Name),
				Detail: fmt.Sprintf("The plan fills %s t of usable capacity on %s", usable, d),
				Entity: "warehouse", EntityID: wh.ID, Date: d,
			})
		} else if k.FirstWarnDate != "" {
			alerts = append(alerts, domain.Alert{
				Code: "CAPACITY_WARNING", Severity: domain.SeverityWarning,
				Title:  fmt.Sprintf("%s passes %s%% of capacity", wh.Name, warnPct),
				Detail: fmt.Sprintf("Planned stock crosses the warning threshold on %s", k.FirstWarnDate),
				Entity: "warehouse", EntityID: wh.ID, Date: k.FirstWarnDate,
			})
		}
		// What it would take to stay inside the alert threshold.
		limit := usable.Mul(alertPct).Div(domain.DI(100))
		k.RequiredShipTPD = domain.RequiredDailyShipment(wh.OpeningBalance, limit, netReceipts)
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].WarehouseCode < out[j].WarehouseCode })
	return out, alerts, nil
}

func (a *Analytics) shipmentKPIs(ctx context.Context, planFilter, actualFilter store.PlanFilter,
	dates []domain.BusinessDate) ([]ChannelKPI, []NamedSeries, error) {

	channels, err := a.store.MasterData().Channels().List(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return nil, nil, err
	}
	plan, err := a.store.Planning().ListShipments(ctx, withSeries(planFilter, domain.SeriesPlan))
	if err != nil {
		return nil, nil, err
	}
	actual, err := a.store.Planning().ListShipments(ctx, withSeries(actualFilter, domain.SeriesActual))
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[string]domain.ShipmentChannel, len(channels.Items))
	for _, c := range channels.Items {
		byID[c.ID] = c
	}

	planned := map[string]domain.Dec{}
	actuals := map[string]domain.Dec{}
	for _, r := range plan {
		planned[r.ChannelID] = planned[r.ChannelID].Add(r.Quantity)
	}
	for _, r := range actual {
		actuals[r.ChannelID] = actuals[r.ChannelID].Add(r.Quantity)
	}

	var out []ChannelKPI
	for _, c := range channels.Items {
		p, aq := planned[c.ID], actuals[c.ID]
		if p.IsZero() && aq.IsZero() {
			continue
		}
		out = append(out, ChannelKPI{
			ChannelID: c.ID, ChannelCode: c.Code, ChannelName: c.Name,
			PlannedTons: domain.RoundQty(p), ActualTons: domain.RoundQty(aq),
			VarianceTons: domain.Variance(aq, p),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ChannelCode < out[j].ChannelCode })
	return out, shipmentTrend(dates, plan, actual, byID), nil
}

func (a *Analytics) downtimeKPIs(ctx context.Context, season domain.Season, req DashboardRequest) (DowntimeKPIs, error) {
	events, err := a.store.Planning().ListDowntime(ctx, store.PlanFilter{
		FactoryID: season.FactoryID, From: req.From, To: req.To,
	})
	if err != nil {
		return DowntimeKPIs{}, err
	}
	lines, err := a.store.MasterData().Lines().List(ctx, store.ListOptions{
		Top: 1000, ParentID: season.FactoryID,
	})
	if err != nil {
		return DowntimeKPIs{}, err
	}
	rated := map[string]domain.Dec{}
	for _, l := range lines.Items {
		rated[l.ID] = l.RatedTPH
	}

	reasons, err := a.store.MasterData().ReasonCodes().List(ctx, store.ListOptions{Top: 1000})
	if err != nil {
		return DowntimeKPIs{}, err
	}
	named := map[string]string{}
	for _, rc := range reasons.Items {
		named[rc.Code] = rc.Name
	}

	k := DowntimeKPIs{EventCount: len(events)}
	byReason := map[string]*DowntimeReasonKPI{}
	for _, e := range events {
		lost := domain.LostTons(e.DurationHrs, rated[e.LineID])
		k.Hours = k.Hours.Add(e.DurationHrs)
		k.LostTons = k.LostTons.Add(lost)

		code := e.ReasonCode
		if code == "" {
			// A stoppage recorded against no reason is still lost time, and
			// bucketing it silently into another reason would misdirect the
			// person reading the chart.
			code = "(none)"
		}
		bucket, ok := byReason[code]
		if !ok {
			bucket = &DowntimeReasonKPI{ReasonCode: code, ReasonName: named[code]}
			byReason[code] = bucket
		}
		bucket.Hours = bucket.Hours.Add(e.DurationHrs)
		bucket.LostTons = bucket.LostTons.Add(lost)
		bucket.EventCount++
	}
	k.Hours, k.LostTons = domain.RoundRate(k.Hours), domain.RoundQty(k.LostTons)

	k.ByReason = make([]DowntimeReasonKPI, 0, len(byReason))
	for _, b := range byReason {
		b.Hours, b.LostTons = domain.RoundRate(b.Hours), domain.RoundQty(b.LostTons)
		b.SharePct = domain.RoundPct(domain.SafePct(b.Hours, k.Hours))
		k.ByReason = append(k.ByReason, *b)
	}
	// Worst first: a Pareto that is not sorted is a bar chart.
	sort.Slice(k.ByReason, func(i, j int) bool {
		if !k.ByReason[i].Hours.Equal(k.ByReason[j].Hours) {
			return k.ByReason[i].Hours.GreaterThan(k.ByReason[j].Hours)
		}
		return k.ByReason[i].ReasonCode < k.ByReason[j].ReasonCode
	})
	// The cumulative share is calculated from the running hours, not by adding
	// up the rounded percentages. Four reasons at 62.069, 24.138, 6.897 and
	// 6.897 sum to 100.001, and a Pareto whose line finishes past 100 % is
	// visibly wrong to the one person in the room who checks.
	running := domain.Zero
	for i := range k.ByReason {
		running = running.Add(k.ByReason[i].Hours)
		k.ByReason[i].CumSharePct = domain.RoundPct(domain.SafePct(running, k.Hours))
	}
	return k, nil
}

// scheduleAlerts warns when the campaign is drifting past its planned end.
// scheduleAlerts compares the crushing forecast against the last planned day of
// cane. Against the end of the campaign it would never fire: the refinery runs
// until September whatever the mill is doing.
func scheduleAlerts(cane CaneKPIs, season domain.Season) []domain.Alert {
	if !cane.ForecastReliable || cane.PlannedEndDate == "" || cane.DaysBehindSchedule <= 0 {
		return nil
	}
	severity := domain.SeverityWarning
	if cane.DaysBehindSchedule > 7 {
		severity = domain.SeverityError
	}
	return []domain.Alert{{
		Code: "CRUSHING_BEHIND_SCHEDULE", Severity: severity,
		Title: fmt.Sprintf("Crushing is forecast to finish %d days late", cane.DaysBehindSchedule),
		Detail: fmt.Sprintf(
			"At the current rolling average of %s t/day the remaining %s t finishes on %s, against a planned end of %s",
			cane.RollingAvgPerDay, cane.Remaining, cane.ForecastEndDate, cane.PlannedEndDate),
		Entity: "season", EntityID: season.ID, Date: cane.ForecastEndDate,
	}}
}

func sortAlerts(alerts []domain.Alert) {
	rank := map[domain.Severity]int{
		domain.SeverityError: 0, domain.SeverityWarning: 1,
		domain.SeverityInfo: 2, domain.SeveritySuccess: 3,
	}
	sort.SliceStable(alerts, func(i, j int) bool {
		if rank[alerts[i].Severity] != rank[alerts[j].Severity] {
			return rank[alerts[i].Severity] < rank[alerts[j].Severity]
		}
		return alerts[i].Date < alerts[j].Date
	})
}

func withSeries(f store.PlanFilter, s domain.Series) store.PlanFilter {
	f.Series = s
	return f
}

func lastActualDate(series []domain.SeriesPoint) domain.BusinessDate {
	for i := len(series) - 1; i >= 0; i-- {
		if series[i].HasActual {
			return series[i].Date
		}
	}
	if len(series) > 0 {
		return series[0].Date
	}
	return ""
}

func indexOfDate(series []domain.SeriesPoint, date domain.BusinessDate) int {
	for i, p := range series {
		if p.Date == date {
			return i
		}
	}
	return -1
}
