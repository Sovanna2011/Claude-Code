// Package domain holds the model types and the area arithmetic the whole system reports on.
//
// The arithmetic lives here, in pure functions over plain numbers, so it can be reasoned about
// and tested without a database. The same rules are enforced a second time in SQL — check
// constraints and deferred constraint triggers — because a rule that only exists in application
// code is a rule that a migration script or a direct UPDATE can walk straight past.
package domain

import (
	"fmt"
	"math"
)

// Areas is the six-figure classification of a parcel of land, in hectares. It applies unchanged at
// block, zone and farm level: the higher levels are the sum of the blocks beneath them.
type Areas struct {
	TotalHa        float64 `json:"totalAreaHa"`
	PlantableHa    float64 `json:"plantableAreaHa"`
	NonPlantableHa float64 `json:"nonPlantableAreaHa"`
	NewPlantingHa  float64 `json:"newPlantingAreaHa"`
	RatoonHa       float64 `json:"ratoonAreaHa"`
	WithCaneHa     float64 `json:"areaWithCaneHa"`
	AvailableHa    float64 `json:"availableAreaHa"`
}

// Add accumulates one node's areas into another. This is the only place the roll-up from block to
// zone to farm happens in Go, and it is a plain sum — no level may hold a figure of its own.
func (a *Areas) Add(other Areas) {
	a.TotalHa += other.TotalHa
	a.PlantableHa += other.PlantableHa
	a.NonPlantableHa += other.NonPlantableHa
	a.NewPlantingHa += other.NewPlantingHa
	a.RatoonHa += other.RatoonHa
	a.WithCaneHa += other.WithCaneHa
	a.AvailableHa += other.AvailableHa
}

// Round trims the accumulated floating-point noise to the precision the database stores, so a sum
// of four hectare figures does not surface as 1493.2999999999997.
func (a *Areas) Round() {
	round := func(v float64) float64 { return math.Round(v*10000) / 10000 }
	a.TotalHa = round(a.TotalHa)
	a.PlantableHa = round(a.PlantableHa)
	a.NonPlantableHa = round(a.NonPlantableHa)
	a.NewPlantingHa = round(a.NewPlantingHa)
	a.RatoonHa = round(a.RatoonHa)
	a.WithCaneHa = round(a.WithCaneHa)
	a.AvailableHa = round(a.AvailableHa)
}

// PercentOfTotal expresses a figure as a percentage of the total area, which is what the KPI cards
// show beside every hectare figure. A plantation with no registered land reports 0, not NaN.
func PercentOfTotal(value, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return math.Round(value/total*10000) / 100
}

// Achievement is actual over planned as a percentage. Planned zero with actual work done is
// reported as 100 rather than infinity: the programme was met, there was simply nothing planned.
func Achievement(actual, planned float64) float64 {
	if planned <= 0 {
		if actual > 0 {
			return 100
		}
		return 0
	}
	return math.Round(actual/planned*10000) / 100
}

// tolerance absorbs the last digit of numeric(12,4) when comparing hectare figures. Without it a
// block whose plantable area is exactly its planted area can fail its own validation.
const tolerance = 0.00005

// BlockAreaInput is what a caller may set on a block. Everything else about its areas — the
// unplantable remainder, the area under cane, the area still available — is derived.
type BlockAreaInput struct {
	TotalHa           float64
	PlantableHa       float64
	NewPlantingHa     float64
	RatoonHa          float64
	NonPlantableClaim float64 // hectares accounted for by recorded non-plantable reasons
}

// ValidateBlockAreas applies section 10 in full. It returns every problem it finds rather than the
// first, because a form that reports one error at a time takes as many round trips as it has
// mistakes.
func ValidateBlockAreas(in BlockAreaInput) []FieldError {
	var errs []FieldError
	add := func(field, message string) { errs = append(errs, FieldError{Field: field, Message: message}) }

	if in.TotalHa < -tolerance {
		add("totalAreaHa", "Total area cannot be negative.")
	}
	if in.PlantableHa < -tolerance {
		add("plantableAreaHa", "Plantable area cannot be negative.")
	}
	if in.NewPlantingHa < -tolerance {
		add("newPlantingAreaHa", "New planting area cannot be negative.")
	}
	if in.RatoonHa < -tolerance {
		add("ratoonAreaHa", "Ratoon area cannot be negative.")
	}
	if in.NonPlantableClaim < -tolerance {
		add("nonPlantableAreaHa", "Non-plantable area cannot be negative.")
	}

	if in.PlantableHa > in.TotalHa+tolerance {
		add("plantableAreaHa", fmt.Sprintf(
			"Plantable area (%.4f ha) cannot exceed the total area (%.4f ha).", in.PlantableHa, in.TotalHa))
	}

	nonPlantable := in.TotalHa - in.PlantableHa
	if in.NonPlantableClaim > nonPlantable+tolerance {
		add("nonPlantableAreaHa", fmt.Sprintf(
			"The recorded reasons account for %.4f ha but only %.4f ha of the block is unplantable.",
			in.NonPlantableClaim, nonPlantable))
	}

	if in.NewPlantingHa > in.PlantableHa+tolerance {
		add("newPlantingAreaHa", fmt.Sprintf(
			"New planting area (%.4f ha) cannot exceed the plantable area (%.4f ha).", in.NewPlantingHa, in.PlantableHa))
	}
	if in.RatoonHa > in.PlantableHa+tolerance {
		add("ratoonAreaHa", fmt.Sprintf(
			"Ratoon area (%.4f ha) cannot exceed the plantable area (%.4f ha).", in.RatoonHa, in.PlantableHa))
	}

	// The two together are the area with cane, and it is the sum that has to fit — each half can
	// be within the plantable area while the crop as a whole is not.
	withCane := in.NewPlantingHa + in.RatoonHa
	if withCane > in.PlantableHa+tolerance {
		add("areaWithCaneHa", fmt.Sprintf(
			"Area with cane (%.4f ha of new planting plus ratoon) cannot exceed the plantable area (%.4f ha).",
			withCane, in.PlantableHa))
	}

	return errs
}

// DeriveBlockAreas completes the classification from the figures a user actually supplies.
func DeriveBlockAreas(in BlockAreaInput) Areas {
	areas := Areas{
		TotalHa:        in.TotalHa,
		PlantableHa:    in.PlantableHa,
		NonPlantableHa: in.TotalHa - in.PlantableHa,
		NewPlantingHa:  in.NewPlantingHa,
		RatoonHa:       in.RatoonHa,
		WithCaneHa:     in.NewPlantingHa + in.RatoonHa,
	}
	areas.AvailableHa = math.Max(areas.PlantableHa-areas.WithCaneHa, 0)
	areas.Round()
	return areas
}
