package domain

import (
	"errors"
	"math"
	"time"
)

// Every planning formula in the specification, implemented once as pure functions so the API, the
// requirement engines, the reports and the tests all agree. Nothing here touches the database.
//
// Quantities are float64 rounded to four decimals at every step — one square metre on a hectare,
// which is the precision the numeric(12,4) columns store. Four decimals is exact in float64 up to
// about 10^11, far beyond any plantation, and rounding at each step rather than only at the end is
// what keeps a sum of stored figures equal to the stored sum.

// QuantityScale is the number of decimals every stored quantity carries.
const QuantityScale = 4

// ErrInvalidInput is returned by the formulas that cannot answer for their arguments — a division
// by zero working days, or a negative area. They return an error rather than a plausible number,
// because a requirement of "infinity tractors" reported as zero is worse than a refusal.
var ErrInvalidInput = errors.New("invalid input")

// Round4 applies the storage precision, rounding halves away from zero the way the .NET
// implementation this was ported from does — so the two systems agree digit for digit.
func Round4(v float64) float64 {
	const scale = 1e4
	if v < 0 {
		return -math.Floor(-v*scale+0.5) / scale
	}
	return math.Floor(v*scale+0.5) / scale
}

// ---------------------------------------------------------------- section 5: projection

// HarvestableArea = projected planting area × (1 − expected loss % ÷ 100).
func HarvestableArea(projectedPlantingArea, expectedLossPercent float64) (float64, error) {
	if projectedPlantingArea < 0 {
		return 0, ErrInvalidInput
	}
	if expectedLossPercent < 0 || expectedLossPercent > 100 {
		return 0, ErrInvalidInput
	}
	return Round4(projectedPlantingArea * (1 - expectedLossPercent/100)), nil
}

// ExpectedCaneProduction = harvestable area × expected yield per hectare.
func ExpectedCaneProduction(harvestableArea, expectedYieldPerHectare float64) (float64, error) {
	if harvestableArea < 0 || expectedYieldPerHectare < 0 {
		return 0, ErrInvalidInput
	}
	return Round4(harvestableArea * expectedYieldPerHectare), nil
}

// ---------------------------------------------------------------- section 7: activity plan

// DailyTarget = planned activity area ÷ available working days.
func DailyTarget(plannedArea float64, availableWorkingDays int) (float64, error) {
	if plannedArea < 0 || availableWorkingDays <= 0 {
		return 0, ErrInvalidInput
	}
	return Round4(plannedArea / float64(availableWorkingDays)), nil
}

// WorkingCalendar is the company's working week and its holidays. It is passed to every date
// calculation rather than read from a global, so a scenario can ask "what if we worked Sundays"
// without changing the plan.
type WorkingCalendar struct {
	WorkOnSaturday bool
	WorkOnSunday   bool
	Holidays       map[string]bool // keyed by 2006-01-02
}

// DefaultCalendar matches the specification's default: Saturdays worked, Sundays not.
func DefaultCalendar() WorkingCalendar {
	return WorkingCalendar{WorkOnSaturday: true, WorkOnSunday: false, Holidays: map[string]bool{}}
}

func (c WorkingCalendar) isWorkingDay(d time.Time) bool {
	switch d.Weekday() {
	case time.Saturday:
		if !c.WorkOnSaturday {
			return false
		}
	case time.Sunday:
		if !c.WorkOnSunday {
			return false
		}
	}
	return !c.Holidays[d.Format("2006-01-02")]
}

// WorkingDays counts the working days between two dates, both inclusive.
func WorkingDays(start, end time.Time, calendar WorkingCalendar) int {
	start, end = midnight(start), midnight(end)
	if end.Before(start) {
		return 0
	}
	days := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if calendar.isWorkingDay(d) {
			days++
		}
	}
	return days
}

// AddWorkingDays returns the date that is n working days after start, counting start itself as the
// first working day when it is one. It is what turns "this activity takes twelve working days"
// into an end date the calendar agrees with.
func AddWorkingDays(start time.Time, workingDays int, calendar WorkingCalendar) time.Time {
	d := midnight(start)
	if workingDays <= 0 {
		return d
	}
	counted := 0
	for {
		if calendar.isWorkingDay(d) {
			counted++
			if counted >= workingDays {
				return d
			}
		}
		d = d.AddDate(0, 0, 1)
	}
}

// NextWorkingDay moves a date forward to the first day that is worked, leaving it alone if it
// already is one.
func NextWorkingDay(d time.Time, calendar WorkingCalendar) time.Time {
	d = midnight(d)
	for !calendar.isWorkingDay(d) {
		d = d.AddDate(0, 0, 1)
	}
	return d
}

func midnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// ---------------------------------------------------------------- section 8: machinery

// RequiredTractors = planned area ÷ (capacity per tractor per day × available working days),
// always rounded up. The specification's worked example: 2,600 ÷ (4 × 90) = 7.22 → 8 tractors.
func RequiredTractors(plannedArea, capacityPerTractorPerDay float64, availableWorkingDays int) (int, error) {
	if plannedArea <= 0 {
		return 0, nil
	}
	if capacityPerTractorPerDay <= 0 || availableWorkingDays <= 0 {
		return 0, ErrInvalidInput
	}
	return int(math.Ceil(plannedArea / (capacityPerTractorPerDay * float64(availableWorkingDays)))), nil
}

// RequiredEquipment is the same arithmetic applied to an implement.
func RequiredEquipment(plannedArea, capacityPerUnitPerDay float64, availableWorkingDays int) (int, error) {
	return RequiredTractors(plannedArea, capacityPerUnitPerDay, availableWorkingDays)
}

// ExactMachineRequirement is the un-rounded ratio, reported beside the rounded figure so a planner
// can see that 8 tractors covers 7.22 and judge whether the eighth is worth hiring.
func ExactMachineRequirement(plannedArea, capacityPerUnitPerDay float64, availableWorkingDays int) float64 {
	if plannedArea <= 0 || capacityPerUnitPerDay <= 0 || availableWorkingDays <= 0 {
		return 0
	}
	return Round4(plannedArea / (capacityPerUnitPerDay * float64(availableWorkingDays)))
}

// ---------------------------------------------------------------- section 12: materials

// BaseMaterialRequirement = planned area × standard rate per hectare.
func BaseMaterialRequirement(plannedArea, standardRatePerHectare float64) float64 {
	return Round4(plannedArea * standardRatePerHectare)
}

// WasteQuantity = base requirement × waste % ÷ 100.
func WasteQuantity(baseRequirement, wastePercent float64) (float64, error) {
	if wastePercent < 0 {
		return 0, ErrInvalidInput
	}
	return Round4(baseRequirement * wastePercent / 100), nil
}

// TotalMaterialRequirement = base requirement + waste, over the number of applications.
func TotalMaterialRequirement(plannedArea, standardRatePerHectare, wastePercent float64, applications int) (float64, error) {
	if applications < 1 {
		applications = 1
	}
	base := BaseMaterialRequirement(plannedArea, standardRatePerHectare) * float64(applications)
	waste, err := WasteQuantity(base, wastePercent)
	if err != nil {
		return 0, err
	}
	return Round4(base + waste), nil
}

// RequiredSeedCane = new planting area × seed rate per hectare.
func RequiredSeedCane(newPlantingArea, seedRatePerHectare float64) float64 {
	return Round4(newPlantingArea * seedRatePerHectare)
}

// RequiredChemical = treatment area × application rate × number of applications.
func RequiredChemical(treatmentArea, applicationRate float64, applications int) float64 {
	if applications < 1 {
		applications = 1
	}
	return Round4(treatmentArea * applicationRate * float64(applications))
}

// ---------------------------------------------------------------- section 13: stock

// NetAvailableQuantity = available stock + incoming − reserved.
func NetAvailableQuantity(availableStock, incoming, reserved float64) float64 {
	return Round4(availableStock + incoming - reserved)
}

// ShortageQuantity = max(total requirement − net available, 0).
func ShortageQuantity(totalRequirement, netAvailable float64) float64 {
	return Round4(math.Max(totalRequirement-netAvailable, 0))
}

// SurplusQuantity = max(net available − total requirement, 0).
func SurplusQuantity(totalRequirement, netAvailable float64) float64 {
	return Round4(math.Max(netAvailable-totalRequirement, 0))
}

// ---------------------------------------------------------------- section 14: fuel and labour

// ProjectedFuelByArea = planned area × standard litres per hectare.
func ProjectedFuelByArea(plannedArea, litresPerHectare float64) float64 {
	return Round4(plannedArea * litresPerHectare)
}

// ProjectedFuelByHour = planned working hours × standard litres per hour.
func ProjectedFuelByHour(plannedWorkingHours, litresPerHour float64) float64 {
	return Round4(plannedWorkingHours * litresPerHour)
}

// RequiredLabourDays = planned area × standard labour-days per hectare.
func RequiredLabourDays(plannedArea, labourDaysPerHectare float64) float64 {
	return Round4(plannedArea * labourDaysPerHectare)
}

// RequiredWorkers = required labour-days ÷ available working days, rounded up.
func RequiredWorkers(requiredLabourDays float64, availableWorkingDays int) (int, error) {
	if requiredLabourDays <= 0 {
		return 0, nil
	}
	if availableWorkingDays <= 0 {
		return 0, ErrInvalidInput
	}
	return int(math.Ceil(requiredLabourDays / float64(availableWorkingDays))), nil
}

// ---------------------------------------------------------------- section 15: capacity

// Capacity grades of the specification.
const (
	CapacitySufficient  = "Sufficient"
	CapacityAtRisk      = "AtRisk"
	CapacityShortage    = "Shortage"
	CapacityUnavailable = "Unavailable"
)

// DefaultAtRiskThreshold: coverage below 90 % of the requirement is a shortage, not merely a risk.
const DefaultAtRiskThreshold = 0.90

// EvaluateCapacity grades a requirement against what is available.
func EvaluateCapacity(required, available, atRiskThreshold float64) string {
	if atRiskThreshold <= 0 {
		atRiskThreshold = DefaultAtRiskThreshold
	}
	if required <= 0 {
		return CapacitySufficient
	}
	if available <= 0 {
		return CapacityUnavailable
	}
	if available >= required {
		return CapacitySufficient
	}
	if available/required >= atRiskThreshold {
		return CapacityAtRisk
	}
	return CapacityShortage
}

// CoveragePercent is available ÷ required as a percentage, capped so a requirement of almost
// nothing against a full store does not report a number nobody can read.
func CoveragePercent(required, available float64) float64 {
	if required <= 0 {
		return 100
	}
	return Round4(math.Min(available/required*100, 999.99))
}

// ---------------------------------------------------------------- section 17: variance

// AreaVariance = actual − planned. Negative means behind the programme.
func AreaVariance(actualArea, plannedArea float64) float64 { return Round4(actualArea - plannedArea) }

// MaterialVariance = actual − planned.
func MaterialVariance(actual, planned float64) float64 { return Round4(actual - planned) }

// FuelVariance = actual − planned.
func FuelVariance(actual, planned float64) float64 { return Round4(actual - planned) }

// ScheduleVarianceDays = actual completion − planned completion, in days. Nil when either date is
// still open, because "no answer yet" is not the same as "on time".
func ScheduleVarianceDays(actualCompletion, plannedCompletion *time.Time) *int {
	if actualCompletion == nil || plannedCompletion == nil {
		return nil
	}
	days := int(midnight(*actualCompletion).Sub(midnight(*plannedCompletion)).Hours() / 24)
	return &days
}

// CompletionPercentage = actual completed area ÷ planned area × 100.
func CompletionPercentage(actualCompletedArea, plannedArea float64) float64 {
	if plannedArea <= 0 {
		return 0
	}
	return Round4(actualCompletedArea / plannedArea * 100)
}

// ---------------------------------------------------------------- unit conversion

// ToBaseUnit converts a quantity in a material's alternative unit into its base unit.
func ToBaseUnit(quantityInAlternativeUnit, conversionFactor float64) (float64, error) {
	if conversionFactor <= 0 {
		return 0, ErrInvalidInput
	}
	return Round4(quantityInAlternativeUnit * conversionFactor), nil
}

// ToAlternativeUnit is the inverse of ToBaseUnit.
func ToAlternativeUnit(quantityInBaseUnit, conversionFactor float64) (float64, error) {
	if conversionFactor <= 0 {
		return 0, ErrInvalidInput
	}
	return Round4(quantityInBaseUnit / conversionFactor), nil
}

// ---------------------------------------------------------------- date helpers

// DateRangesOverlap is true when two inclusive date ranges share at least one day. It is the rule
// behind "this block already has a committed plan then" and behind every booking conflict.
func DateRangesOverlap(startA, endA, startB, endB time.Time) bool {
	return !midnight(startA).After(midnight(endB)) && !midnight(startB).After(midnight(endA))
}

// TimeRangesOverlap is true when two half-open time windows share at least one minute. Half-open
// is deliberate: a machine freed at 12:00 can be booked again from 12:00.
func TimeRangesOverlap(startA, endA, startB, endB time.Time) bool {
	return startA.Before(endB) && startB.Before(endA)
}
