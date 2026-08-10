package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sovanna2011/sugarcane-go/backend/internal/database"
	"github.com/sovanna2011/sugarcane-go/backend/internal/domain"
	"github.com/sovanna2011/sugarcane-go/backend/internal/repository"
)

// ProjectionService is the planting projection and its approval workflow. It is the module every
// later one reads: an approved projection is what the activity plan, the machinery schedule and the
// material requirement are generated from.
type ProjectionService struct {
	db    *database.DB
	repo  *repository.ProjectionRepository
	audit *repository.SupportRepository
}

const isoDate = "2006-01-02"

func (s *ProjectionService) List(ctx context.Context, f domain.Filter, p domain.Page, u domain.User) (domain.PagedResult[domain.Projection], error) {
	p = p.Normalise()
	items, total, err := s.repo.ListProjections(ctx, f, p)
	if err != nil {
		return domain.PagedResult[domain.Projection]{}, err
	}
	for i := range items {
		s.decorate(&items[i], u)
	}
	return domain.NewPagedResult(items, p, total), nil
}

// Get returns the header with its lines and its trail — everything the detail screen shows in one
// round trip, because a projection is read as a whole or not at all.
func (s *ProjectionService) Get(ctx context.Context, id int, u domain.User) (domain.Projection, error) {
	p, err := s.repo.GetProjection(ctx, nil, id)
	if err != nil {
		return domain.Projection{}, err
	}
	if p.Lines, err = s.repo.Lines(ctx, nil, id); err != nil {
		return domain.Projection{}, err
	}
	if p.History, err = s.repo.History(ctx, nil, id); err != nil {
		return domain.Projection{}, err
	}
	s.decorate(&p, u)
	return p, nil
}

// decorate fills in what this caller may do next. It is answered from the same graph the workflow
// endpoints enforce, so the buttons a screen draws are exactly the ones that will be honoured.
func (s *ProjectionService) decorate(p *domain.Projection, u domain.User) {
	p.Actions = domain.AllowedActions(p.Status, u)
	p.LinesEditable = domain.LinesEditable(p.Status) &&
		u.HasAnyRole(domain.RoleAdmin, domain.RoleManager, domain.RolePlanner)
}

// ---------------------------------------------------------------- the header

func (s *ProjectionService) Save(ctx context.Context, u domain.User, remote string, id *int, in domain.ProjectionInput) (domain.Projection, error) {
	if err := requireRole(u, domain.RoleAdmin, domain.RoleManager, domain.RolePlanner); err != nil {
		return domain.Projection{}, err
	}
	if err := requireText("projectionNo", in.ProjectionNo, 40); err != nil {
		return domain.Projection{}, err
	}
	in.ProjectionNo = strings.TrimSpace(in.ProjectionNo)

	projectionDate, err := requireDate("projectionDate", in.ProjectionDate)
	if err != nil {
		return domain.Projection{}, err
	}
	start, err := requireDate("planningStart", in.PlanningStart)
	if err != nil {
		return domain.Projection{}, err
	}
	end, err := requireDate("planningEnd", in.PlanningEnd)
	if err != nil {
		return domain.Projection{}, err
	}
	if end.Before(start) {
		return domain.Projection{}, domain.Invalid([]domain.FieldError{{
			Field: "planningEnd", Message: "The planning window cannot end before it starts."}})
	}
	if projectionDate.After(end) {
		return domain.Projection{}, domain.Invalid([]domain.FieldError{{
			Field: "projectionDate", Message: "A plan cannot be dated after the window it plans."}})
	}

	var saved domain.Projection
	err = s.inTx(ctx, func(tx database.Querier) error {
		action, before := "Create", any(nil)
		if id != nil {
			existing, err := s.repo.GetProjection(ctx, tx, *id)
			if err != nil {
				return err
			}
			// The header of a decided plan is as committed as its lines. Changing the season or the
			// window under an approved plan would silently move work other modules have scheduled.
			if !domain.LinesEditable(existing.Status) {
				return domain.Conflict("NOT_EDITABLE", fmt.Sprintf(
					"Projection %s is %s and can no longer be changed. Open a revision instead.",
					existing.ProjectionNo, existing.Status))
			}
			action, before = "Update", existing
		}

		newID, err := s.repo.SaveProjection(ctx, tx, id, in, u.Username)
		if err != nil {
			return err
		}
		after, err := s.repo.GetProjection(ctx, tx, newID)
		if err != nil {
			return err
		}
		saved = after
		return s.audit.Write(ctx, tx, u.Username, action, "Projection", fmt.Sprint(newID), before, after, remote)
	})
	if err != nil {
		return domain.Projection{}, err
	}
	s.decorate(&saved, u)
	return saved, nil
}

// ---------------------------------------------------------------- the lines

// SaveLine writes one block's share of the plan. Everything the caller does not need to decide is
// settled here: the available-area snapshot comes from the block, the yield and loss default to the
// variety's, and the expected harvest date is dated from the growing period.
func (s *ProjectionService) SaveLine(ctx context.Context, u domain.User, remote string,
	projectionID int, lineID *int, in domain.ProjectionLineInput) (domain.Projection, error) {

	if err := requireRole(u, domain.RoleAdmin, domain.RoleManager, domain.RolePlanner); err != nil {
		return domain.Projection{}, err
	}

	if in.PlantingType == "" {
		in.PlantingType = "NewPlanting"
	}
	plantingType, ok := domain.Canonical(in.PlantingType, domain.ValidPlantingTypes)
	if !ok {
		return domain.Projection{}, domain.BadRequest("INVALID_PLANTING_TYPE",
			"A line is either a NewPlanting or a Ratoon.")
	}

	start, err := requireDate("plannedPlantingStart", in.PlannedPlantingStart)
	if err != nil {
		return domain.Projection{}, err
	}
	end, err := requireDate("plannedPlantingEnd", in.PlannedPlantingEnd)
	if err != nil {
		return domain.Projection{}, err
	}
	if end.Before(start) {
		return domain.Projection{}, domain.Invalid([]domain.FieldError{{
			Field: "plannedPlantingEnd", Message: "Planting cannot end before it starts."}})
	}
	if in.ProjectedPlantingAreaHa <= 0 {
		return domain.Projection{}, domain.Invalid([]domain.FieldError{{
			Field: "projectedPlantingAreaHa", Message: "The projected area must be greater than zero."}})
	}

	var saved domain.Projection
	err = s.inTx(ctx, func(tx database.Querier) error {
		header, err := s.repo.GetProjection(ctx, tx, projectionID)
		if err != nil {
			return err
		}
		if !domain.LinesEditable(header.Status) {
			return domain.Conflict("NOT_EDITABLE", fmt.Sprintf(
				"Projection %s is %s; its lines can no longer be changed. Open a revision instead.",
				header.ProjectionNo, header.Status))
		}

		block, err := s.repo.BlockFacts(ctx, tx, in.BlockID)
		if err != nil {
			return err
		}
		variety, err := s.repo.VarietyDefaults(ctx, tx, in.CaneVarietyID)
		if err != nil {
			return err
		}

		values := repository.LineValues{
			BlockID:       in.BlockID,
			CaneVarietyID: in.CaneVarietyID,
			PlantingType:  plantingType,
			AvailableHa:   block.PlantableHa,
			ProjectedHa:   in.ProjectedPlantingAreaHa,
			Start:         start.Format(isoDate),
			End:           end.Format(isoDate),
			YieldPerHa:    variety.YieldPerHa,
			LossPercent:   variety.LossPercent,
			Priority:      5,
			Remark:        in.Remark,
		}
		if in.ExpectedYieldPerHa != nil {
			values.YieldPerHa = *in.ExpectedYieldPerHa
		}
		if in.ExpectedLossPercent != nil {
			values.LossPercent = *in.ExpectedLossPercent
		}
		if in.Priority != nil {
			values.Priority = *in.Priority
		}

		// The line must fit the block, and so must every line this plan has on that block put
		// together — a new planting and a ratoon share the same ground. The database enforces both,
		// the second as a deferred trigger; checking here names the block in the message instead of
		// quoting a constraint at commit time.
		if in.ProjectedPlantingAreaHa > block.PlantableHa {
			return domain.Invalid([]domain.FieldError{{
				Field: "projectedPlantingAreaHa",
				Message: fmt.Sprintf("Block %s has %.4f ha that can carry cane; %.4f ha was projected.",
					block.Code, block.PlantableHa, in.ProjectedPlantingAreaHa)}})
		}
		others, err := s.repo.CommittedAreaOnBlock(ctx, tx, projectionID, in.BlockID, lineID)
		if err != nil {
			return err
		}
		if total := others + in.ProjectedPlantingAreaHa; total > block.PlantableHa {
			return domain.Invalid([]domain.FieldError{{
				Field: "projectedPlantingAreaHa",
				Message: fmt.Sprintf(
					"This plan already has %.4f ha on block %s; a further %.4f ha would need %.4f ha of the %.4f ha that can carry cane.",
					others, block.Code, in.ProjectedPlantingAreaHa, total, block.PlantableHa)}})
		}

		// Planting has to happen inside the season's window, when the season declares one — that is
		// what the window is for, and a line outside it would schedule work nobody can do.
		if err := s.checkSeasonWindow(ctx, tx, header.CropSeasonID, block.Code, start, end); err != nil {
			return err
		}

		// And inside the plan's own window, which the planner set on the header.
		if err := withinPlanningWindow(header, block.Code, start, end); err != nil {
			return err
		}

		values.HarvestDate, err = harvestDate(in.ExpectedHarvestDate, end, variety.GrowingPeriodMonths)
		if err != nil {
			return err
		}

		action, before := "CreateLine", any(nil)
		if lineID != nil {
			existing, err := s.repo.GetLine(ctx, tx, *lineID)
			if err != nil {
				return err
			}
			if existing.ProjectionID != projectionID {
				return domain.NotFound("Projection line", *lineID)
			}
			action, before = "UpdateLine", existing
		}

		newID, err := s.repo.SaveLine(ctx, tx, projectionID, lineID, values)
		if err != nil {
			return err
		}
		after, err := s.repo.GetLine(ctx, tx, newID)
		if err != nil {
			return err
		}
		if err := s.audit.Write(ctx, tx, u.Username, action, "ProjectionLine",
			fmt.Sprint(newID), before, after, remote); err != nil {
			return err
		}
		saved, err = s.reload(ctx, tx, projectionID)
		return err
	})
	if err != nil {
		return domain.Projection{}, err
	}
	s.decorate(&saved, u)
	return saved, nil
}

func (s *ProjectionService) DeleteLine(ctx context.Context, u domain.User, remote string, projectionID, lineID int) (domain.Projection, error) {
	if err := requireRole(u, domain.RoleAdmin, domain.RoleManager, domain.RolePlanner); err != nil {
		return domain.Projection{}, err
	}

	var saved domain.Projection
	err := s.inTx(ctx, func(tx database.Querier) error {
		header, err := s.repo.GetProjection(ctx, tx, projectionID)
		if err != nil {
			return err
		}
		if !domain.LinesEditable(header.Status) {
			return domain.Conflict("NOT_EDITABLE", fmt.Sprintf(
				"Projection %s is %s; its lines can no longer be changed.", header.ProjectionNo, header.Status))
		}
		before, err := s.repo.GetLine(ctx, tx, lineID)
		if err != nil {
			return err
		}
		if err := s.repo.DeleteLine(ctx, tx, projectionID, lineID); err != nil {
			return err
		}
		if err := s.audit.Write(ctx, tx, u.Username, "DeleteLine", "ProjectionLine",
			fmt.Sprint(lineID), before, nil, remote); err != nil {
			return err
		}
		saved, err = s.reload(ctx, tx, projectionID)
		return err
	})
	if err != nil {
		return domain.Projection{}, err
	}
	s.decorate(&saved, u)
	return saved, nil
}

// ---------------------------------------------------------------- the workflow

// Act takes a projection through one step of the workflow. Every step is the same shape — check the
// edge is legal for this status and this caller, apply it, record it — which is why there is one
// method rather than six.
func (s *ProjectionService) Act(ctx context.Context, u domain.User, remote string,
	id int, action string, req domain.WorkflowRequest) (domain.Projection, error) {

	var saved domain.Projection
	err := s.inTx(ctx, func(tx database.Querier) error {
		before, err := s.repo.GetProjection(ctx, tx, id)
		if err != nil {
			return err
		}

		transition, err := domain.CheckTransition(action, before.Status, u)
		if err != nil {
			return err
		}
		comments := strings.TrimSpace(req.Comments)
		if transition.NeedsReason && comments == "" {
			return domain.Invalid([]domain.FieldError{{
				Field:   "comments",
				Message: fmt.Sprintf("Say why: a projection cannot be %sd without a reason.", transition.Action)}})
		}
		// The row version the caller read. Zero means they did not send one, which is only safe on
		// a workflow step because the status check above already caught a plan that has moved on.
		rowVersion := req.Version
		if rowVersion == 0 {
			rowVersion = before.Version
		}

		var comment *string
		if comments != "" {
			comment = &comments
		}

		switch transition.Action {
		case domain.ActionSubmit:
			if before.LineCount == 0 {
				return domain.Invalid([]domain.FieldError{{
					Field:   "lines",
					Message: "A projection with no lines has nothing to approve; add at least one block."}})
			}
		case domain.ActionApprove:
			// The moment that commits the land. Checked here rather than on every line write,
			// because two planners drafting alternatives for the same block is normal — only one
			// of them may be approved.
			if err := s.repo.AssertNoCommittedOverlap(ctx, tx, id); err != nil {
				return err
			}
		}

		if err := s.repo.ApplyTransition(ctx, tx, id, before.Status, transition.To,
			transition.Action, u.Username, comment, rowVersion); err != nil {
			return err
		}

		// A revision is the one step that produces a second row: the decided plan becomes Revised
		// and a fresh Draft carrying a copy of its lines takes its place as the current version.
		target := id
		if transition.Action == domain.ActionRevise {
			newID, err := s.repo.CreateRevision(ctx, tx, id, u.Username)
			if err != nil {
				return err
			}
			target = newID
		}

		after, err := s.repo.GetProjection(ctx, tx, target)
		if err != nil {
			return err
		}
		if err := s.audit.Write(ctx, tx, u.Username, transition.Action, "Projection",
			fmt.Sprint(id), before, after, remote); err != nil {
			return err
		}
		saved, err = s.reload(ctx, tx, target)
		return err
	})
	if err != nil {
		return domain.Projection{}, err
	}
	s.decorate(&saved, u)
	return saved, nil
}

// ---------------------------------------------------------------- helpers

// inTx runs the work and translates whatever comes back. The cross-row area rule is a deferred
// constraint trigger, so it fires at COMMIT — outside the repository call that caused it, and past
// every mapping the repository does. Without this, breaking that rule would be a 500.
func (s *ProjectionService) inTx(ctx context.Context, fn func(tx database.Querier) error) error {
	return repository.TranslateProjectionError(s.db.InTx(ctx, fn))
}

func (s *ProjectionService) reload(ctx context.Context, tx database.Querier, id int) (domain.Projection, error) {
	p, err := s.repo.GetProjection(ctx, tx, id)
	if err != nil {
		return domain.Projection{}, err
	}
	if p.Lines, err = s.repo.Lines(ctx, tx, id); err != nil {
		return domain.Projection{}, err
	}
	p.History, err = s.repo.History(ctx, tx, id)
	return p, err
}

func (s *ProjectionService) checkSeasonWindow(ctx context.Context, tx database.Querier,
	seasonID int, blockCode string, start, end time.Time) error {

	code, from, to, err := s.repo.SeasonWindow(ctx, tx, seasonID)
	if err != nil {
		return err
	}
	if from == nil || to == nil {
		return nil
	}
	windowStart, err := time.Parse(isoDate, *from)
	if err != nil {
		return err
	}
	windowEnd, err := time.Parse(isoDate, *to)
	if err != nil {
		return err
	}
	// A window that wraps the year end — October to February — is an ordinary season here, and the
	// planting dates of a plan for it can legitimately sit either side of the new year. Only a
	// window that does not wrap can be checked by a simple comparison.
	if windowEnd.Before(windowStart) {
		return nil
	}
	if start.Before(windowStart) || end.After(windowEnd) {
		return domain.Invalid([]domain.FieldError{{
			Field: "plannedPlantingStart",
			Message: fmt.Sprintf("Season %s plants between %s and %s; block %s was planned for %s to %s.",
				code, *from, *to, blockCode, start.Format(isoDate), end.Format(isoDate))}})
	}
	return nil
}

func withinPlanningWindow(header domain.Projection, blockCode string, start, end time.Time) error {
	planStart, err := time.Parse(isoDate, header.PlanningStart)
	if err != nil {
		return err
	}
	planEnd, err := time.Parse(isoDate, header.PlanningEnd)
	if err != nil {
		return err
	}
	if start.Before(planStart) || end.After(planEnd) {
		return domain.Invalid([]domain.FieldError{{
			Field: "plannedPlantingStart",
			Message: fmt.Sprintf("This plan covers %s to %s; block %s was planned for %s to %s.",
				header.PlanningStart, header.PlanningEnd, blockCode,
				start.Format(isoDate), end.Format(isoDate))}})
	}
	return nil
}

// harvestDate takes what the caller sent, or dates the harvest from the end of planting plus the
// variety's growing period when they left it out.
func harvestDate(given *string, plantingEnd time.Time, growingPeriodMonths int) (*string, error) {
	if given != nil && strings.TrimSpace(*given) != "" {
		parsed, err := time.Parse(isoDate, strings.TrimSpace(*given))
		if err != nil {
			return nil, domain.Invalid([]domain.FieldError{{
				Field: "expectedHarvestDate", Message: "Use the form YYYY-MM-DD."}})
		}
		if parsed.Before(plantingEnd) {
			return nil, domain.Invalid([]domain.FieldError{{
				Field: "expectedHarvestDate", Message: "Cane cannot be harvested before it is planted."}})
		}
		out := parsed.Format(isoDate)
		return &out, nil
	}
	if growingPeriodMonths <= 0 {
		return nil, nil
	}
	out := plantingEnd.AddDate(0, growingPeriodMonths, 0).Format(isoDate)
	return &out, nil
}

func requireDate(field, value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, domain.Invalid([]domain.FieldError{{
			Field: field, Message: "This date is required."}})
	}
	parsed, err := time.Parse(isoDate, trimmed)
	if err != nil {
		return time.Time{}, domain.Invalid([]domain.FieldError{{
			Field: field, Message: "Use the form YYYY-MM-DD."}})
	}
	return parsed, nil
}
