package domain

import "testing"

// The specification's section 10 is a list of things that must never be storable. Each one gets a
// test, because a validation rule with no test is a rule that quietly stops working.

func TestValidateBlockAreas_acceptsAConsistentBlock(t *testing.T) {
	problems := ValidateBlockAreas(BlockAreaInput{
		TotalHa: 100, PlantableHa: 90, NewPlantingHa: 40, RatoonHa: 30, NonPlantableClaim: 10,
	})
	if len(problems) != 0 {
		t.Fatalf("expected no problems, got %v", problems)
	}
}

func TestValidateBlockAreas_rejectsEachRule(t *testing.T) {
	cases := []struct {
		name  string
		in    BlockAreaInput
		field string
	}{
		{"negative total", BlockAreaInput{TotalHa: -1}, "totalAreaHa"},
		{"negative plantable", BlockAreaInput{TotalHa: 10, PlantableHa: -5}, "plantableAreaHa"},
		{"negative new planting", BlockAreaInput{TotalHa: 10, PlantableHa: 10, NewPlantingHa: -1}, "newPlantingAreaHa"},
		{"negative ratoon", BlockAreaInput{TotalHa: 10, PlantableHa: 10, RatoonHa: -1}, "ratoonAreaHa"},
		{"plantable beyond total", BlockAreaInput{TotalHa: 10, PlantableHa: 11}, "plantableAreaHa"},
		{"reasons beyond unplantable", BlockAreaInput{TotalHa: 10, PlantableHa: 9, NonPlantableClaim: 2}, "nonPlantableAreaHa"},
		{"new planting beyond plantable", BlockAreaInput{TotalHa: 10, PlantableHa: 8, NewPlantingHa: 9}, "newPlantingAreaHa"},
		{"ratoon beyond plantable", BlockAreaInput{TotalHa: 10, PlantableHa: 8, RatoonHa: 9}, "ratoonAreaHa"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			problems := ValidateBlockAreas(tc.in)
			if len(problems) == 0 {
				t.Fatalf("expected %s to be rejected", tc.name)
			}
			for _, p := range problems {
				if p.Field == tc.field {
					return
				}
			}
			t.Fatalf("expected a problem on %q, got %v", tc.field, problems)
		})
	}
}

// The sum is the rule that matters and the one a per-field check misses: each half fits, the crop
// as a whole does not.
func TestValidateBlockAreas_catchesTheSumEvenWhenEachHalfFits(t *testing.T) {
	problems := ValidateBlockAreas(BlockAreaInput{
		TotalHa: 100, PlantableHa: 60, NewPlantingHa: 40, RatoonHa: 40,
	})

	var found bool
	for _, p := range problems {
		if p.Field == "areaWithCaneHa" {
			found = true
		}
		if p.Field == "newPlantingAreaHa" || p.Field == "ratoonAreaHa" {
			t.Fatalf("neither half exceeds the plantable area on its own: %v", p)
		}
	}
	if !found {
		t.Fatalf("expected the combined cane area to be rejected, got %v", problems)
	}
}

// A block planted right up to its boundary is legal. Without a tolerance the last digit of
// numeric(12,4) makes it fail its own validation.
func TestValidateBlockAreas_allowsAFullyPlantedBlock(t *testing.T) {
	problems := ValidateBlockAreas(BlockAreaInput{
		TotalHa: 130.9, PlantableHa: 119.1, NewPlantingHa: 83.4, RatoonHa: 35.7,
	})
	if len(problems) != 0 {
		t.Fatalf("83.4 + 35.7 is exactly 119.1; expected no problems, got %v", problems)
	}
}

func TestValidateBlockAreas_reportsEveryProblemAtOnce(t *testing.T) {
	problems := ValidateBlockAreas(BlockAreaInput{TotalHa: -5, PlantableHa: -2, NewPlantingHa: -1})
	if len(problems) < 3 {
		t.Fatalf("a form should learn all its mistakes in one round trip, got %v", problems)
	}
}

func TestDeriveBlockAreas(t *testing.T) {
	areas := DeriveBlockAreas(BlockAreaInput{TotalHa: 130.9, PlantableHa: 119.1, NewPlantingHa: 65.5, RatoonHa: 35.7})

	if got, want := areas.NonPlantableHa, 11.8; !closeTo(got, want) {
		t.Errorf("non-plantable = %v, want %v", got, want)
	}
	if got, want := areas.WithCaneHa, 101.2; !closeTo(got, want) {
		t.Errorf("with cane = %v, want %v", got, want)
	}
	if got, want := areas.AvailableHa, 17.9; !closeTo(got, want) {
		t.Errorf("available = %v, want %v", got, want)
	}
}

// Available area is clamped: a block cannot have negative land left, whatever the records say.
func TestDeriveBlockAreas_neverReportsNegativeAvailableLand(t *testing.T) {
	areas := DeriveBlockAreas(BlockAreaInput{TotalHa: 100, PlantableHa: 50, NewPlantingHa: 80})
	if areas.AvailableHa != 0 {
		t.Fatalf("available = %v, want 0", areas.AvailableHa)
	}
}

func TestAreasAdd_isTheOnlyRollUp(t *testing.T) {
	total := Areas{}
	total.Add(Areas{TotalHa: 10, PlantableHa: 9, NonPlantableHa: 1, NewPlantingHa: 4, RatoonHa: 3, WithCaneHa: 7, AvailableHa: 2})
	total.Add(Areas{TotalHa: 20, PlantableHa: 18, NonPlantableHa: 2, NewPlantingHa: 5, RatoonHa: 5, WithCaneHa: 10, AvailableHa: 8})
	total.Round()

	want := Areas{TotalHa: 30, PlantableHa: 27, NonPlantableHa: 3, NewPlantingHa: 9, RatoonHa: 8, WithCaneHa: 17, AvailableHa: 10}
	if total != want {
		t.Fatalf("roll-up = %+v, want %+v", total, want)
	}
}

// Summing hectare figures in binary floating point leaves noise; the dashboard must not show it.
func TestAreasRound_clearsFloatingPointNoise(t *testing.T) {
	a := Areas{}
	for i := 0; i < 3; i++ {
		a.Add(Areas{AvailableHa: 497.7666})
	}
	a.Round()
	if a.AvailableHa != 1493.2998 {
		t.Fatalf("available = %v, want 1493.2998", a.AvailableHa)
	}
}

func TestPercentOfTotal(t *testing.T) {
	if got := PercentOfTotal(845.3, 2618.8); got != 32.28 {
		t.Errorf("percent = %v, want 32.28", got)
	}
	if got := PercentOfTotal(100, 0); got != 0 {
		t.Errorf("an estate with no land reports 0%%, got %v", got)
	}
}

func TestAchievement(t *testing.T) {
	if got := Achievement(845.3, 1070.5); got != 78.96 {
		t.Errorf("achievement = %v, want 78.96", got)
	}
	if got := Achievement(0, 0); got != 0 {
		t.Errorf("nothing planned and nothing done is 0%%, got %v", got)
	}
	// Unplanned work is met in full, not infinitely over-achieved.
	if got := Achievement(50, 0); got != 100 {
		t.Errorf("unplanned work reports 100%%, got %v", got)
	}
}

func TestGoogleMapsURL(t *testing.T) {
	lat, lng := -15.47432, 28.23328
	want := "https://www.google.com/maps/search/?api=1&query=-15.47432,28.23328"
	if got := GoogleMapsURL(&lat, &lng); got != want {
		t.Errorf("url = %q, want %q", got, want)
	}
	if got := GoogleMapsURL(nil, &lng); got != "" {
		t.Errorf("a block with no coordinates has no link, got %q", got)
	}
}

func TestPageNormalise(t *testing.T) {
	if got := (Page{Number: 0, Size: 0}).Normalise(); got.Number != 1 || got.Size != defaultPageSize {
		t.Errorf("defaults = %+v", got)
	}
	if got := (Page{Number: 3, Size: 99999}).Normalise(); got.Size != maxPageSize {
		t.Errorf("page size should be capped, got %d", got.Size)
	}
	if got := (Page{Number: 3, Size: 25}).Normalise().Offset(); got != 50 {
		t.Errorf("offset = %d, want 50", got)
	}
}

func TestCanonical(t *testing.T) {
	if got, ok := Canonical("ratoon", ValidPlantingTypes); !ok || got != "Ratoon" {
		t.Errorf("canonical(ratoon) = %q, %v", got, ok)
	}
	if _, ok := Canonical("Coppice", ValidPlantingTypes); ok {
		t.Error("an unknown planting type must be refused before it reaches SQL")
	}
}

func closeTo(a, b float64) bool {
	diff := a - b
	return diff < 0.0001 && diff > -0.0001
}
