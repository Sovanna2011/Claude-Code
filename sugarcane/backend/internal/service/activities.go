package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/sovanna2011/sugarcane-go/backend/internal/database"
	"github.com/sovanna2011/sugarcane-go/backend/internal/domain"
	"github.com/sovanna2011/sugarcane-go/backend/internal/repository"
)

// ActivityService is the planning master data: growing seasons, cane varieties, and the planting
// activity chain the schedule is generated from.
type ActivityService struct {
	db    *database.DB
	repo  *repository.ActivityRepository
	audit *repository.SupportRepository
}

// ---------------------------------------------------------------- seasons

func (s *ActivityService) ListSeasons(ctx context.Context, f domain.Filter, p domain.Page) (domain.PagedResult[domain.Season], error) {
	p = p.Normalise()
	items, total, err := s.repo.ListSeasons(ctx, f, p)
	if err != nil {
		return domain.PagedResult[domain.Season]{}, err
	}
	return domain.NewPagedResult(items, p, total), nil
}

func (s *ActivityService) GetSeason(ctx context.Context, id int) (domain.Season, error) {
	return s.repo.GetSeason(ctx, nil, id)
}

func (s *ActivityService) SaveSeason(ctx context.Context, u domain.User, remote string, id *int, in domain.SeasonInput) (domain.Season, error) {
	if err := requireRole(u, domain.RoleAdmin, domain.RoleManager); err != nil {
		return domain.Season{}, err
	}
	if err := requireText("code", in.Code, 20); err != nil {
		return domain.Season{}, err
	}
	if err := requireText("name", in.Name, 150); err != nil {
		return domain.Season{}, err
	}
	if in.Status == "" {
		in.Status = "Open"
	}
	if !domain.IsOneOf(in.Status, domain.ValidSeasonStatuses) {
		return domain.Season{}, domain.BadRequest("INVALID_STATUS",
			"A season's status must be one of "+strings.Join(domain.ValidSeasonStatuses, ", ")+".")
	}
	if in.CropYear < 1900 || in.CropYear > 2200 {
		return domain.Season{}, domain.Invalid([]domain.FieldError{{
			Field: "cropYear", Message: "The crop year does not look like a year."}})
	}

	var saved domain.Season
	err := s.db.InTx(ctx, func(tx database.Querier) error {
		action, before := "Create", any(nil)
		if id != nil {
			action = "Update"
			existing, err := s.repo.GetSeason(ctx, tx, *id)
			if err != nil {
				return err
			}
			before = existing
		}
		newID, err := s.repo.SaveSeason(ctx, tx, id, in)
		if err != nil {
			return err
		}
		after, err := s.repo.GetSeason(ctx, tx, newID)
		if err != nil {
			return err
		}
		saved = after
		return s.audit.Write(ctx, tx, u.Username, action, "Season", fmt.Sprint(newID), before, after, remote)
	})
	return saved, err
}

// ---------------------------------------------------------------- varieties

func (s *ActivityService) ListVarieties(ctx context.Context, f domain.Filter, p domain.Page) (domain.PagedResult[domain.Variety], error) {
	p = p.Normalise()
	items, total, err := s.repo.ListVarieties(ctx, f, p)
	if err != nil {
		return domain.PagedResult[domain.Variety]{}, err
	}
	return domain.NewPagedResult(items, p, total), nil
}

func (s *ActivityService) GetVariety(ctx context.Context, id int) (domain.Variety, error) {
	return s.repo.GetVariety(ctx, nil, id)
}

func (s *ActivityService) SaveVariety(ctx context.Context, u domain.User, remote string, id *int, in domain.VarietyInput) (domain.Variety, error) {
	if err := requireRole(u, domain.RoleAdmin, domain.RoleManager); err != nil {
		return domain.Variety{}, err
	}
	if err := requireText("code", in.Code, 20); err != nil {
		return domain.Variety{}, err
	}
	if err := requireText("name", in.Name, 150); err != nil {
		return domain.Variety{}, err
	}

	// These three figures are what the projection formulas read; a variety with a nonsense value
	// silently corrupts every projection that uses it, so it is refused here as well as by the
	// check constraints.
	var problems []domain.FieldError
	if in.SeedRatePerHa <= 0 {
		problems = append(problems, domain.FieldError{
			Field: "seedRatePerHa", Message: "The seed rate must be greater than zero."})
	}
	if in.ExpectedYieldPerHa < 0 {
		problems = append(problems, domain.FieldError{
			Field: "expectedYieldPerHa", Message: "The expected yield cannot be negative."})
	}
	if in.ExpectedLossPercent < 0 || in.ExpectedLossPercent > 100 {
		problems = append(problems, domain.FieldError{
			Field: "expectedLossPercent", Message: "The expected loss must be between 0 and 100 per cent."})
	}
	if in.GrowingPeriodMonths < 1 || in.GrowingPeriodMonths > 36 {
		problems = append(problems, domain.FieldError{
			Field: "growingPeriodMonths", Message: "The growing period must be between 1 and 36 months."})
	}
	if len(problems) > 0 {
		return domain.Variety{}, domain.Invalid(problems)
	}

	var saved domain.Variety
	err := s.db.InTx(ctx, func(tx database.Querier) error {
		action, before := "Create", any(nil)
		if id != nil {
			action = "Update"
			existing, err := s.repo.GetVariety(ctx, tx, *id)
			if err != nil {
				return err
			}
			before = existing
		}
		newID, err := s.repo.SaveVariety(ctx, tx, id, in)
		if err != nil {
			return err
		}
		after, err := s.repo.GetVariety(ctx, tx, newID)
		if err != nil {
			return err
		}
		saved = after
		return s.audit.Write(ctx, tx, u.Username, action, "Variety", fmt.Sprint(newID), before, after, remote)
	})
	return saved, err
}

// ---------------------------------------------------------------- activities

func (s *ActivityService) ListActivities(ctx context.Context, f domain.Filter, p domain.Page) (domain.PagedResult[domain.Activity], error) {
	p = p.Normalise()
	items, total, err := s.repo.ListActivities(ctx, f, p)
	if err != nil {
		return domain.PagedResult[domain.Activity]{}, err
	}
	return domain.NewPagedResult(items, p, total), nil
}

func (s *ActivityService) GetActivity(ctx context.Context, id int) (domain.Activity, error) {
	return s.repo.GetActivity(ctx, nil, id)
}

func (s *ActivityService) SaveActivity(ctx context.Context, u domain.User, remote string, id *int, in domain.ActivityInput) (domain.Activity, error) {
	if err := requireRole(u, domain.RoleAdmin, domain.RoleManager); err != nil {
		return domain.Activity{}, err
	}
	if err := requireText("code", in.Code, 20); err != nil {
		return domain.Activity{}, err
	}
	if err := requireText("name", in.Name, 150); err != nil {
		return domain.Activity{}, err
	}

	category, ok := domain.Canonical(in.Category, domain.ValidActivityCategories)
	if !ok {
		return domain.Activity{}, domain.BadRequest("INVALID_CATEGORY",
			"The activity category must be one of "+strings.Join(domain.ValidActivityCategories, ", ")+".")
	}
	in.Category = category

	if in.ApplicableCropType == "" {
		in.ApplicableCropType = "Both"
	}
	cropType, ok := domain.Canonical(in.ApplicableCropType, domain.ValidApplicableCropTypes)
	if !ok {
		return domain.Activity{}, domain.BadRequest("INVALID_CROP_TYPE",
			"The applicable crop type must be NewPlanting, Ratoon or Both.")
	}
	in.ApplicableCropType = cropType

	var problems []domain.FieldError
	if in.SequenceNo <= 0 {
		problems = append(problems, domain.FieldError{
			Field: "sequenceNo", Message: "The sequence number must be greater than zero."})
	}
	// The daily capacity is the divisor in every duration and machine calculation. An activity the
	// engine has to schedule but cannot size would produce a plan with no duration, so the rule is
	// enforced here and again as a check constraint.
	if (in.RequiresTractor || in.RequiresEquipment) && in.StandardCapacityPerDay <= 0 {
		problems = append(problems, domain.FieldError{
			Field: "standardCapacityPerDay",
			Message: "An activity that needs a tractor or an implement must have a daily capacity, " +
				"or the engine cannot work out how long it takes."})
	}
	if in.RequiresLabour && in.StandardLabourDaysPerHa <= 0 {
		problems = append(problems, domain.FieldError{
			Field:   "standardLabourDaysPerHa",
			Message: "An activity that needs labour must say how many labour-days a hectare takes."})
	}
	if in.StandardCapacityPerDay < 0 || in.StandardCapacityPerHour < 0 ||
		in.StandardDurationPerHa < 0 || in.StandardLabourDaysPerHa < 0 {
		problems = append(problems, domain.FieldError{
			Field: "standardCapacityPerDay", Message: "Capacities and durations cannot be negative."})
	}
	if len(problems) > 0 {
		return domain.Activity{}, domain.Invalid(problems)
	}

	var saved domain.Activity
	err := s.db.InTx(ctx, func(tx database.Querier) error {
		action, before := "Create", any(nil)
		if id != nil {
			action = "Update"
			existing, err := s.repo.GetActivity(ctx, tx, *id)
			if err != nil {
				return err
			}
			before = existing
		}
		newID, err := s.repo.SaveActivity(ctx, tx, id, in, u.Username)
		if err != nil {
			return err
		}
		after, err := s.repo.GetActivity(ctx, tx, newID)
		if err != nil {
			return err
		}
		saved = after
		return s.audit.Write(ctx, tx, u.Username, action, "Activity", fmt.Sprint(newID), before, after, remote)
	})
	return saved, err
}

// ---------------------------------------------------------------- dependencies

func (s *ActivityService) AddDependency(ctx context.Context, u domain.User, remote string, in domain.DependencyInput) (domain.Activity, error) {
	if err := requireRole(u, domain.RoleAdmin, domain.RoleManager); err != nil {
		return domain.Activity{}, err
	}
	if in.ActivityID == in.DependsOnID {
		return domain.Activity{}, domain.Invalid([]domain.FieldError{{
			Field: "dependsOnId", Message: "An activity cannot wait for itself."}})
	}
	if in.LagDays < 0 {
		return domain.Activity{}, domain.Invalid([]domain.FieldError{{
			Field: "lagDays", Message: "A lag cannot be negative; an activity cannot start before the one it waits for."}})
	}

	var saved domain.Activity
	err := s.db.InTx(ctx, func(tx database.Querier) error {
		// The database refuses a cycle at commit; asking first turns that into a message naming
		// the two activities rather than a constraint violation surfacing from the commit.
		cyclic, err := s.wouldCycle(ctx, tx, in.ActivityID, in.DependsOnID)
		if err != nil {
			return err
		}
		if cyclic {
			from, _ := s.repo.GetActivity(ctx, tx, in.ActivityID)
			to, _ := s.repo.GetActivity(ctx, tx, in.DependsOnID)
			return domain.Invalid([]domain.FieldError{{
				Field: "dependsOnId",
				Message: fmt.Sprintf("%s already comes after %s, so this dependency would close a loop.",
					to.Code, from.Code)}})
		}

		id, err := s.repo.AddDependency(ctx, tx, in)
		if err != nil {
			return err
		}
		after, err := s.repo.GetActivity(ctx, tx, in.ActivityID)
		if err != nil {
			return err
		}
		saved = after
		return s.audit.Write(ctx, tx, u.Username, "AddDependency", "ActivityDependency",
			fmt.Sprint(id), nil, in, remote)
	})
	return saved, err
}

// wouldCycle walks the chain of what dependsOn already waits for, looking for the activity that is
// about to wait for it.
func (s *ActivityService) wouldCycle(ctx context.Context, tx database.Querier, activityID, dependsOnID int) (bool, error) {
	var cyclic bool
	err := tx.QueryRow(ctx, `
		WITH RECURSIVE chain(activity_id, depth) AS (
		    SELECT $2::int, 1
		     UNION ALL
		    SELECT d.depends_on_id, chain.depth + 1
		      FROM activity_dependency d
		      JOIN chain ON chain.activity_id = d.activity_id
		     WHERE chain.depth < 200
		)
		SELECT EXISTS (SELECT 1 FROM chain WHERE activity_id = $1)`, activityID, dependsOnID).Scan(&cyclic)
	return cyclic, err
}

func (s *ActivityService) DeleteDependency(ctx context.Context, u domain.User, remote string, id int) error {
	if err := requireRole(u, domain.RoleAdmin, domain.RoleManager); err != nil {
		return err
	}
	return s.db.InTx(ctx, func(tx database.Querier) error {
		if err := s.repo.DeleteDependency(ctx, tx, id); err != nil {
			return err
		}
		return s.audit.Write(ctx, tx, u.Username, "DeleteDependency", "ActivityDependency",
			fmt.Sprint(id), nil, nil, remote)
	})
}

// Chain returns the activity master in sequence for one crop type — the order the engine walks.
func (s *ActivityService) Chain(ctx context.Context, companyID int, cropType string) ([]domain.Activity, error) {
	if cropType != "" {
		canonical, ok := domain.Canonical(cropType, domain.ValidApplicableCropTypes)
		if !ok {
			return nil, domain.BadRequest("INVALID_CROP_TYPE",
				"The crop type must be NewPlanting, Ratoon or Both.")
		}
		cropType = canonical
	}
	return s.repo.Chain(ctx, companyID, cropType)
}
