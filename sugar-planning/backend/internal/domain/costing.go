package domain

import (
	"fmt"
	"sort"
)

// This file is the costing model of section 21: what a ton of sugar costs, how
// that splits by cost element, and why the actual differs from the plan.
//
// The whole of it rests on one idea. A cost is a **rate** multiplied by a
// **driver quantity** - dollars per ton of cane crushed, dollars per hour the
// mill ran, dollars per day whether it ran or not. Keeping the two apart is
// what makes the variance decomposable: when the cost moves, it moved because
// the rate changed, or because the quantity changed, and a controller needs to
// know which.

// ---------------------------------------------------------------------------
// Cost elements and their drivers
// ---------------------------------------------------------------------------

// CostDriver says what a cost element scales with.
type CostDriver string

const (
	// DriverCaneTon is per ton of cane crushed: cane payment, harvesting,
	// haulage, milling chemicals.
	DriverCaneTon CostDriver = "CANE_TON"
	// DriverSugarTon is per ton of finished sugar: refining chemicals,
	// packaging, storage handling.
	DriverSugarTon CostDriver = "SUGAR_TON"
	// DriverRunHour is per hour of crushing: energy, water, direct labour on a
	// shift pattern.
	DriverRunHour CostDriver = "RUN_HOUR"
	// DriverCalendarDay is per day of the campaign whether the mill ran or not:
	// salaried staff, depreciation, insurance, security.
	DriverCalendarDay CostDriver = "CALENDAR_DAY"
	// DriverFixedSeason is a lump sum for the whole season: the annual
	// maintenance shutdown, a licence fee.
	DriverFixedSeason CostDriver = "FIXED_SEASON"
)

// ValidDriver reports whether the value is a known driver.
func ValidDriver(d CostDriver) bool {
	switch d {
	case DriverCaneTon, DriverSugarTon, DriverRunHour, DriverCalendarDay, DriverFixedSeason:
		return true
	}
	return false
}

// DriverUnit names the unit a driver is measured in, for a screen or a report
// header.
func DriverUnit(d CostDriver) string {
	switch d {
	case DriverCaneTon, DriverSugarTon:
		return "t"
	case DriverRunHour:
		return "h"
	case DriverCalendarDay:
		return "d"
	case DriverFixedSeason:
		return "season"
	}
	return ""
}

// CostCategory groups elements for the profit and loss view.
type CostCategory string

const (
	CategoryCane        CostCategory = "CANE"
	CategoryLabour      CostCategory = "LABOUR"
	CategoryEnergy      CostCategory = "ENERGY"
	CategoryChemicals   CostCategory = "CHEMICALS"
	CategoryPackaging   CostCategory = "PACKAGING"
	CategoryMaintenance CostCategory = "MAINTENANCE"
	CategoryOverhead    CostCategory = "OVERHEAD"
)

// ValidCostCategory reports whether the value is a known category.
func ValidCostCategory(c CostCategory) bool {
	switch c {
	case CategoryCane, CategoryLabour, CategoryEnergy, CategoryChemicals,
		CategoryPackaging, CategoryMaintenance, CategoryOverhead:
		return true
	}
	return false
}

// CostElement is one line of the cost structure: cane payment, boiler fuel,
// lime, jumbo bags, mill maintenance, and so on.
//
// Variable is not derived from the driver, because it is a different question.
// A cost per calendar day is fixed; a cost per ton is variable; a cost per run
// hour is usually variable but a site with a salaried shift crew may class it
// as fixed, and the contribution analysis has to follow the site's view rather
// than the software's.
type CostElement struct {
	ID       string       `json:"id"`
	Code     string       `json:"code"`
	Name     string       `json:"name"`
	Category CostCategory `json:"category"`
	Driver   CostDriver   `json:"driver"`
	Variable bool         `json:"variable"`
	Note     string       `json:"note,omitempty"`
	Validity
	AuditFields
}

// ---------------------------------------------------------------------------
// Rates
// ---------------------------------------------------------------------------

// RateType separates what a cost was budgeted at from what it turned out to be.
type RateType string

const (
	// RateStandard is the planned rate, set before the season and held still so
	// that a variance means something.
	RateStandard RateType = "STANDARD"
	// RateActual is what was really paid, entered as invoices arrive.
	RateActual RateType = "ACTUAL"
)

// ValidRateType reports whether the value is a known rate type.
func ValidRateType(t RateType) bool { return t == RateStandard || t == RateActual }

// CostRate is the money per unit of driver, effective dated so that a mid-season
// fuel price rise does not rewrite the cost of the weeks before it.
type CostRate struct {
	ID        string       `json:"id"`
	ElementID string       `json:"elementId"`
	FactoryID string       `json:"factoryId"`
	SeasonID  string       `json:"seasonId,omitempty"`
	RateType  RateType     `json:"rateType"`
	Rate      Dec          `json:"rate"`
	Currency  string       `json:"currency"`
	ValidFrom BusinessDate `json:"validFrom"`
	ValidTo   BusinessDate `json:"validTo,omitempty"`
	Note      string       `json:"note,omitempty"`
	AuditFields
}

// IsEffectiveOn reports whether the rate applies on a date.
func (r CostRate) IsEffectiveOn(on BusinessDate) bool {
	if r.ValidFrom != "" && on < r.ValidFrom {
		return false
	}
	if r.ValidTo != "" && on > r.ValidTo {
		return false
	}
	return true
}

// RateOn picks the rate in force on a date.
//
// Where two rates for the same element and type are both effective - a price
// change entered without ending the previous one - the later start date wins,
// for the same reason a quality specification does: it is the one somebody most
// recently decided on, and leaving the choice to the order rows came back in
// would make a cost depend on the storage engine.
func RateOn(rates []CostRate, elementID string, rateType RateType, on BusinessDate) (CostRate, bool) {
	var best CostRate
	found := false
	for _, r := range rates {
		if r.ElementID != elementID || r.RateType != rateType || !r.IsEffectiveOn(on) {
			continue
		}
		if !found || r.ValidFrom > best.ValidFrom {
			best, found = r, true
		}
	}
	return best, found
}

// ExchangeRate converts one currency into another on a date. Cane is paid in
// riel, fuel is invoiced in dollars, and the board reports in one of them.
type ExchangeRate struct {
	ID           string       `json:"id"`
	FromCurrency string       `json:"fromCurrency"`
	ToCurrency   string       `json:"toCurrency"`
	Rate         Dec          `json:"rate"`
	ValidFrom    BusinessDate `json:"validFrom"`
	AuditFields
}

// Convert is C42: a money amount expressed in another currency.
//
//	converted = amount × rate(from → to)
//
// A conversion to the same currency is the identity and needs no rate, which
// keeps a single-currency site from having to configure anything at all.
func Convert(amount Dec, from, to string, rates []ExchangeRate, on BusinessDate) (Dec, error) {
	if from == to || from == "" || to == "" {
		return amount, nil
	}
	var best ExchangeRate
	found := false
	for _, r := range rates {
		if r.FromCurrency != from || r.ToCurrency != to {
			continue
		}
		if r.ValidFrom != "" && on < r.ValidFrom {
			continue
		}
		if !found || r.ValidFrom > best.ValidFrom {
			best, found = r, true
		}
	}
	if found {
		return RoundMoney(amount.Mul(best.Rate)), nil
	}
	// An inverse quotation is as good as a direct one, and a site that has
	// entered USD→KHR should not have to enter KHR→USD as well.
	for _, r := range rates {
		if r.FromCurrency != to || r.ToCurrency != from || r.Rate.LessThanOrEqual(Zero) {
			continue
		}
		if r.ValidFrom != "" && on < r.ValidFrom {
			continue
		}
		if !found || r.ValidFrom > best.ValidFrom {
			best, found = r, true
		}
	}
	if found {
		return RoundMoney(SafeDiv(amount, best.Rate)), nil
	}
	return Zero, fmt.Errorf("%w: no exchange rate from %s to %s in force on %s",
		ErrValidation, from, to, on)
}

// ---------------------------------------------------------------------------
// C35-C41  The cost calculation
// ---------------------------------------------------------------------------

// DriverQuantities are the measured volumes a cost run multiplies rates by.
// They come from the plan for a planned run and from the actuals for an actual
// one; the arithmetic that follows does not know which.
type DriverQuantities struct {
	CaneTons     Dec
	SugarTons    Dec
	RunHours     Dec
	CalendarDays Dec
}

// Quantity is C35: the driver quantity an element scales with.
//
// A fixed seasonal cost has a driver quantity of one, so that the same
// rate × quantity arithmetic covers it without a special case.
func (d DriverQuantities) Quantity(driver CostDriver) Dec {
	switch driver {
	case DriverCaneTon:
		return d.CaneTons
	case DriverSugarTon:
		return d.SugarTons
	case DriverRunHour:
		return d.RunHours
	case DriverCalendarDay:
		return d.CalendarDays
	case DriverFixedSeason:
		return DI(1)
	}
	return Zero
}

// ElementCost is C36:
//
//	cost = rate × driver quantity
func ElementCost(rate, quantity Dec) Dec { return RoundMoney(rate.Mul(quantity)) }

// UnitCost is C38: cost per ton of finished sugar.
//
//	unit cost = total cost / output tons
//
// Zero output gives zero rather than an error: at the start of a season the
// cost is real and the output is not yet, and a dashboard should say "0.00"
// rather than fail.
func UnitCost(totalCost, outputTons Dec) Dec {
	return RoundMoney(SafeDiv(totalCost, outputTons))
}

// RateVariance is C39: the part of the difference caused by paying a different
// price.
//
//	rate variance = (actual rate - standard rate) × actual quantity
//
// Positive is unfavourable: it cost more than the plan allowed.
func RateVariance(actualRate, standardRate, actualQuantity Dec) Dec {
	return RoundMoney(actualRate.Sub(standardRate).Mul(actualQuantity))
}

// UsageVariance is C40: the part caused by using a different quantity.
//
//	usage variance = (actual quantity - planned quantity) × standard rate
//
// The standard rate is deliberately the one used here. Valuing the extra
// quantity at the actual rate would count the price difference twice, and the
// two variances would no longer add up to the total.
//
// This is the definition. A cost run does not call it directly: it rounds the
// rate variance and takes the usage variance as the remainder, so that the two
// always reconstruct the total exactly. See SplitVariance.
func UsageVariance(actualQuantity, plannedQuantity, standardRate Dec) Dec {
	return RoundMoney(actualQuantity.Sub(plannedQuantity).Mul(standardRate))
}

// SplitVariance divides a total variance into its rate and usage halves so that
// the two always add back up to it.
//
// The identity is exact before rounding:
//
//	actual − planned = (actual rate − standard rate) × actual qty
//	                 + (actual qty − planned qty) × standard rate
//
// Rounding each half to the cent independently breaks it: two values rounded
// separately need not sum to the rounded total, and a report whose columns do
// not add up is a report nobody trusts. So one half is rounded and the other
// takes the remainder - the same cumulative-rounding treatment this codebase
// gives a season total split across 137 days.
//
// The rate half is the rounded one because it is the half a controller checks
// against an invoice: it has to match the paperwork to the cent. The usage half
// absorbs the difference, which is at most one cent and is the conventional
// accounting treatment of the residual.
func SplitVariance(actualCost, plannedCost, actualRate, standardRate, actualQuantity Dec) (total, rate, usage Dec) {
	total = RoundMoney(actualCost.Sub(plannedCost))
	rate = RateVariance(actualRate, standardRate, actualQuantity)
	usage = RoundMoney(total.Sub(rate))
	return total, rate, usage
}

// VarianceCheck is C41: the two variances reconstruct the total difference.
//
//	total variance = actual cost - planned cost = rate variance + usage variance
//
// It holds exactly, which is why the decomposition is worth having: a
// controller can point at either half and the halves still sum to the number on
// the report.
func VarianceCheck(actualCost, plannedCost, rateVariance, usageVariance Dec) bool {
	return RoundMoney(actualCost.Sub(plannedCost)).
		Equal(RoundMoney(rateVariance.Add(usageVariance)))
}

// ---------------------------------------------------------------------------
// The cost run
// ---------------------------------------------------------------------------

// CostLine is one element's contribution to a cost run.
type CostLine struct {
	ElementID     string       `json:"elementId"`
	ElementCode   string       `json:"elementCode"`
	ElementName   string       `json:"elementName"`
	Category      CostCategory `json:"category"`
	Driver        CostDriver   `json:"driver"`
	DriverUnit    string       `json:"driverUnit"`
	Variable      bool         `json:"variable"`
	StandardRate  Dec          `json:"standardRate"`
	ActualRate    Dec          `json:"actualRate"`
	PlannedQty    Dec          `json:"plannedQty"`
	ActualQty     Dec          `json:"actualQty"`
	PlannedCost   Dec          `json:"plannedCost"`
	ActualCost    Dec          `json:"actualCost"`
	RateVariance  Dec          `json:"rateVariance"`
	UsageVariance Dec          `json:"usageVariance"`
	TotalVariance Dec          `json:"totalVariance"`
	// Missing names what the run could not find, so a line that is zero because
	// nobody entered a rate is not mistaken for a line that is genuinely free.
	Missing []string `json:"missing,omitempty"`
}

// CostTotals is the run's bottom line.
type CostTotals struct {
	PlannedCost   Dec `json:"plannedCost"`
	ActualCost    Dec `json:"actualCost"`
	RateVariance  Dec `json:"rateVariance"`
	UsageVariance Dec `json:"usageVariance"`
	TotalVariance Dec `json:"totalVariance"`
	// VariancePct is the total variance against the plan.
	VariancePct Dec `json:"variancePct"`

	PlannedSugarTons Dec `json:"plannedSugarTons"`
	ActualSugarTons  Dec `json:"actualSugarTons"`
	PlannedUnitCost  Dec `json:"plannedUnitCost"`
	ActualUnitCost   Dec `json:"actualUnitCost"`
	UnitCostVariance Dec `json:"unitCostVariance"`

	FixedCost    Dec `json:"fixedCost"`
	VariableCost Dec `json:"variableCost"`
}

// CostInput is everything a cost run needs. As with the plan generator, the
// service gathers it and this function does the arithmetic, so the whole
// calculation can be tested without a database.
type CostInput struct {
	Elements []CostElement
	Rates    []CostRate
	Planned  DriverQuantities
	Actual   DriverQuantities
	// On is the date the rates are read for. A run covering a range uses its
	// last day, so a rate change inside the range applies to the whole of it;
	// splitting the range is how a site avoids that.
	On BusinessDate
	// Currency is what the run reports in. Rates in another currency are
	// converted with Exchange.
	Currency string
	Exchange []ExchangeRate
}

// CostOutput is the result of a run.
type CostOutput struct {
	Lines    []CostLine   `json:"lines"`
	Totals   CostTotals   `json:"totals"`
	Currency string       `json:"currency"`
	On       BusinessDate `json:"on"`
	Warnings []Alert      `json:"warnings,omitempty"`
}

// CalculateCost runs the costing for one set of elements, rates and driver
// quantities.
//
// It never fails on missing data. A cost element with no standard rate in force
// contributes nothing and says so on its line and in a warning; refusing the
// whole run because one rate is missing would make the report useless exactly
// when it is most needed, which is early in a season while the rates are still
// being entered.
func CalculateCost(in CostInput) CostOutput {
	out := CostOutput{Currency: in.Currency, On: in.On, Lines: []CostLine{}}

	elements := append([]CostElement(nil), in.Elements...)
	sort.Slice(elements, func(i, j int) bool {
		if elements[i].Category != elements[j].Category {
			return elements[i].Category < elements[j].Category
		}
		return elements[i].Code < elements[j].Code
	})

	for _, element := range elements {
		if !element.Active {
			continue
		}
		line := CostLine{
			ElementID: element.ID, ElementCode: element.Code, ElementName: element.Name,
			Category: element.Category, Driver: element.Driver,
			DriverUnit: DriverUnit(element.Driver), Variable: element.Variable,
			StandardRate: Zero, ActualRate: Zero,
			PlannedQty:  in.Planned.Quantity(element.Driver),
			ActualQty:   in.Actual.Quantity(element.Driver),
			PlannedCost: Zero, ActualCost: Zero,
			RateVariance: Zero, UsageVariance: Zero, TotalVariance: Zero,
		}

		standard, hasStandard := RateOn(in.Rates, element.ID, RateStandard, in.On)
		if hasStandard {
			line.StandardRate = convertRate(standard, in, &out)
		} else {
			line.Missing = append(line.Missing, "standard rate")
			out.Warnings = append(out.Warnings, Alert{
				Code: "COST_NO_STANDARD_RATE", Severity: SeverityWarning,
				Title: fmt.Sprintf("%s has no standard rate", element.Code),
				Detail: fmt.Sprintf(
					"No standard rate for %s (%s) is in force on %s, so it contributes nothing to the planned cost.",
					element.Name, element.Code, in.On),
			})
		}

		// A site that has not entered an actual rate is charged at standard,
		// which is the ordinary accounting treatment before the invoice
		// arrives: the quantity is known, the price is assumed, and the rate
		// variance is zero until somebody knows better.
		actual, hasActual := RateOn(in.Rates, element.ID, RateActual, in.On)
		switch {
		case hasActual:
			line.ActualRate = convertRate(actual, in, &out)
		case hasStandard:
			line.ActualRate = line.StandardRate
			line.Missing = append(line.Missing, "actual rate, charged at standard")
		default:
			line.Missing = append(line.Missing, "actual rate")
		}

		line.PlannedCost = ElementCost(line.StandardRate, line.PlannedQty)
		line.ActualCost = ElementCost(line.ActualRate, line.ActualQty)
		line.TotalVariance, line.RateVariance, line.UsageVariance = SplitVariance(
			line.ActualCost, line.PlannedCost, line.ActualRate, line.StandardRate, line.ActualQty)

		out.Lines = append(out.Lines, line)

		// C37: the total is the sum of the element costs. It is written out
		// rather than derived from a driver, because a season's cost is what its
		// elements add up to and nothing else.
		out.Totals.PlannedCost = out.Totals.PlannedCost.Add(line.PlannedCost)
		out.Totals.ActualCost = out.Totals.ActualCost.Add(line.ActualCost)
		out.Totals.RateVariance = out.Totals.RateVariance.Add(line.RateVariance)
		out.Totals.UsageVariance = out.Totals.UsageVariance.Add(line.UsageVariance)
		if element.Variable {
			out.Totals.VariableCost = out.Totals.VariableCost.Add(line.ActualCost)
		} else {
			out.Totals.FixedCost = out.Totals.FixedCost.Add(line.ActualCost)
		}
	}

	t := &out.Totals
	t.PlannedCost, t.ActualCost = RoundMoney(t.PlannedCost), RoundMoney(t.ActualCost)
	t.RateVariance, t.UsageVariance = RoundMoney(t.RateVariance), RoundMoney(t.UsageVariance)
	t.FixedCost, t.VariableCost = RoundMoney(t.FixedCost), RoundMoney(t.VariableCost)
	t.TotalVariance = RoundMoney(t.ActualCost.Sub(t.PlannedCost))
	t.VariancePct = RoundPct(SafePct(t.TotalVariance, t.PlannedCost))

	t.PlannedSugarTons, t.ActualSugarTons = in.Planned.SugarTons, in.Actual.SugarTons
	t.PlannedUnitCost = UnitCost(t.PlannedCost, t.PlannedSugarTons)
	t.ActualUnitCost = UnitCost(t.ActualCost, t.ActualSugarTons)
	t.UnitCostVariance = RoundMoney(t.ActualUnitCost.Sub(t.PlannedUnitCost))

	// SplitVariance makes each line reconcile by construction, and the totals
	// are sums of reconciling lines, so this can only fail if that construction
	// has been broken. It stays as an assertion rather than as an expected
	// outcome: a report whose columns do not add up is one nobody trusts, and
	// it should say so loudly rather than let a reader find it.
	if !VarianceCheck(t.ActualCost, t.PlannedCost, t.RateVariance, t.UsageVariance) {
		difference := RoundMoney(t.TotalVariance.Sub(t.RateVariance.Add(t.UsageVariance)))
		out.Warnings = append(out.Warnings, Alert{
			Code: "COST_VARIANCE_UNRECONCILED", Severity: SeverityError,
			Title: "The variance split does not reconstruct the total",
			Detail: fmt.Sprintf(
				"Rate and usage variance sum to %s against a total variance of %s, a difference of %s. "+
					"This is a defect in the costing, not a property of the data.",
				RoundMoney(t.RateVariance.Add(t.UsageVariance)), t.TotalVariance, difference),
		})
	}
	return out
}

// convertRate expresses a rate in the run's currency, recording a warning when
// it cannot.
func convertRate(rate CostRate, in CostInput, out *CostOutput) Dec {
	if in.Currency == "" || rate.Currency == "" || rate.Currency == in.Currency {
		return rate.Rate
	}
	converted, err := Convert(rate.Rate, rate.Currency, in.Currency, in.Exchange, in.On)
	if err != nil {
		out.Warnings = append(out.Warnings, Alert{
			Code: "COST_NO_EXCHANGE_RATE", Severity: SeverityError,
			Title:  fmt.Sprintf("No exchange rate from %s to %s", rate.Currency, in.Currency),
			Detail: err.Error(),
		})
		return Zero
	}
	return converted
}

// ---------------------------------------------------------------------------
// Stored runs
// ---------------------------------------------------------------------------

// CostRun is a saved costing, so that a figure quoted in a board pack can be
// reproduced later even after the rates have moved on.
type CostRun struct {
	ID        string       `json:"id"`
	SeasonID  string       `json:"seasonId"`
	VersionID string       `json:"versionId"`
	FactoryID string       `json:"factoryId"`
	Code      string       `json:"code"`
	From      BusinessDate `json:"from"`
	To        BusinessDate `json:"to"`
	Currency  string       `json:"currency"`
	Totals    CostTotals   `json:"totals"`
	Lines     []CostLine   `json:"lines,omitempty"`
	Note      string       `json:"note,omitempty"`
	AuditFields
}
