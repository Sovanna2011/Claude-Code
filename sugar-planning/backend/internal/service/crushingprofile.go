package service

import (
	"context"
	"fmt"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/store"
)

// The shape of a crushing season.
//
// A season plan is not a straight line, and until the reference workbook was
// supplied the generator could only draw one. This is where a planner says how
// the campaign actually runs: the start-up ramp, how often the mill washes out,
// and how the cane thins at the end. The full rate is not entered - it is
// solved from the target - because that is the one figure a spreadsheet gets
// wrong the moment the tonnage changes.

// CrushingProfileView is the stored profile with the curve it produces, so a
// screen can show the plan the settings imply before anybody generates it.
type CrushingProfileView struct {
	VersionID string                 `json:"versionId"`
	Profile   domain.CrushingProfile `json:"profile"`
	// Preview is the campaign this profile and the version's own cane target
	// would produce. It is calculated, never stored: a preview that could
	// disagree with the generator would be worse than no preview.
	Preview CrushingPreview `json:"preview"`
}

// CrushingPreview is the shape a profile implies, summarised.
type CrushingPreview struct {
	CampaignDays int        `json:"campaignDays"`
	CrushingDays int        `json:"crushingDays"`
	CleaningDays int        `json:"cleaningDays"`
	PlateauTons  domain.Dec `json:"plateauRateTons"`
	TotalTons    domain.Dec `json:"totalTons"`
	// Daily is the whole curve, one tonnage per campaign day.
	Daily []domain.Dec `json:"daily"`
}

// CrushingProfile reads a version's campaign shape and the curve it makes.
func (p *Planning) CrushingProfile(ctx context.Context, versionID string) (CrushingProfileView, error) {
	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanRead); err != nil {
		return CrushingProfileView{}, err
	}
	if _, err := p.versionInScope(ctx, versionID); err != nil {
		return CrushingProfileView{}, err
	}
	profile, err := p.loadProfile(ctx, versionID)
	if err != nil {
		return CrushingProfileView{}, err
	}
	out := CrushingProfileView{VersionID: versionID, Profile: profile}
	out.Preview, err = p.previewProfile(ctx, versionID, profile)
	if err != nil {
		return CrushingProfileView{}, err
	}
	return out, nil
}

// SaveCrushingProfile replaces a version's campaign shape.
//
// It is refused if it cannot produce a plan: a profile whose shoulders already
// exceed the season target is not a shape, it is a mistake, and storing it
// would leave a version that can be edited but never generated.
func (p *Planning) SaveCrushingProfile(ctx context.Context, versionID string,
	profile domain.CrushingProfile) (CrushingProfileView, error) {

	caller := auth.FromContext(ctx)
	if err := caller.Require(domain.PermPlanWrite); err != nil {
		return CrushingProfileView{}, err
	}
	version, err := p.versionInScope(ctx, versionID)
	if err != nil {
		return CrushingProfileView{}, err
	}
	if !version.IsEditable() {
		return CrushingProfileView{}, fmt.Errorf("%w: version %s is %s",
			domain.ErrLocked, version.Code, version.Status)
	}
	if err := profile.Validate(); err != nil {
		return CrushingProfileView{}, err
	}

	// Checked against this version's own target before it is stored, so the
	// refusal names the real problem rather than surfacing later as a failed
	// generate.
	preview, err := p.previewProfile(ctx, versionID, profile)
	if err != nil {
		return CrushingProfileView{}, err
	}

	err = p.store.InTx(ctx, func(tx store.Store) error {
		if err := tx.Planning().ReplaceCrushingSteps(ctx, versionID,
			domain.RowsFromProfile(versionID, profile), caller.Username); err != nil {
			return err
		}
		// The two scalars are assumptions, because that is what they are: one
		// number each, effective-dated, and shown beside the recovery and the
		// quota rate a planner is already reading.
		for _, a := range []domain.PlanAssumption{
			{Code: domain.AsmCleaningEveryDays, Description: "Wash-out cadence",
				Value: domain.DI(int64(profile.CleaningEveryDays)), UOM: "DAY"},
			{Code: domain.AsmPreCleaningRate, Description: "Crushing rate the day before a wash-out",
				Value: profile.PreCleaningRate, UOM: "TON/DAY"},
		} {
			a.VersionID = versionID
			a.ValidFrom = version.EffectiveFrom
			if _, err := tx.Planning().SaveAssumption(ctx, a, caller.Username); err != nil {
				return err
			}
		}
		return writeAudit(ctx, tx, p.now, auditEntry{
			action: "SAVE_CRUSHING_PROFILE", entity: "plan_version", entityID: versionID,
			after: profile,
		})
	})
	if err != nil {
		return CrushingProfileView{}, err
	}
	return CrushingProfileView{VersionID: versionID, Profile: profile, Preview: preview}, nil
}

// loadProfile reads the stored steps and the two scalar assumptions back into a
// profile.
func (p *Planning) loadProfile(ctx context.Context, versionID string) (domain.CrushingProfile, error) {
	steps, err := p.store.Planning().ListCrushingSteps(ctx, versionID)
	if err != nil {
		return domain.CrushingProfile{}, err
	}
	stored, err := p.store.Planning().ListAssumptions(ctx, versionID)
	if err != nil {
		return domain.CrushingProfile{}, err
	}
	values := map[string]domain.Dec{}
	for _, a := range stored {
		values[a.Code] = a.Value
	}
	return domain.ProfileFromRows(steps,
		int(values[domain.AsmCleaningEveryDays].IntPart()),
		values[domain.AsmPreCleaningRate]), nil
}

// previewProfile lays a profile over the version's own target and days.
func (p *Planning) previewProfile(ctx context.Context, versionID string,
	profile domain.CrushingProfile) (CrushingPreview, error) {

	stored, err := p.store.Planning().ListAssumptions(ctx, versionID)
	if err != nil {
		return CrushingPreview{}, err
	}
	values := map[string]domain.Dec{}
	for _, a := range stored {
		values[a.Code] = a.Value
	}
	days := int(values[domain.AsmSeasonDays].IntPart())
	if days <= 0 {
		// Nothing to lay the profile over yet. Not an error: assumptions and
		// the profile can be entered in either order.
		return CrushingPreview{}, nil
	}
	plan, err := domain.BuildCrushingPlan(values[domain.AsmCaneTarget], days, profile)
	if err != nil {
		return CrushingPreview{}, err
	}
	total := domain.Zero
	for _, v := range plan.Daily {
		total = total.Add(v)
	}
	return CrushingPreview{
		CampaignDays: days,
		CrushingDays: plan.PlateauDays + plan.ShoulderDays,
		CleaningDays: plan.CleaningDays,
		PlateauTons:  plan.PlateauRate,
		TotalTons:    total,
		Daily:        plan.Daily,
	}, nil
}
