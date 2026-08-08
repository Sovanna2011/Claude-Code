package service_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/kss/sugarplan/internal/auth"
	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
)

// A bulk write refused entirely on permission must say so.
//
// The daily rows go in as a batch and are checked one at a time, because a
// batch can legitimately mix planned and actual rows that need different
// permissions. That per-row reporting is right. What was wrong was the answer
// when nothing survived and every issue was a refusal: the caller got a
// validation error, which a client reads as "your payload is malformed". It was
// not. A screen keyed on 403 to hide its save button never hid it, and the user
// was left correcting fields that were already correct.

func actualCaneRow(h *harness, date domain.BusinessDate) domain.DailyCanePlan {
	return domain.DailyCanePlan{
		VersionID:     h.seeded.ActualID,
		FactoryID:     h.seeded.FactoryID,
		BusinessDate:  date,
		Series:        domain.SeriesActual,
		CaneAvailable: domain.D("18000"),
		CaneDelivered: domain.D("18000"),
		CaneAccepted:  domain.D("18000"),
		CaneCrushed:   domain.D("18000"),
		AvailableHrs:  domain.D("24"),
		CrushRateTPH:  domain.D("750"),
	}
}

func TestARefusedBatchIsForbiddenNotInvalid(t *testing.T) {
	h := newHarness(t, 14)
	row := actualCaneRow(h, "2026-12-20")

	// The supervisor may record the day, and the identical payload succeeds -
	// so nothing about the rows themselves is wrong.
	res, err := h.planning.UpsertCane(h.as(auth.RoleShiftSupervisor), h.seeded.ActualID,
		[]domain.DailyCanePlan{row}, service.UpsertOptions{})
	if err != nil {
		t.Fatalf("supervisor: %v", err)
	}
	if res.Accepted != 1 {
		t.Fatalf("supervisor accepted %d rows, want 1", res.Accepted)
	}

	// The executive may not, and must be told that rather than sent looking for
	// a bad field.
	_, err = h.planning.UpsertCane(h.as(auth.RoleExecutiveViewer), h.seeded.ActualID,
		[]domain.DailyCanePlan{row}, service.UpsertOptions{})
	if err == nil {
		t.Fatal("the executive recorded a day of crushing")
	}
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("error is %v, want a permission refusal", err)
	}
	var invalid *domain.ValidationError
	if errors.As(err, &invalid) {
		t.Errorf("a refusal was reported as a validation error: %v", err)
	}
}

func TestARealDataProblemIsStillAValidationError(t *testing.T) {
	h := newHarness(t, 14)
	row := actualCaneRow(h, "2026-12-21")
	row.AvailableHrs = domain.D("30") // there are 24 hours in a day

	_, err := h.planning.UpsertCane(h.as(auth.RoleShiftSupervisor), h.seeded.ActualID,
		[]domain.DailyCanePlan{row}, service.UpsertOptions{})
	if err == nil {
		t.Fatal("a 30-hour day was accepted")
	}
	var invalid *domain.ValidationError
	if !errors.As(err, &invalid) {
		t.Fatalf("error is %v, want a validation error", err)
	}
	if errors.Is(err, domain.ErrForbidden) {
		t.Error("a bad field was reported as a permission refusal")
	}
}

// The mixed case is the reason the per-row reporting exists, so it must not
// have been broken by making the all-refused case a refusal.
func TestAMixedBatchStillReportsEveryRow(t *testing.T) {
	h := newHarness(t, 14)
	good := actualCaneRow(h, "2026-12-22")
	bad := actualCaneRow(h, "2026-12-23")
	bad.CaneCrushed = domain.D("-1")

	_, err := h.planning.UpsertCane(h.as(auth.RoleShiftSupervisor), h.seeded.ActualID,
		[]domain.DailyCanePlan{good, bad}, service.UpsertOptions{})
	var invalid *domain.ValidationError
	if !errors.As(err, &invalid) {
		t.Fatalf("error is %v, want a validation error naming the bad row", err)
	}
	if len(invalid.Errors) != 1 || invalid.Errors[0].Row == nil || *invalid.Errors[0].Row != 1 {
		t.Errorf("issues = %+v, want one issue against row 1", invalid.Errors)
	}
}

// Regenerating a season is not the same right as editing a row in it.
//
// A shipment planner needs plan:write to enter planned shipment rows. That one
// permission also gated Generate, so the same account could replace every cane,
// production, storage and shipment row in the season - the production planner's
// whole plan - with a single request. Nothing warned anyone; the rows simply
// changed underneath them.
func TestOnlyThePlannerMayRegenerateTheSeason(t *testing.T) {
	h := newHarness(t, 0)

	if _, err := h.planning.Generate(h.as(auth.RoleShipmentPlanner), h.seeded.BudgetID,
		service.GenerateRequest{Replace: true}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("a shipment planner regenerated the season: err = %v", err)
	}
	if _, err := h.planning.GenerateSupplySchedule(h.as(auth.RoleShipmentPlanner),
		h.seeded.BudgetID); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("a shipment planner rebuilt the delivery schedule: err = %v", err)
	}

	// The shipment planner keeps what the role is for: writing planned shipment
	// rows. Splitting the permission must not have taken that away, or the fix
	// would have cost more than the problem.
	var product, channel string
	for _, id := range h.seeded.Products {
		product = id
		break
	}
	for _, id := range h.seeded.Channels {
		channel = id
		break
	}
	res, err := h.planning.UpsertShipments(h.as(auth.RoleShipmentPlanner), h.seeded.BudgetID,
		[]domain.DailyShipmentPlan{{
			VersionID: h.seeded.BudgetID, BusinessDate: "2027-01-15",
			Series: domain.SeriesPlan, ProductID: product, ChannelID: channel,
			Quantity: domain.D("500"),
		}}, service.UpsertOptions{})
	if err != nil {
		t.Fatalf("the shipment planner can no longer plan a shipment: %v", err)
	}
	if res.Accepted != 1 {
		t.Errorf("accepted %d shipment rows, want 1", res.Accepted)
	}

	if _, err := h.planning.Generate(h.as(auth.RoleProductionPlanner), h.seeded.BudgetID,
		service.GenerateRequest{Replace: true}); err != nil {
		t.Errorf("the production planner can no longer generate: %v", err)
	}
}

// A caller with no right to a plan is refused, not asked to refresh.
//
// Transition checked the row version before the permission, so an account that
// could never perform the action was told "version V3 is at row version 1" and
// invited to retry with a fresh copy. Retrying would never have worked, and the
// refusal handed out the plan's current row version on the way past. Order the
// checks the other way and both problems go.
func TestAnUnauthorisedTransitionIsRefusedBeforeTheRowVersionIsConsidered(t *testing.T) {
	h := newHarness(t, 0)

	// A deliberately stale row version, so the concurrency check would fire if
	// it were reached.
	req := service.TransitionRequest{Action: domain.ActionSubmit, RowVersion: 999999}

	_, err := h.planning.Transition(h.as(auth.RoleExecutiveViewer), h.seeded.BudgetID, req)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("executive got %v, want a permission refusal", err)
	}
	if errors.Is(err, domain.ErrConflict) {
		t.Errorf("an unauthorised caller was told the row version was stale: %v", err)
	}
	if err != nil && strings.Contains(err.Error(), "row version") {
		t.Errorf("the refusal leaks the plan's row version: %v", err)
	}

	// The planner may submit, so for them the stale row version is the real
	// answer and must still be reported.
	_, err = h.planning.Transition(h.as(auth.RoleProductionPlanner), h.seeded.BudgetID, req)
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("planner got %v, want the concurrency conflict", err)
	}
}
