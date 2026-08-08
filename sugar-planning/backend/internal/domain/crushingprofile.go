package domain

import (
	"fmt"
	"sort"
)

// The shape of a crushing season.
//
// The generator originally spread the cane target evenly across the campaign:
// 2,300,000 t over 137 days is 16,788 t a day, every day, from the first to the
// last. No mill runs like that, and the reference workbook proves it. Kampong
// Speu's own plan opens at 17,000 t while the boilers come up, settles at
// 19,000 t, drops to half rate the day before each wash-out, stops entirely for
// the wash-out itself, and then runs down through 15,000, 12,000, 8,000, 4,000
// and 3,000 t as the last of the cane comes in.
//
// The difference is not cosmetic. An even line puts the raw sugar into the silo
// at a constant rate, so the date the silo fills - which is the whole reason
// anyone plans jumbo bagging - lands in the wrong place. A plan that cannot say
// "we wash out every three weeks" cannot answer the question it was built for.
//
// The profile below is how the mill actually thinks about it: the shoulders are
// decided (the planner knows the start-up rate and the run-down), the wash-out
// cadence is a maintenance policy, and the plateau is whatever rate makes the
// season come to target. So the plateau is *solved*, not entered - which is the
// one number a spreadsheet gets wrong when the target changes.

// CrushingStep is a run of days at one rate.
//
// Rate is tons per day. Days is how many consecutive days it lasts.
type CrushingStep struct {
	Rate Dec `json:"rateTons"`
	Days int `json:"days"`
}

// CrushingProfile describes the shape of a campaign without fixing its size.
type CrushingProfile struct {
	// RampUp is the opening shoulder, in order from the first day.
	RampUp []CrushingStep `json:"rampUp,omitempty"`
	// RunDown is the closing shoulder, in order, ending on the last day.
	RunDown []CrushingStep `json:"runDown,omitempty"`

	// CleaningEveryDays is the wash-out cadence: every Nth day of the campaign
	// the mill stops. Zero means no scheduled wash-outs.
	CleaningEveryDays int `json:"cleaningEveryDays,omitempty"`
	// CleaningDays overrides the cadence with an explicit list of campaign day
	// numbers, one-based. The workbook's last wash-out is pulled forward from
	// day 126 to day 119 so it does not land in the run-down, and its own note
	// says the day "is adjusted according to suitability and situation" - so
	// the list has to be able to say something the cadence cannot.
	CleaningDays []int `json:"cleaningDays,omitempty"`
	// PreCleaningRate is the rate on the day before a wash-out. A mill winds
	// down before it stops; it does not go from full rate to nothing.
	PreCleaningRate Dec `json:"preCleaningRateTons,omitempty"`
}

// Step kinds, as stored.
const (
	CrushRampUp   = "RAMP_UP"
	CrushRunDown  = "RUN_DOWN"
	CrushCleaning = "CLEANING"
)

// CrushingStepRow is one stored row of a profile.
//
// The profile is a small ordered thing and it is stored as rows rather than as
// a JSON blob, because the generator reads it: a document the server interprets
// but cannot constrain is a document that eventually holds something the server
// cannot run. The check constraint in migration 0014 is the other half of this
// type.
type CrushingStepRow struct {
	ID        string `json:"id"`
	VersionID string `json:"versionId"`
	Kind      string `json:"kind"`
	Seq       int    `json:"seq"`
	// RateTons and Days describe a RAMP_UP or RUN_DOWN step.
	RateTons Dec `json:"rateTons,omitempty"`
	Days     int `json:"days,omitempty"`
	// CampaignDay is the one-based day a CLEANING step stops the mill.
	CampaignDay int `json:"campaignDay,omitempty"`
	AuditFields
}

// Validate checks one stored step.
func (s CrushingStepRow) Validate() error {
	verr := &ValidationError{}
	if s.VersionID == "" {
		verr.Add("versionId", "REQUIRED", "a step belongs to a plan version")
	}
	switch s.Kind {
	case CrushRampUp, CrushRunDown:
		if s.Days <= 0 {
			verr.Add("days", "INVALID", "a step lasts at least one day")
		}
		if s.RateTons.IsNegative() {
			verr.Add("rateTons", "NEGATIVE", "a crushing rate cannot be negative")
		}
		if s.CampaignDay != 0 {
			verr.Add("campaignDay", "INVALID", "only a cleaning step names a campaign day")
		}
	case CrushCleaning:
		if s.CampaignDay <= 0 {
			verr.Add("campaignDay", "INVALID", "a wash-out happens on a campaign day, counted from one")
		}
		if s.Days != 0 || !s.RateTons.IsZero() {
			verr.Add("rateTons", "INVALID", "a wash-out has no rate and no duration; it is one day")
		}
	default:
		verr.Add("kind", "INVALID", "a step is RAMP_UP, RUN_DOWN or CLEANING")
	}
	if verr.HasErrors() {
		return verr
	}
	return nil
}

// ProfileFromRows rebuilds a profile from its stored rows.
//
// The rows are assumed to arrive ordered by kind and sequence, which is what
// the store guarantees; ordering here as well would hide a store that stopped.
func ProfileFromRows(rows []CrushingStepRow, everyDays int, preCleaning Dec) CrushingProfile {
	p := CrushingProfile{CleaningEveryDays: everyDays, PreCleaningRate: preCleaning}
	for _, r := range rows {
		switch r.Kind {
		case CrushRampUp:
			p.RampUp = append(p.RampUp, CrushingStep{Rate: r.RateTons, Days: r.Days})
		case CrushRunDown:
			p.RunDown = append(p.RunDown, CrushingStep{Rate: r.RateTons, Days: r.Days})
		case CrushCleaning:
			p.CleaningDays = append(p.CleaningDays, r.CampaignDay)
		}
	}
	return p
}

// RowsFromProfile is the inverse, for storing a profile a planner has entered.
func RowsFromProfile(versionID string, p CrushingProfile) []CrushingStepRow {
	var out []CrushingStepRow
	for i, s := range p.RampUp {
		out = append(out, CrushingStepRow{VersionID: versionID, Kind: CrushRampUp,
			Seq: i, RateTons: s.Rate, Days: s.Days})
	}
	for i, s := range p.RunDown {
		out = append(out, CrushingStepRow{VersionID: versionID, Kind: CrushRunDown,
			Seq: i, RateTons: s.Rate, Days: s.Days})
	}
	for i, d := range p.CleaningDays {
		out = append(out, CrushingStepRow{VersionID: versionID, Kind: CrushCleaning,
			Seq: i, CampaignDay: d})
	}
	return out
}

// IsZero reports whether the profile says nothing, in which case the generator
// spreads the target evenly and behaves exactly as it did before profiles
// existed.
func (p CrushingProfile) IsZero() bool {
	return len(p.RampUp) == 0 && len(p.RunDown) == 0 &&
		p.CleaningEveryDays == 0 && len(p.CleaningDays) == 0
}

// cleaningSet resolves the wash-out days for a campaign of the given length.
// The explicit list wins over the cadence, because it exists to say what the
// cadence cannot.
func (p CrushingProfile) cleaningSet(days int) map[int]bool {
	out := map[int]bool{}
	if len(p.CleaningDays) > 0 {
		for _, d := range p.CleaningDays {
			if d >= 1 && d <= days {
				out[d] = true
			}
		}
		return out
	}
	if p.CleaningEveryDays > 0 {
		for d := p.CleaningEveryDays; d <= days; d += p.CleaningEveryDays {
			out[d] = true
		}
	}
	return out
}

// CrushingPlan is the result of applying a profile to a target over a campaign.
type CrushingPlan struct {
	// Daily is one tonnage per campaign day, in order. A wash-out day is zero.
	Daily []Dec
	// PlateauRate is the solved full rate.
	PlateauRate Dec
	// PlateauDays, CleaningDays and ShoulderDays account for every day of the
	// campaign, so a summary can be checked without re-walking the series.
	PlateauDays  int
	CleaningDays int
	ShoulderDays int
}

// BuildCrushingPlan lays a profile over `days` campaign days and solves the
// plateau rate so the daily tonnages sum to exactly `target`.
//
// The shoulders and the wash-outs are fixed by the profile; whatever tonnage
// they do not account for is spread across the remaining days. The remainder of
// the division lands on the last plateau day rather than being dropped, so the
// series adds back to the target to the kilogram - the same rule the rest of
// the planner follows, and the reason a season total can be quoted at all.
func BuildCrushingPlan(target Dec, days int, p CrushingProfile) (CrushingPlan, error) {
	if days <= 0 {
		return CrushingPlan{}, fmt.Errorf("%w: a campaign needs at least one day", ErrValidation)
	}
	if target.IsNegative() {
		return CrushingPlan{}, fmt.Errorf("%w: the cane target cannot be negative", ErrValidation)
	}

	daily := make([]Dec, days)
	fixed := make([]bool, days)
	shoulder := Zero
	shoulderDays := 0

	// The opening shoulder runs forward from day one.
	at := 0
	for _, step := range p.RampUp {
		for i := 0; i < step.Days && at < days; i++ {
			daily[at], fixed[at] = step.Rate, true
			shoulder = shoulder.Add(step.Rate)
			shoulderDays++
			at++
		}
	}

	// The closing shoulder runs backward from the last day, so the profile is
	// written in the order a planner reads it - 15,000 then 12,000 then 8,000 -
	// rather than reversed to suit the loop.
	end := days - 1
	for i := len(p.RunDown) - 1; i >= 0; i-- {
		step := p.RunDown[i]
		for j := 0; j < step.Days && end >= 0; j++ {
			if fixed[end] {
				return CrushingPlan{}, fmt.Errorf(
					"%w: the ramp up and the run down overlap in a %d-day campaign", ErrValidation, days)
			}
			daily[end], fixed[end] = step.Rate, true
			shoulder = shoulder.Add(step.Rate)
			shoulderDays++
			end--
		}
	}

	// Wash-outs, and the wind-down before each. A wash-out inside a shoulder is
	// left alone: the shoulder rate is what the planner asked for on that day,
	// and a stop already declared twice is still one stop.
	cleaning := p.cleaningSet(days)
	cleaningDays := 0
	for day := range cleaning {
		i := day - 1
		if fixed[i] {
			continue
		}
		daily[i], fixed[i] = Zero, true
		cleaningDays++
		if i > 0 && !fixed[i-1] && p.PreCleaningRate.GreaterThan(Zero) {
			daily[i-1], fixed[i-1] = p.PreCleaningRate, true
			shoulder = shoulder.Add(p.PreCleaningRate)
			shoulderDays++
		}
	}

	// Whatever is left is the plateau.
	var plateau []int
	for i := range daily {
		if !fixed[i] {
			plateau = append(plateau, i)
		}
	}
	if len(plateau) == 0 {
		if !shoulder.Equal(target) {
			return CrushingPlan{}, fmt.Errorf(
				"%w: the profile fixes every day of the campaign at %s t, which is not the %s t target",
				ErrValidation, shoulder, target)
		}
		return CrushingPlan{Daily: daily, CleaningDays: cleaningDays, ShoulderDays: shoulderDays}, nil
	}

	remaining := target.Sub(shoulder)
	if remaining.IsNegative() {
		return CrushingPlan{}, fmt.Errorf(
			"%w: the ramp up, the run down and the wash-outs already come to %s t, more than the %s t target",
			ErrValidation, shoulder, target)
	}

	rate := RoundQty(remaining.Div(DI(int64(len(plateau)))))
	running := Zero
	for n, i := range plateau {
		v := rate
		if n == len(plateau)-1 {
			// The last plateau day absorbs the rounding, so the season adds up.
			v = RoundQty(remaining.Sub(running))
		}
		daily[i] = v
		running = running.Add(v)
	}

	sort.Ints(plateau)
	return CrushingPlan{
		Daily:        daily,
		PlateauRate:  rate,
		PlateauDays:  len(plateau),
		CleaningDays: cleaningDays,
		ShoulderDays: shoulderDays,
	}, nil
}

// Validate checks a profile on its own, before it meets a target.
func (p CrushingProfile) Validate() error {
	verr := &ValidationError{}
	for i, s := range p.RampUp {
		if s.Days <= 0 {
			verr.Add(fmt.Sprintf("rampUp[%d].days", i), "INVALID", "a step lasts at least one day")
		}
		if s.Rate.IsNegative() {
			verr.Add(fmt.Sprintf("rampUp[%d].rateTons", i), "NEGATIVE", "a crushing rate cannot be negative")
		}
	}
	for i, s := range p.RunDown {
		if s.Days <= 0 {
			verr.Add(fmt.Sprintf("runDown[%d].days", i), "INVALID", "a step lasts at least one day")
		}
		if s.Rate.IsNegative() {
			verr.Add(fmt.Sprintf("runDown[%d].rateTons", i), "NEGATIVE", "a crushing rate cannot be negative")
		}
	}
	if p.CleaningEveryDays < 0 {
		verr.Add("cleaningEveryDays", "INVALID", "a wash-out cadence cannot be negative")
	}
	if p.PreCleaningRate.IsNegative() {
		verr.Add("preCleaningRateTons", "NEGATIVE", "a crushing rate cannot be negative")
	}
	if verr.HasErrors() {
		return verr
	}
	return nil
}
