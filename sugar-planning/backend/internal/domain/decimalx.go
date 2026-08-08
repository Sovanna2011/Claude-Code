// Package domain holds the pure business model of the sugar production
// planning system: entities, enumerations and the calculation catalogue.
//
// Nothing in this package talks to a database, an HTTP request or a clock it
// did not receive as an argument. That keeps the calculation catalogue
// (docs/06-calculation-catalogue.md) directly testable.
package domain

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Dec is the single numeric type used for every quantity, rate and percentage
// in the system. Binary floating point is never used for inventory,
// production quantities or recovery calculations.
type Dec = decimal.Decimal

// Scales used when a result is persisted or displayed. Intermediate results
// keep full precision; rounding happens once, at the end of a calculation, so
// that a chain of operations does not accumulate rounding drift.
const (
	// ScaleQuantity is the scale of stored quantities (tons, kg, pieces).
	ScaleQuantity int32 = 3
	// ScaleRate is the scale of rates such as tons/hour or tons/day.
	ScaleRate int32 = 3
	// ScalePercent is the scale of percentages (11.000 %, 82.415 %).
	ScalePercent int32 = 3
	// ScaleFactor is the scale of conversion factors and assumptions.
	ScaleFactor int32 = 6
	// ScaleMoney is the scale of a money amount. Two places is what an invoice
	// and a ledger use, and a cost report that carried more would be quoting a
	// precision the accounts do not have.
	ScaleMoney int32 = 2
	// ScaleUnitRate is the scale of a rate expressed in money per unit. It is
	// finer than money itself because a cost of a few cents per ton, multiplied
	// by 2,300,000 t, is a real number on the report.
	ScaleUnitRate int32 = 6
)

// Zero is the neutral quantity.
var Zero = decimal.Zero

// D parses a decimal from a string. It panics on malformed input and is
// intended for constants and tests, never for user input.
func D(s string) Dec {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(fmt.Sprintf("domain.D: invalid decimal %q: %v", s, err))
	}
	return d
}

// DI builds a decimal from an int64.
func DI(i int64) Dec { return decimal.NewFromInt(i) }

// ParseDec parses user supplied numeric input, returning a domain error rather
// than panicking.
func ParseDec(s string) (Dec, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Zero, fmt.Errorf("%w: %q is not a valid decimal number", ErrValidation, s)
	}
	return d, nil
}

// RoundQty rounds a quantity to the storage scale using half-away-from-zero,
// the rounding mode agreed with the business for tonnage.
func RoundQty(d Dec) Dec { return d.Round(ScaleQuantity) }

// RoundRate rounds a rate to the storage scale.
func RoundRate(d Dec) Dec { return d.Round(ScaleRate) }

// RoundPct rounds a percentage to the storage scale.
func RoundPct(d Dec) Dec { return d.Round(ScalePercent) }

// RoundFactor rounds a conversion factor or assumption value.
func RoundFactor(d Dec) Dec { return d.Round(ScaleFactor) }

// RoundMoney rounds a money amount to the ledger scale.
func RoundMoney(d Dec) Dec { return d.Round(ScaleMoney) }

// RoundUnitRate rounds money per unit of a cost driver.
func RoundUnitRate(d Dec) Dec { return d.Round(ScaleUnitRate) }

// SafeDiv divides a by b and returns zero when b is zero. Division by zero is
// a routine situation in this domain (day 1 of a season has no cumulative
// target yet) and must never panic or produce NaN.
func SafeDiv(a, b Dec) Dec {
	if b.IsZero() {
		return Zero
	}
	return a.Div(b)
}

// SafePct returns 100 * a / b, or zero when b is zero.
func SafePct(a, b Dec) Dec {
	if b.IsZero() {
		return Zero
	}
	return a.Div(b).Mul(DI(100))
}

// MaxDec returns the larger of a and b.
func MaxDec(a, b Dec) Dec {
	if a.GreaterThan(b) {
		return a
	}
	return b
}

// MinDec returns the smaller of a and b.
func MinDec(a, b Dec) Dec {
	if a.LessThan(b) {
		return a
	}
	return b
}

// ClampNonNegative replaces negative values with zero. Used where the business
// rule is "never below zero" (for example a purchase requirement).
func ClampNonNegative(d Dec) Dec {
	if d.IsNegative() {
		return Zero
	}
	return d
}

// SumDec adds a slice of decimals.
func SumDec(values ...Dec) Dec {
	total := Zero
	for _, v := range values {
		total = total.Add(v)
	}
	return total
}

// CeilInt returns the smallest integer greater than or equal to d. Used for
// discrete items such as bags, pallets and liners, which cannot be bought in
// fractions.
func CeilInt(d Dec) int64 {
	return d.Ceil().IntPart()
}
