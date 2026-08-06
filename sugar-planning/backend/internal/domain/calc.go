package domain

// This file is the executable form of the calculation catalogue documented in
// docs/06-calculation-catalogue.md. Every exported function here corresponds
// to a numbered formula in that document; the doc comments repeat the formula
// so that the two cannot drift apart unnoticed.

// ---------------------------------------------------------------------------
// C1-C7  Target versus actual series
// ---------------------------------------------------------------------------

// SeriesPoint is one day of a target-versus-actual curve, with the cumulative
// figures the planners work with.
type SeriesPoint struct {
	Date           BusinessDate `json:"date"`
	Target         Dec          `json:"target"`
	Actual         Dec          `json:"actual"`
	CumTarget      Dec          `json:"cumTarget"`
	CumActual      Dec          `json:"cumActual"`
	Variance       Dec          `json:"variance"`
	CumVariance    Dec          `json:"cumVariance"`
	AchievementPct Dec          `json:"achievementPct"`
	HasActual      bool         `json:"hasActual"`
}

// BuildSeries walks the dates in order and produces the cumulative curve.
//
//	C1  cumulative target = previous cumulative target + daily target
//	C2  cumulative actual = previous cumulative actual + daily actual
//	C3  daily variance    = actual - target
//	C4  achievement %     = 100 * cumulative actual / cumulative target
//
// Days without a recorded actual contribute nothing to the cumulative actual
// and are flagged, so a dashboard can stop the actual curve at "today" instead
// of drawing it flat to the end of the season.
func BuildSeries(dates []BusinessDate, target, actual map[BusinessDate]Dec) []SeriesPoint {
	out := make([]SeriesPoint, 0, len(dates))
	cumT, cumA := Zero, Zero
	for _, d := range dates {
		t := target[d]
		a, has := actual[d]
		cumT = cumT.Add(t)
		if has {
			cumA = cumA.Add(a)
		}
		p := SeriesPoint{
			Date:        d,
			Target:      RoundQty(t),
			Actual:      RoundQty(a),
			CumTarget:   RoundQty(cumT),
			CumActual:   RoundQty(cumA),
			Variance:    RoundQty(a.Sub(t)),
			CumVariance: RoundQty(cumA.Sub(cumT)),
			HasActual:   has,
		}
		if has {
			p.AchievementPct = RoundPct(SafePct(cumA, cumT))
		}
		out = append(out, p)
	}
	return out
}

// Variance is C3: actual minus target. Positive means ahead of plan.
func Variance(actual, target Dec) Dec { return RoundQty(actual.Sub(target)) }

// AchievementPct is C4: 100 * actual / target, zero-safe.
func AchievementPct(actual, target Dec) Dec { return RoundPct(SafePct(actual, target)) }

// Remaining is C5: season target minus cumulative actual, floored at zero.
func Remaining(seasonTarget, cumActual Dec) Dec {
	return RoundQty(ClampNonNegative(seasonTarget.Sub(cumActual)))
}

// RollingAverage is C6: the trailing average over the last `window` values.
// Element i averages elements [i-window+1 .. i], using however many exist.
// Only days flagged as having an actual are counted, so a season that has not
// started does not drag the average to zero.
func RollingAverage(points []SeriesPoint, window int) []Dec {
	out := make([]Dec, len(points))
	for i := range points {
		sum, n := Zero, 0
		for j := i; j >= 0 && j > i-window; j-- {
			if points[j].HasActual {
				sum = sum.Add(points[j].Actual)
				n++
			}
		}
		if n == 0 {
			out[i] = Zero
			continue
		}
		out[i] = RoundQty(sum.Div(DI(int64(n))))
	}
	return out
}

// ForecastCompletion is C7: the date on which the remaining quantity is
// exhausted at the given daily rate.
//
//	days remaining  = ceil(remaining / daily rate)
//	completion date = from + days remaining
//
// It returns ok=false when the rate is zero or negative, because in that case
// the campaign never completes and reporting a date would be misleading.
func ForecastCompletion(from BusinessDate, remaining, dailyRate Dec) (date BusinessDate, days int, ok bool) {
	if dailyRate.LessThanOrEqual(Zero) {
		return "", 0, false
	}
	if remaining.LessThanOrEqual(Zero) {
		return from, 0, true
	}
	days = int(CeilInt(remaining.Div(dailyRate)))
	return from.AddDays(days), days, true
}

// ---------------------------------------------------------------------------
// C8-C12  Cane crushing and raw sugar recovery
// ---------------------------------------------------------------------------

// EffectiveCrushCapacity is C8: rate * (available hours - stoppage hours).
func EffectiveCrushCapacity(ratePerHour, availableHours, stoppageHours Dec) Dec {
	return RoundQty(ratePerHour.Mul(ClampNonNegative(availableHours.Sub(stoppageHours))))
}

// UtilisationPct is C9: 100 * run hours / available hours.
func UtilisationPct(availableHours, stoppageHours Dec) Dec {
	return RoundPct(SafePct(ClampNonNegative(availableHours.Sub(stoppageHours)), availableHours))
}

// ExpectedRawSugar is C10: cane crushed * recovery %, with the recovery given
// as a percentage (11.00 means eleven percent).
//
//	2,300,000 t x 11.00 % = 253,000 t
func ExpectedRawSugar(caneCrushed, recoveryPct Dec) Dec {
	return RoundQty(caneCrushed.Mul(recoveryPct).Div(DI(100)))
}

// ActualRecoveryPct is C11: 100 * raw sugar produced / cane crushed, zero-safe.
func ActualRecoveryPct(rawSugar, caneCrushed Dec) Dec {
	return RoundPct(SafePct(rawSugar, caneCrushed))
}

// RecoveryVerdict is C12: classify a recovery figure against the configured
// operating window. Outside the window the planner gets a warning; a negative
// or absurd value is an error.
func RecoveryVerdict(recoveryPct, minPct, maxPct Dec) Severity {
	switch {
	case recoveryPct.IsNegative() || recoveryPct.GreaterThan(DI(100)):
		return SeverityError
	case recoveryPct.LessThan(minPct) || recoveryPct.GreaterThan(maxPct):
		return SeverityWarning
	default:
		return SeveritySuccess
	}
}

// ---------------------------------------------------------------------------
// C13-C18  Stock ledger, capacity and shipment rates
// ---------------------------------------------------------------------------

// LedgerMovement is one day of movements for a product in one warehouse,
// before the balances are rolled forward.
type LedgerMovement struct {
	Date              BusinessDate
	ProductionReceipt Dec
	TransferIn        Dec
	RepackIn          Dec
	Adjustment        Dec // signed: positive corrects up, negative corrects down
	RemeltIssue       Dec
	ShipmentQty       Dec
	TransferOut       Dec
	RepackOut         Dec
	ProcessLoss       Dec
	HoldQty           Dec // quantity blocked by quality, part of the balance
	PhysicalBalance   *Dec
}

// LedgerDay is one rolled-forward day of the stock ledger.
type LedgerDay struct {
	Date             BusinessDate `json:"date"`
	BeginningBalance Dec          `json:"beginningBalance"`
	Receipts         Dec          `json:"receipts"`
	Issues           Dec          `json:"issues"`
	EndingBalance    Dec          `json:"endingBalance"`
	AvailableBalance Dec          `json:"availableBalance"`
	CapacityUsePct   Dec          `json:"capacityUsePct"`
	Reconciliation   *Dec         `json:"reconciliationDiff,omitempty"`
}

// RollLedger is C13-C16.
//
//	C13  beginning balance = previous day's ending balance
//	C14  ending balance    = beginning + production + transfer in + repack in
//	                         + adjustment - remelt issue - shipment
//	                         - transfer out - repack out - process loss
//	C15  available balance = ending balance - quantity on quality hold
//	C16  capacity use %    = 100 * ending balance / usable capacity
//
// The reconciliation difference (calculated ending balance minus the recorded
// physical balance) is filled in only for days that carry a physical count.
// Movements must be supplied in date order.
func RollLedger(opening Dec, usableCapacity Dec, movements []LedgerMovement) []LedgerDay {
	out := make([]LedgerDay, 0, len(movements))
	balance := opening
	for _, m := range movements {
		receipts := SumDec(m.ProductionReceipt, m.TransferIn, m.RepackIn)
		issues := SumDec(m.RemeltIssue, m.ShipmentQty, m.TransferOut, m.RepackOut, m.ProcessLoss)
		beginning := balance
		ending := beginning.Add(receipts).Add(m.Adjustment).Sub(issues)
		day := LedgerDay{
			Date:             m.Date,
			BeginningBalance: RoundQty(beginning),
			Receipts:         RoundQty(receipts),
			Issues:           RoundQty(issues),
			EndingBalance:    RoundQty(ending),
			AvailableBalance: RoundQty(ending.Sub(m.HoldQty)),
			CapacityUsePct:   RoundPct(SafePct(ending, usableCapacity)),
		}
		if m.PhysicalBalance != nil {
			diff := RoundQty(ending.Sub(*m.PhysicalBalance))
			day.Reconciliation = &diff
		}
		out = append(out, day)
		balance = ending
	}
	return out
}

// CapacityUsePct is C16 on its own, for single point-in-time checks.
func CapacityUsePct(balance, usableCapacity Dec) Dec {
	return RoundPct(SafePct(balance, usableCapacity))
}

// FirstBreach is C17: the first day on which the ending balance reaches or
// exceeds the given limit in tons. Callers pass usable capacity times the
// threshold (for example 90 %) as the limit.
func FirstBreach(days []LedgerDay, limitTons Dec) (BusinessDate, bool) {
	for _, d := range days {
		if d.EndingBalance.GreaterThanOrEqual(limitTons) {
			return d.Date, true
		}
	}
	return "", false
}

// RequiredDailyShipment is C18: the smallest constant daily shipment rate that
// keeps the stock at or below limitTons on every day of the horizon.
//
// For each day i (1-based) the balance without shipment is
//
//	B(i) = opening + sum of receipts up to i
//
// and with a constant rate s the balance is B(i) - i*s, so the constraint
// B(i) - i*s <= limit gives
//
//	s >= (B(i) - limit) / i      for every i
//
// and the answer is the largest of those lower bounds, floored at zero.
// Receipts are the net inflow per day (production plus transfers in, less any
// issues other than the shipment being solved for).
func RequiredDailyShipment(opening, limitTons Dec, dailyNetReceipts []Dec) Dec {
	required := Zero
	running := opening
	for i, r := range dailyNetReceipts {
		running = running.Add(r)
		excess := running.Sub(limitTons)
		if excess.LessThanOrEqual(Zero) {
			continue
		}
		bound := excess.Div(DI(int64(i + 1)))
		required = MaxDec(required, bound)
	}
	return RoundRate(required)
}

// ShipmentToClearBy is C19: the constant daily shipment needed to bring the
// stock down to targetBalance by the end of the horizon.
//
//	rate = (opening + total receipts - target balance) / number of days
//
// A non-positive result means no shipment is needed and zero is returned.
func ShipmentToClearBy(opening, targetBalance Dec, dailyNetReceipts []Dec, days int) Dec {
	if days <= 0 {
		return Zero
	}
	total := opening
	for _, r := range dailyNetReceipts {
		total = total.Add(r)
	}
	rate := total.Sub(targetBalance).Div(DI(int64(days)))
	return RoundRate(ClampNonNegative(rate))
}

// MassBalanceDiff is C20: calculated ending balance minus recorded physical
// balance.
func MassBalanceDiff(calculated, physical Dec) Dec { return RoundQty(calculated.Sub(physical)) }

// WithinTolerance is C21: |difference| <= tolerance % of the reference base.
// The base is normally the throughput of the period, not the closing balance,
// so that a small warehouse with heavy turnover is not judged too harshly.
func WithinTolerance(difference, base, tolerancePct Dec) bool {
	limit := base.Abs().Mul(tolerancePct).Div(DI(100))
	return difference.Abs().LessThanOrEqual(limit)
}

// ---------------------------------------------------------------------------
// C22-C25  Conversions and packaging material requirements
// ---------------------------------------------------------------------------

// TonsToUnits is C22: how many packages of the given net weight are needed for
// a tonnage.
//
//	units = tons * 1000 / net weight in kg
//
// The result is not rounded here; RequiredPackages applies scrap and rounds up.
func TonsToUnits(tons, netWeightKg Dec) Dec {
	if netWeightKg.LessThanOrEqual(Zero) {
		return Zero
	}
	return tons.Mul(DI(1000)).Div(netWeightKg)
}

// RequiredPackages is C23: packages needed including a scrap allowance,
// rounded up because a fraction of a bag cannot be issued.
//
//	packages = ceil(tons * 1000 / net weight kg * (1 + scrap %/100))
func RequiredPackages(tons, netWeightKg, scrapPct Dec) int64 {
	gross := TonsToUnits(tons, netWeightKg)
	if gross.LessThanOrEqual(Zero) {
		return 0
	}
	factor := DI(1).Add(scrapPct.Div(DI(100)))
	return CeilInt(gross.Mul(factor))
}

// PurchaseRequirement is C24:
//
//	purchase = max(0, gross requirement + safety stock - on hand - on order)
func PurchaseRequirement(gross, safetyStock, onHand, onOrder Dec) Dec {
	return RoundQty(ClampNonNegative(gross.Add(safetyStock).Sub(onHand).Sub(onOrder)))
}

// SuggestedOrderDate is C25: required-by date shifted back by the lead time.
func SuggestedOrderDate(requiredBy BusinessDate, leadTimeDays int) BusinessDate {
	return requiredBy.AddDays(-leadTimeDays)
}

// ConvertQty is C26: apply a conversion factor, quantity_to = quantity_from *
// factor.
func ConvertQty(qty, factor Dec) Dec { return RoundQty(qty.Mul(factor)) }

// RemeltInput is C27: the raw sugar input needed to produce a finished goods
// output, using the effective-dated input factor (for example 1.05).
//
//	raw sugar input = finished output * factor
func RemeltInput(finishedOutput, factor Dec) Dec {
	return RoundQty(finishedOutput.Mul(factor))
}

// ---------------------------------------------------------------------------
// C31-C33  Allocation of a season total across days
// ---------------------------------------------------------------------------

// AllocateEvenly is C31: split a season total across n days so that the parts
// sum back to the total exactly.
//
// Dividing and rounding each day independently loses tons: 2,300,000 / 137 =
// 16,788.321167..., and 137 days of 16,788.321 add up to 2,299,999.977. This
// function instead rounds the running cumulative figure and takes the
// difference, so the daily values differ by at most one unit in the last place
// and the season total is preserved to the cent.
func AllocateEvenly(total Dec, n int) []Dec {
	if n <= 0 {
		return nil
	}
	out := make([]Dec, n)
	prevCum := Zero
	for i := 1; i <= n; i++ {
		cum := RoundQty(total.Mul(DI(int64(i))).Div(DI(int64(n))))
		out[i-1] = cum.Sub(prevCum)
		prevCum = cum
	}
	return out
}

// AllocateProportional is C32: split a total across n buckets in proportion to
// the given weights, preserving the total exactly. Zero total weight falls back
// to an even split.
func AllocateProportional(total Dec, weights []Dec) []Dec {
	n := len(weights)
	if n == 0 {
		return nil
	}
	sum := SumDec(weights...)
	if sum.LessThanOrEqual(Zero) {
		return AllocateEvenly(total, n)
	}
	out := make([]Dec, n)
	cumWeight, prevCum := Zero, Zero
	for i, w := range weights {
		cumWeight = cumWeight.Add(w)
		cum := RoundQty(total.Mul(cumWeight).Div(sum))
		out[i] = cum.Sub(prevCum)
		prevCum = cum
	}
	return out
}

// ScaleSeries is C34: multiply a daily series by a factor while preserving the
// exact rounded total.
//
// Rounding each day independently and adding up gives a different answer from
// rounding the season total, which shows up in reconciliation as tens of
// kilograms of phantom sugar. Scaling the running cumulative and taking
// differences keeps every daily value within one unit in the last place of the
// naive result while guaranteeing
//
//	sum(ScaleSeries(daily, f)) == round(sum(daily) * f)
func ScaleSeries(daily []Dec, factor Dec) []Dec {
	out := make([]Dec, len(daily))
	cumIn, prevCumOut := Zero, Zero
	for i, v := range daily {
		cumIn = cumIn.Add(v)
		cumOut := RoundQty(cumIn.Mul(factor))
		out[i] = cumOut.Sub(prevCumOut)
		prevCumOut = cumOut
	}
	return out
}

// AllocateAtRate is C33: run at a fixed daily rate from day one until the total
// is used up, leaving later days at zero. The final working day carries the
// remainder, so the parts still sum to the total. If n days at the given rate
// cannot cover the total, everything that fits is allocated and the shortfall
// is returned.
func AllocateAtRate(total, dailyRate Dec, n int) (parts []Dec, shortfall Dec) {
	if n <= 0 {
		return nil, total
	}
	if dailyRate.LessThanOrEqual(Zero) {
		return AllocateEvenly(total, n), Zero
	}
	parts = make([]Dec, n)
	remaining := total
	for i := 0; i < n; i++ {
		if remaining.LessThanOrEqual(Zero) {
			parts[i] = Zero
			continue
		}
		day := MinDec(dailyRate, remaining)
		parts[i] = RoundQty(day)
		remaining = remaining.Sub(parts[i])
	}
	return parts, ClampNonNegative(remaining)
}

// ---------------------------------------------------------------------------
// C28-C30  Downtime and capacity impact
// ---------------------------------------------------------------------------

// AvailabilityPct is C28: 100 * run time / available time.
func AvailabilityPct(availableHours, downtimeHours Dec) Dec {
	return RoundPct(SafePct(ClampNonNegative(availableHours.Sub(downtimeHours)), availableHours))
}

// LostTons is C29: downtime hours * rated throughput.
func LostTons(downtimeHours, ratedTPH Dec) Dec {
	return RoundQty(downtimeHours.Mul(ratedTPH))
}

// ThroughputPct is C30: 100 * actual throughput / rated throughput, where
// actual throughput is tons produced divided by run hours.
func ThroughputPct(tons, runHours, ratedTPH Dec) Dec {
	if runHours.LessThanOrEqual(Zero) {
		return Zero
	}
	return RoundPct(SafePct(tons.Div(runHours), ratedTPH))
}
