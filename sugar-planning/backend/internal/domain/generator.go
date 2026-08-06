package domain

import (
	"fmt"
	"sort"
)

// GeneratorInput is everything the plan generator needs. It is deliberately a
// plain value: the generator performs no I/O, so a planner can run the same
// calculation in a what-if scenario without touching stored data.
type GeneratorInput struct {
	Season      Season
	Version     PlanVersion
	Assumptions map[string]Dec
	Mix         []ProductMixEntry
	Products    map[string]Product   // by product id
	Warehouses  map[string]Warehouse // by warehouse id
	// RawWarehouseIDs are the raw sugar stores that receive the sugar not sent
	// straight to refining. Receipts are shared between them in proportion to
	// usable capacity, so they fill at the same rate.
	RawWarehouseIDs []string
	// RawProductID is the product code used for raw sugar stock.
	RawProductID string
	// QuotaChannelID receives the generated shipment plan.
	QuotaChannelID string
	// NonWorkingDays are dates excluded from crushing (maintenance windows,
	// approved shutdowns). Cane and production targets skip these days.
	NonWorkingDays map[BusinessDate]bool
}

// GeneratorOutput is a complete daily plan for one version.
type GeneratorOutput struct {
	Dates     []BusinessDate
	Cane      []DailyCanePlan
	Products  []DailyProductPlan
	Storage   []DailyStoragePlan
	Shipments []DailyShipmentPlan
	Summary   PlanSummary
	Warnings  []Alert
}

// PlanSummary is the season roll-up the generator reports back, used both by
// the UI confirmation dialog and by the reconciliation tests.
type PlanSummary struct {
	CaneTarget       Dec `json:"caneTargetTons"`
	CaneAllocated    Dec `json:"caneAllocatedTons"`
	RawSugarExpected Dec `json:"rawSugarExpectedTons"`
	RawDirectRefine  Dec `json:"rawDirectToRefiningTons"`
	RawToStorage     Dec `json:"rawToStorageTons"`
	FinishedGoods    Dec `json:"finishedGoodsTons"`
	RemeltInput      Dec `json:"remeltInputTons"`
	ShipmentPlanned  Dec `json:"shipmentPlannedTons"`
	WorkingDays      int `json:"workingDays"`
}

// requiredAssumptions must be present before a plan can be generated.
var requiredAssumptions = []string{
	AsmCaneTarget, AsmSeasonDays, AsmRecoveryPct, AsmDirectToRefinePct,
	AsmRemeltInputFactor, AsmQuotaShipmentTPD,
}

// Generate builds the daily cane, production, storage and shipment plan for a
// version from its assumptions and product mix.
//
// The chain it implements is the one described in section 5 to 9 of the
// specification:
//
//	cane target -> daily cane -> raw sugar (recovery) -> split direct/storage
//	           -> finished goods by mix -> remelt draw -> stock ledgers
//	           -> shipment at the quota rate
//
// Every rate it uses comes from Assumptions or from master data; nothing is
// hard-coded here.
func Generate(in GeneratorInput) (*GeneratorOutput, error) {
	verr := &ValidationError{}
	for _, code := range requiredAssumptions {
		if _, ok := in.Assumptions[code]; !ok {
			verr.Add("assumptions."+code, "ASSUMPTION_MISSING",
				fmt.Sprintf("assumption %s is required to generate a plan", code))
		}
	}
	if in.Season.StartDate == "" || !in.Season.StartDate.Valid() {
		verr.Add("season.startDate", "INVALID_DATE", "the season needs a valid start date")
	}
	if verr.HasErrors() {
		return nil, verr
	}

	caneTarget := in.Assumptions[AsmCaneTarget]
	seasonDays := int(in.Assumptions[AsmSeasonDays].IntPart())
	recoveryPct := in.Assumptions[AsmRecoveryPct]
	directPct := in.Assumptions[AsmDirectToRefinePct]
	remeltFactor := in.Assumptions[AsmRemeltInputFactor]
	quotaTPD := in.Assumptions[AsmQuotaShipmentTPD]

	if seasonDays <= 0 {
		verr.Add("assumptions."+AsmSeasonDays, "OUT_OF_RANGE", "the season must be at least one day long")
	}
	if caneTarget.IsNegative() {
		verr.Add("assumptions."+AsmCaneTarget, "NEGATIVE", "the cane target cannot be negative")
	}
	if recoveryPct.IsNegative() || recoveryPct.GreaterThan(DI(100)) {
		verr.Add("assumptions."+AsmRecoveryPct, "OUT_OF_RANGE", "recovery must be between 0 and 100 percent")
	}
	if directPct.IsNegative() || directPct.GreaterThan(DI(100)) {
		verr.Add("assumptions."+AsmDirectToRefinePct, "OUT_OF_RANGE",
			"the share sent directly to refining must be between 0 and 100 percent")
	}
	if verr.HasErrors() {
		return nil, verr
	}

	out := &GeneratorOutput{}

	// --- Calendar -----------------------------------------------------------
	// Working days are calendar days minus configured non-working days. The
	// campaign is extended until `seasonDays` working days are found, so a
	// maintenance window pushes the end of the season out rather than silently
	// reducing the tonnage.
	var working []BusinessDate
	d := in.Season.StartDate
	for len(working) < seasonDays {
		if !in.NonWorkingDays[d] {
			working = append(working, d)
		}
		d = d.AddDays(1)
		if len(working)+len(in.NonWorkingDays) > 3650 { // guard against a pathological calendar
			return nil, fmt.Errorf("%w: the calendar excludes too many days to fit the season", ErrValidation)
		}
	}
	out.Dates = working
	out.Summary.WorkingDays = len(working)

	// --- Cane and raw sugar -------------------------------------------------
	caneDaily := AllocateEvenly(caneTarget, len(working))
	crushRate := in.Assumptions[AsmCrushRateTPH]
	availHrs := in.Assumptions[AsmAvailableHours]
	if availHrs.IsZero() {
		availHrs = DI(24)
	}

	// Raw sugar is derived from the cane series with cumulative rounding, so the
	// daily figures still read as "cane x recovery" and the season total is
	// exactly 2,300,000 x 11.00 % = 253,000 t rather than 252,999.955 t.
	rawDaily := ScaleSeries(caneDaily, recoveryPct.Div(DI(100)))

	for i, day := range working {
		rate := crushRate
		if rate.IsZero() {
			rate = RoundRate(SafeDiv(caneDaily[i], availHrs))
		}
		out.Cane = append(out.Cane, DailyCanePlan{
			VersionID:     in.Version.ID,
			FactoryID:     in.Season.FactoryID,
			BusinessDate:  day,
			Series:        SeriesPlan,
			CaneAvailable: caneDaily[i],
			CaneDelivered: caneDaily[i],
			CaneAccepted:  caneDaily[i],
			CaneCrushed:   caneDaily[i],
			CrushRateTPH:  rate,
			AvailableHrs:  availHrs,
		})
		out.Summary.CaneAllocated = out.Summary.CaneAllocated.Add(caneDaily[i])
		out.Summary.RawSugarExpected = out.Summary.RawSugarExpected.Add(rawDaily[i])
	}
	out.Summary.CaneTarget = caneTarget

	// --- Finished goods by product mix -------------------------------------
	// Each mix entry is spread over the campaign, either evenly or at its own
	// daily rate. Production is indexed by day so the ledgers can consume it.
	type mixPlan struct {
		entry  ProductMixEntry
		daily  []Dec
		remelt []Dec // raw sugar input needed for this entry's output
	}
	mixPlans := make([]mixPlan, 0, len(in.Mix))
	for _, m := range in.Mix {
		var daily []Dec
		if m.DailyRateTons.GreaterThan(Zero) {
			var short Dec
			daily, short = AllocateAtRate(m.SeasonTons, m.DailyRateTons, len(working))
			if short.GreaterThan(Zero) {
				out.Warnings = append(out.Warnings, Alert{
					Code:     "MIX_RATE_TOO_LOW",
					Severity: SeverityWarning,
					Title:    "Planned daily rate cannot deliver the season tonnage",
					Detail: fmt.Sprintf("%s t of product %s cannot be produced at %s t/day within %d days; %s t is unplanned",
						m.SeasonTons, productName(in.Products, m.ProductID), m.DailyRateTons, len(working), short),
					Entity:   "product_mix",
					EntityID: m.ID,
				})
			}
		} else {
			daily = AllocateEvenly(m.SeasonTons, len(working))
		}
		mixPlans = append(mixPlans, mixPlan{
			entry: m, daily: daily, remelt: ScaleSeries(daily, remeltFactor),
		})
		out.Summary.FinishedGoods = out.Summary.FinishedGoods.Add(m.SeasonTons)
	}

	finishedPerDay := make([]Dec, len(working))
	remeltNeedPerDay := make([]Dec, len(working))
	for i, day := range working {
		for _, mp := range mixPlans {
			qty := mp.daily[i]
			if qty.IsZero() {
				continue
			}
			finishedPerDay[i] = finishedPerDay[i].Add(qty)
			remeltNeedPerDay[i] = remeltNeedPerDay[i].Add(mp.remelt[i])
			out.Products = append(out.Products, DailyProductPlan{
				VersionID:    in.Version.ID,
				FactoryID:    in.Season.FactoryID,
				LineID:       mp.entry.LineID,
				BusinessDate: day,
				ProductID:    mp.entry.ProductID,
				PackagingID:  mp.entry.PackagingID,
				Series:       SeriesPlan,
				Quantity:     qty,
				RemeltInput:  mp.remelt[i],
			})
		}
		out.Summary.RemeltInput = out.Summary.RemeltInput.Add(remeltNeedPerDay[i])
	}

	// --- Raw sugar split and remelt draw ------------------------------------
	// Raw sugar leaves the boiling house on one of two routes: straight to the
	// refinery, or into raw storage. The planned direct share is capped by what
	// the refinery actually needs that day, and anything refining still needs
	// beyond that is drawn from storage - but never more than the stock that
	// exists, because a plan that empties a silo below zero is not a plan.
	plannedDirect := ScaleSeries(rawDaily, directPct.Div(DI(100)))
	rawDirect := make([]Dec, len(working))
	rawToStore := make([]Dec, len(working))
	remeltFromStore := make([]Dec, len(working))

	rawOpening := Zero
	for _, id := range in.RawWarehouseIDs {
		rawOpening = rawOpening.Add(in.Warehouses[id].OpeningBalance)
	}
	rawStock := rawOpening
	supplyShort, firstShortDate := Zero, BusinessDate("")

	for i, day := range working {
		need := remeltNeedPerDay[i]
		direct := MinDec(plannedDirect[i], need)
		toStore := RoundQty(rawDaily[i].Sub(direct))
		available := rawStock.Add(toStore)
		draw := MinDec(ClampNonNegative(need.Sub(direct)), ClampNonNegative(available))

		if unmet := need.Sub(direct).Sub(draw); unmet.GreaterThan(Zero) {
			supplyShort = supplyShort.Add(unmet)
			if firstShortDate == "" {
				firstShortDate = day
			}
		}

		rawDirect[i] = direct
		rawToStore[i] = toStore
		remeltFromStore[i] = draw
		rawStock = available.Sub(draw)

		out.Summary.RawDirectRefine = out.Summary.RawDirectRefine.Add(direct)
		out.Summary.RawToStorage = out.Summary.RawToStorage.Add(toStore)
	}

	if supplyShort.GreaterThan(Zero) {
		out.Warnings = append(out.Warnings, Alert{
			Code:     "REMELT_SUPPLY_SHORT",
			Severity: SeverityError,
			Title:    "Refining demand exceeds the planned raw sugar supply",
			Detail: fmt.Sprintf(
				"The finished goods mix needs %s t of raw sugar at a factor of %s, but the plan produces only %s t. "+
					"The shortfall of %s t first appears on %s. Reduce the finished goods mix, raise the cane target "+
					"or the recovery assumption, or plan an opening stock of raw sugar.",
				RoundQty(out.Summary.RemeltInput), remeltFactor, RoundQty(out.Summary.RawSugarExpected),
				RoundQty(supplyShort), firstShortDate),
			Entity: "plan_version",
			Date:   firstShortDate,
		})
	}

	// --- Raw sugar stock ledger --------------------------------------------
	rawIDs := append([]string(nil), in.RawWarehouseIDs...)
	sort.Strings(rawIDs)
	if len(rawIDs) > 0 && in.RawProductID != "" {
		weights := make([]Dec, len(rawIDs))
		for i, id := range rawIDs {
			weights[i] = in.Warehouses[id].UsableCapacity()
		}
		for i, day := range working {
			recvSplit := AllocateProportional(rawToStore[i], weights)
			issueSplit := AllocateProportional(remeltFromStore[i], weights)
			for j, id := range rawIDs {
				out.Storage = append(out.Storage, DailyStoragePlan{
					VersionID:         in.Version.ID,
					WarehouseID:       id,
					ProductID:         in.RawProductID,
					BusinessDate:      day,
					Series:            SeriesPlan,
					ProductionReceipt: recvSplit[j],
					RemeltIssue:       issueSplit[j],
				})
			}
		}
	}

	// --- Finished goods ledger and shipment plan ----------------------------
	// Shipment runs at the planned quota rate, shared between products in
	// proportion to what was produced that day, and never ships more than the
	// stock on hand.
	fgStock := map[string]Dec{} // warehouse|product -> running balance
	for i, day := range working {
		weights := make([]Dec, len(mixPlans))
		for j, mp := range mixPlans {
			weights[j] = mp.daily[i]
		}
		shipSplit := AllocateProportional(MinDec(quotaTPD, finishedPerDay[i]), weights)

		for j, mp := range mixPlans {
			wh := mp.entry.WarehouseID
			if wh == "" {
				continue
			}
			key := wh + "|" + mp.entry.ProductID
			ship := MinDec(shipSplit[j], fgStock[key].Add(mp.daily[i]))
			ship = ClampNonNegative(ship)
			fgStock[key] = fgStock[key].Add(mp.daily[i]).Sub(ship)

			out.Storage = append(out.Storage, DailyStoragePlan{
				VersionID:         in.Version.ID,
				WarehouseID:       wh,
				ProductID:         mp.entry.ProductID,
				BusinessDate:      day,
				Series:            SeriesPlan,
				ProductionReceipt: mp.daily[i],
				ShipmentQty:       ship,
			})
			if ship.GreaterThan(Zero) && in.QuotaChannelID != "" {
				out.Shipments = append(out.Shipments, DailyShipmentPlan{
					VersionID:    in.Version.ID,
					WarehouseID:  wh,
					ProductID:    mp.entry.ProductID,
					ChannelID:    in.QuotaChannelID,
					BusinessDate: day,
					Series:       SeriesPlan,
					Quantity:     ship,
				})
				out.Summary.ShipmentPlanned = out.Summary.ShipmentPlanned.Add(ship)
			}
		}
	}

	out.Warnings = append(out.Warnings, capacityWarnings(in, out)...)
	return out, nil
}

// capacityWarnings replays the generated ledgers through the capacity rules and
// reports the first day each store crosses its warning and alert thresholds.
func capacityWarnings(in GeneratorInput, out *GeneratorOutput) []Alert {
	warnPct := in.Assumptions[AsmCapacityWarnPct]
	if warnPct.IsZero() {
		warnPct = DI(80)
	}
	alertPct := in.Assumptions[AsmCapacityAlertPct]
	if alertPct.IsZero() {
		alertPct = DI(90)
	}

	byStore := map[string][]LedgerMovement{}
	for _, s := range out.Storage {
		key := s.WarehouseID + "|" + s.ProductID
		byStore[key] = append(byStore[key], LedgerMovement{
			Date:              s.BusinessDate,
			ProductionReceipt: s.ProductionReceipt,
			TransferIn:        s.TransferIn,
			RepackIn:          s.RepackIn,
			Adjustment:        s.Adjustment,
			RemeltIssue:       s.RemeltIssue,
			ShipmentQty:       s.ShipmentQty,
			TransferOut:       s.TransferOut,
			RepackOut:         s.RepackOut,
			ProcessLoss:       s.ProcessLoss,
		})
	}

	keys := make([]string, 0, len(byStore))
	for k := range byStore {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var alerts []Alert
	for _, key := range keys {
		movements := byStore[key]
		sort.Slice(movements, func(a, b int) bool { return movements[a].Date < movements[b].Date })
		whID := key[:indexOrLen(key, '|')]
		wh, ok := in.Warehouses[whID]
		if !ok {
			continue
		}
		usable := wh.UsableCapacity()
		if usable.LessThanOrEqual(Zero) {
			continue
		}
		days := RollLedger(wh.OpeningBalance, usable, movements)

		for _, thr := range []struct {
			pct      Dec
			severity Severity
			label    string
		}{
			{warnPct, SeverityWarning, "warning"},
			{alertPct, SeverityError, "critical"},
			{DI(100), SeverityError, "full"},
		} {
			limit := usable.Mul(thr.pct).Div(DI(100))
			if date, breached := FirstBreach(days, limit); breached {
				alerts = append(alerts, Alert{
					Code:     "CAPACITY_" + thr.label,
					Severity: thr.severity,
					Title:    fmt.Sprintf("%s reaches %s%% of usable capacity", wh.Name, thr.pct),
					Detail: fmt.Sprintf("Planned stock crosses %s t (%s%% of %s t usable) on %s",
						RoundQty(limit), thr.pct, usable, date),
					Entity:   "warehouse",
					EntityID: wh.ID,
					Date:     date,
				})
			}
		}
	}
	return alerts
}

func indexOrLen(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return len(s)
}

func productName(products map[string]Product, id string) string {
	if p, ok := products[id]; ok {
		return p.Name
	}
	return id
}
