package service

import (
	"context"
	"fmt"

	"github.com/sovanna2011/sugarcane-go/backend/internal/database"
	"github.com/sovanna2011/sugarcane-go/backend/internal/domain"
	"github.com/sovanna2011/sugarcane-go/backend/internal/repository"
)

// PlanService turns an approved projection into a dated programme of work.
//
// It owns none of the arithmetic: the schedule comes from domain.GenerateTasks, which is a pure
// function this layer feeds and stores. What lives here is everything about *when* generating is
// allowed — which projections may be planned, and what a released plan will no longer accept.
type PlanService struct {
	db          *database.DB
	repo        *repository.PlanRepository
	projections *repository.ProjectionRepository
	audit       *repository.SupportRepository
}

func (s *PlanService) List(ctx context.Context, f domain.Filter, p domain.Page, u domain.User) (domain.PagedResult[domain.ActivityPlan], error) {
	p = p.Normalise()
	items, total, err := s.repo.ListPlans(ctx, f, p)
	if err != nil {
		return domain.PagedResult[domain.ActivityPlan]{}, err
	}
	for i := range items {
		s.decorate(&items[i], u)
	}
	return domain.NewPagedResult(items, p, total), nil
}

// Get returns the header with its whole programme. The filter narrows the tasks, so a block's own
// page can ask for one block's rows without a second endpoint.
func (s *PlanService) Get(ctx context.Context, id int, f domain.Filter, u domain.User) (domain.ActivityPlan, error) {
	plan, err := s.repo.GetPlan(ctx, nil, id)
	if err != nil {
		return domain.ActivityPlan{}, err
	}
	if plan.Tasks, err = s.repo.Tasks(ctx, nil, id, f); err != nil {
		return domain.ActivityPlan{}, err
	}
	s.decorate(&plan, u)
	return plan, nil
}

func (s *PlanService) decorate(p *domain.ActivityPlan, u domain.User) {
	p.Actions = domain.PlanActionsFor(p.Status, u)
	p.Editable = p.Status == domain.PlanDraft &&
		u.HasAnyRole(domain.RoleAdmin, domain.RoleManager, domain.RolePlanner)
}

// ---------------------------------------------------------------- generating

// Generate lays an approved projection out over the calendar. It creates the plan the first time
// and replaces the programme on every later call, because the engine is deterministic: regenerating
// after the projection or the calendar changed is the only way to keep the two in step.
func (s *PlanService) Generate(ctx context.Context, u domain.User, remote string, in domain.PlanInput) (domain.ActivityPlan, error) {
	if err := requireRole(u, domain.RoleAdmin, domain.RoleManager, domain.RolePlanner); err != nil {
		return domain.ActivityPlan{}, err
	}
	calendar, err := in.CalendarFrom()
	if err != nil {
		return domain.ActivityPlan{}, err
	}

	var saved domain.ActivityPlan
	err = s.db.InTx(ctx, func(tx database.Querier) error {
		projection, err := s.projections.GetProjection(ctx, tx, in.ProjectionID)
		if err != nil {
			return err
		}

		// Only an approved plan commits land, and only committed land is worth scheduling work
		// against. Generating from a draft would produce a programme for blocks another plan may
		// still take.
		if projection.Status != domain.ProjectionApproved {
			return domain.Conflict("PROJECTION_NOT_APPROVED", fmt.Sprintf(
				"Projection %s is %s. Only an approved projection can be laid out over the calendar.",
				projection.ProjectionNo, projection.Status))
		}

		lines, err := s.repo.PlanLines(ctx, tx, projection.ID)
		if err != nil {
			return err
		}
		if len(lines) == 0 {
			return domain.Invalid([]domain.FieldError{{
				Field: "projectionId", Message: "That projection has no blocks, so there is nothing to schedule."}})
		}

		activities, err := s.repo.PlanningActivities(ctx, tx, projection.CompanyID)
		if err != nil {
			return err
		}
		if len(activities) == 0 {
			return domain.Invalid([]domain.FieldError{{
				Field: "projectionId",
				Message: "No planting activities are defined for this company, so there is no programme to lay out."}})
		}

		// Existing plan, or a new one.
		plan, err := s.repo.PlanForProjection(ctx, tx, projection.ID)
		action := "Regenerate"
		switch {
		case err == nil:
			if plan.Status != domain.PlanDraft {
				return domain.Conflict("PLAN_NOT_EDITABLE", fmt.Sprintf(
					"Plan %s is %s; its dates are a commitment now. Reopen it before regenerating.",
					plan.PlanNo, plan.Status))
			}
			if updErr := s.repo.UpdateCalendar(ctx, tx, plan.ID, calendar, plan.Version); updErr != nil {
				return updErr
			}
		case domain.IsNotFound(err):
			action = "Generate"
			planNo, noErr := s.repo.NextPlanNo(ctx, tx, projection.ID)
			if noErr != nil {
				return noErr
			}
			newID, createErr := s.repo.CreatePlan(ctx, tx, projection.ID, planNo, calendar, in.Remark)
			if createErr != nil {
				return createErr
			}
			if plan, err = s.repo.GetPlan(ctx, tx, newID); err != nil {
				return err
			}
		default:
			return err
		}

		tasks, genErr := domain.GenerateTasks(lines, activities, calendar)
		if genErr != nil {
			// The engine only refuses input it cannot schedule — an activity with no daily capacity,
			// a block with no area. Both are data problems a planner can fix.
			return domain.Invalid([]domain.FieldError{{Field: "projectionId", Message: genErr.Error()}})
		}
		if err := s.repo.ReplaceTasks(ctx, tx, plan.ID, tasks); err != nil {
			return err
		}

		after, err := s.repo.GetPlan(ctx, tx, plan.ID)
		if err != nil {
			return err
		}
		saved = after
		return s.audit.Write(ctx, tx, u.Username, action, "ActivityPlan",
			fmt.Sprint(plan.ID), nil, after, remote)
	})
	if err != nil {
		return domain.ActivityPlan{}, err
	}
	return s.reload(ctx, saved.ID, u)
}

// ---------------------------------------------------------------- the lifecycle

// Release hands the programme to the field. Close finishes it; Reopen puts a closed plan back so it
// can be corrected. Only a draft may be regenerated, which is what makes releasing meaningful.
func (s *PlanService) Act(ctx context.Context, u domain.User, remote string, id int, action string, req domain.WorkflowRequest) (domain.ActivityPlan, error) {
	if err := requireRole(u, domain.RoleAdmin, domain.RoleManager); err != nil {
		return domain.ActivityPlan{}, err
	}

	var saved domain.ActivityPlan
	err := s.db.InTx(ctx, func(tx database.Querier) error {
		before, err := s.repo.GetPlan(ctx, tx, id)
		if err != nil {
			return err
		}

		to, err := nextPlanStatus(before, action)
		if err != nil {
			return err
		}
		if action == "Release" && before.TaskCount == 0 {
			return domain.Invalid([]domain.FieldError{{
				Field: "tasks", Message: "A plan with no tasks has nothing to release; generate it first."}})
		}

		rowVersion := req.Version
		if rowVersion == 0 {
			rowVersion = before.Version
		}
		if err := s.repo.SetStatus(ctx, tx, id, to, u.Username, rowVersion); err != nil {
			return err
		}

		after, err := s.repo.GetPlan(ctx, tx, id)
		if err != nil {
			return err
		}
		saved = after
		return s.audit.Write(ctx, tx, u.Username, action, "ActivityPlan", fmt.Sprint(id), before, after, remote)
	})
	if err != nil {
		return domain.ActivityPlan{}, err
	}
	return s.reload(ctx, saved.ID, u)
}

func nextPlanStatus(plan domain.ActivityPlan, action string) (string, error) {
	switch action {
	case "Release":
		if plan.Status != domain.PlanDraft {
			return "", domain.Conflict("INVALID_TRANSITION",
				"Only a draft plan can be released; this one is "+plan.Status+".")
		}
		return domain.PlanReleased, nil
	case "Close":
		if plan.Status != domain.PlanReleased {
			return "", domain.Conflict("INVALID_TRANSITION",
				"Only a released plan can be closed; this one is "+plan.Status+".")
		}
		return domain.PlanClosed, nil
	case "Reopen":
		if plan.Status == domain.PlanDraft {
			return "", domain.Conflict("INVALID_TRANSITION", "This plan is already a draft.")
		}
		return domain.PlanDraft, nil
	}
	return "", domain.BadRequest("UNKNOWN_ACTION", fmt.Sprintf("There is no plan action called %q.", action))
}

func (s *PlanService) Delete(ctx context.Context, u domain.User, remote string, id int) error {
	if err := requireRole(u, domain.RoleAdmin, domain.RoleManager); err != nil {
		return err
	}
	return s.db.InTx(ctx, func(tx database.Querier) error {
		before, err := s.repo.GetPlan(ctx, tx, id)
		if err != nil {
			return err
		}
		if before.Status != domain.PlanDraft {
			return domain.Conflict("PLAN_NOT_EDITABLE",
				"Only a draft plan can be deleted; this one is "+before.Status+".")
		}
		if err := s.repo.DeletePlan(ctx, tx, id); err != nil {
			return err
		}
		return s.audit.Write(ctx, tx, u.Username, "Delete", "ActivityPlan", fmt.Sprint(id), before, nil, remote)
	})
}

func (s *PlanService) reload(ctx context.Context, id int, u domain.User) (domain.ActivityPlan, error) {
	return s.Get(ctx, id, domain.Filter{}, u)
}
